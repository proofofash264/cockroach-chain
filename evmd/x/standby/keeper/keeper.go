package keeper

import (
	"bytes"
	"context"
	"errors"
	"math/bits"
	"sort"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cosmos/evm/evmd/x/standby/types"
)

var maxStakeRate = math.LegacyNewDecWithPrec(36, 2)

type Keeper struct {
	cdc       codec.BinaryCodec
	authority string
	sk        types.StakingKeeper
	bk        types.BankKeeper
	mk        *mintkeeper.Keeper

	Schema       collections.Schema
	Params       collections.Item[types.Params]
	RecentHashes collections.Map[int64, []byte]
	Online       collections.Map[string, uint32]
	Day          collections.Item[int64]
	TotalPaid    collections.Item[math.Int]
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	authority string,
	sk types.StakingKeeper,
	bk types.BankKeeper,
	mk *mintkeeper.Keeper,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:          cdc,
		authority:    authority,
		sk:           sk,
		bk:           bk,
		mk:           mk,
		Params:       collections.NewItem(sb, collections.NewPrefix(1), "params", codec.CollValue[types.Params](cdc)),
		RecentHashes: collections.NewMap(sb, collections.NewPrefix(2), "recent_hashes", collections.Int64Key, collections.BytesValue),
		Online:       collections.NewMap(sb, collections.NewPrefix(3), "online", collections.StringKey, collections.Uint32Value),
		Day:          collections.NewItem(sb, collections.NewPrefix(4), "day", collections.Int64Value),
		TotalPaid:    collections.NewItem(sb, collections.NewPrefix(5), "total_paid", sdk.IntValue),
	}
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

func (k Keeper) BeginBlocker(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	h := sdkCtx.BlockHeight()
	p, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}
	if err := k.RecentHashes.Set(ctx, h, sdkCtx.HeaderHash()); err != nil {
		return err
	}
	_ = k.RecentHashes.Remove(ctx, h-int64(p.HeartbeatWindow)-1)

	day := sdkCtx.BlockTime().Unix() / 86400
	last, err := k.Day.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return k.Day.Set(ctx, day)
	} else if err != nil {
		return err
	}
	if day == last {
		return nil
	}
	if err := k.payDay(ctx, p); err != nil {
		return err
	}
	if err := k.Online.Clear(ctx, nil); err != nil {
		return err
	}
	return k.Day.Set(ctx, day)
}

func (k Keeper) payDay(ctx context.Context, p types.Params) error {
	mp, err := k.mk.Params.Get(ctx)
	if err != nil {
		return err
	}
	rate := mp.InflationMax
	if rate.GT(maxStakeRate) {
		rate = maxStakeRate
	}

	var set []stakingtypes.Validator
	err = k.Online.Walk(ctx, nil, func(valStr string, _ uint32) (bool, error) {
		valAddr, err := sdk.ValAddressFromBech32(valStr)
		if err != nil {
			return false, nil
		}
		v, err := k.sk.GetValidator(ctx, valAddr)
		if err != nil || v.IsJailed() || v.IsBonded() {
			return false, nil
		}
		self, err := k.sk.GetDelegation(ctx, sdk.AccAddress(valAddr), valAddr)
		if err != nil || v.TokensFromShares(self.Shares).TruncateInt().LT(p.MinSelfStake) {
			return false, nil
		}
		set = append(set, v)
		return false, nil
	})
	if err != nil {
		return err
	}
	sort.Slice(set, func(i, j int) bool { return set[i].Tokens.GT(set[j].Tokens) })
	if len(set) > int(p.MaxStandby) {
		set = set[:p.MaxStandby]
	}

	for _, v := range set {
		mask, _ := k.Online.Get(ctx, v.OperatorAddress)
		hours := int64(bits.OnesCount32(mask))
		reward := rate.MulInt(v.Tokens).MulInt64(hours).QuoInt64(24 * 365).TruncateInt()
		if !reward.IsPositive() {
			continue
		}
		coin := sdk.NewCoin(mp.MintDenom, reward)
		if err := k.bk.MintCoins(ctx, types.ModuleName, sdk.NewCoins(coin)); err != nil {
			return err
		}
		if err := k.distribute(ctx, v, coin); err != nil {
			return err
		}
		paid, err := k.TotalPaid.Get(ctx)
		if errors.Is(err, collections.ErrNotFound) {
			paid = math.ZeroInt()
		} else if err != nil {
			return err
		}
		_ = k.TotalPaid.Set(ctx, paid.Add(reward))
	}
	return nil
}

func (k Keeper) distribute(ctx context.Context, v stakingtypes.Validator, coin sdk.Coin) error {
	valAddr, _ := sdk.ValAddressFromBech32(v.OperatorAddress)
	commission := v.Commission.Rate.MulInt(coin.Amount).TruncateInt()
	pool := coin.Amount.Sub(commission)
	paid := math.ZeroInt()
	dels, err := k.sk.GetValidatorDelegations(ctx, valAddr)
	if err != nil {
		return err
	}
	for _, d := range dels {
		amt := d.Shares.MulInt(pool).Quo(v.DelegatorShares).TruncateInt()
		if !amt.IsPositive() {
			continue
		}
		to, err := sdk.AccAddressFromBech32(d.DelegatorAddress)
		if err != nil {
			return err
		}
		if err := k.bk.SendCoinsFromModuleToAccount(ctx, types.ModuleName, to,
			sdk.NewCoins(sdk.NewCoin(coin.Denom, amt))); err != nil {
			return err
		}
		paid = paid.Add(amt)
	}
	rest := coin.Amount.Sub(paid)
	return k.bk.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sdk.AccAddress(valAddr),
		sdk.NewCoins(sdk.NewCoin(coin.Denom, rest)))
}

func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) {
	if err := k.Params.Set(ctx, gs.Params); err != nil {
		panic(err)
	}
	if err := k.TotalPaid.Set(ctx, gs.TotalPaid); err != nil {
		panic(err)
	}
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	p, err := k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}
	total, err := k.TotalPaid.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		total = math.ZeroInt()
	} else if err != nil {
		panic(err)
	}
	return &types.GenesisState{Params: p, TotalPaid: total}
}

type msgServer struct{ Keeper }

func NewMsgServerImpl(k Keeper) types.MsgServer { return &msgServer{k} }

func (m msgServer) Heartbeat(ctx context.Context, msg *types.MsgHeartbeat) (*types.MsgHeartbeatResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	p, err := m.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return nil, err
	}
	operator, err := sdk.AccAddressFromBech32(msg.Operator)
	if err != nil || !bytes.Equal(operator, valAddr.Bytes()) {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only the validator operator can send a heartbeat")
	}
	v, err := m.sk.GetValidator(ctx, valAddr)
	if err != nil {
		return nil, err
	}
	if v.IsJailed() {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "validator is jailed")
	}
	if v.IsBonded() {
		return &types.MsgHeartbeatResponse{}, nil
	}
	cur := sdkCtx.BlockHeight()
	if msg.Height > cur || cur-msg.Height > int64(p.HeartbeatWindow) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "heartbeat block is too old")
	}
	hash, err := m.RecentHashes.Get(ctx, msg.Height)
	if err != nil || !bytes.Equal(hash, msg.BlockHash) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "block hash does not match")
	}
	pk, err := v.ConsPubKey()
	if err != nil {
		return nil, err
	}
	sb := types.HeartbeatSignBytes(sdkCtx.ChainID(), msg.ValidatorAddress, msg.Height, msg.BlockHash)
	if !pk.VerifySignature(sb, msg.Signature) {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "bad consensus-key signature")
	}
	hour := uint32(sdkCtx.BlockTime().UTC().Hour()) //nolint:gosec
	mask, err := m.Online.Get(ctx, msg.ValidatorAddress)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, err
	}
	return &types.MsgHeartbeatResponse{}, m.Online.Set(ctx, msg.ValidatorAddress, mask|(1<<hour))
}

func (m msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if msg.Authority != m.authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "expected %s", m.authority)
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}
	if err := m.Params.Set(ctx, msg.Params); err != nil {
		return nil, err
	}
	return &types.MsgUpdateParamsResponse{}, nil
}

type queryServer struct{ Keeper }

func NewQueryServer(k Keeper) types.QueryServer { return queryServer{k} }

func (q queryServer) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	p, err := q.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &types.QueryParamsResponse{Params: p}, nil
}

var _ = minttypes.ModuleName
