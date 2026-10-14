package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/evm/evmd/x/burn/types"
)

type Keeper struct {
	cdc       codec.BinaryCodec
	authority string
	ak        types.AccountKeeper
	bk        types.BankKeeper
	sk        types.StakingKeeper

	Schema     collections.Schema
	Params     collections.Item[types.Params]
	TotalBurned collections.Item[math.Int]
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	authority string,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	sk types.StakingKeeper,
) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		cdc:         cdc,
		authority:   authority,
		ak:          ak,
		bk:          bk,
		sk:          sk,
		Params:      collections.NewItem(sb, collections.NewPrefix(1), "params", codec.CollValue[types.Params](cdc)),
		TotalBurned: collections.NewItem(sb, collections.NewPrefix(2), "total_burned", sdk.IntValue),
	}
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

func (k Keeper) GetAuthority() string { return k.authority }

func (k Keeper) BurnFees(ctx context.Context) error {
	p, err := k.Params.Get(ctx)
	if err != nil || !p.BurnFraction.IsPositive() {
		return err
	}
	denom, err := k.sk.BondDenom(ctx)
	if err != nil {
		return err
	}
	fees := k.bk.GetBalance(ctx, k.ak.GetModuleAddress(authtypes.FeeCollectorName), denom)
	amt := p.BurnFraction.MulInt(fees.Amount).TruncateInt()
	if !amt.IsPositive() {
		return nil
	}
	coin := sdk.NewCoin(denom, amt)
	if err := k.bk.SendCoinsFromModuleToModule(ctx, authtypes.FeeCollectorName, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		return err
	}
	return k.burn(ctx, coin, "fees")
}

func (k Keeper) BurnAddressBalance(ctx context.Context) error {
	denom, err := k.sk.BondDenom(ctx)
	if err != nil {
		return err
	}
	bal := k.bk.GetBalance(ctx, k.ak.GetModuleAddress(types.ModuleName), denom)
	if !bal.Amount.IsPositive() {
		return nil
	}
	return k.burn(ctx, bal, "burn_address")
}

func (k Keeper) burn(ctx context.Context, coin sdk.Coin, source string) error {
	if err := k.bk.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(coin)); err != nil {
		return err
	}
	total, err := k.TotalBurned.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		total = math.ZeroInt()
	} else if err != nil {
		return err
	}
	if err := k.TotalBurned.Set(ctx, total.Add(coin.Amount)); err != nil {
		return err
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(sdk.NewEvent("burn",
		sdk.NewAttribute("amount", coin.String()),
		sdk.NewAttribute("source", source)))
	return nil
}

func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) {
	if err := k.Params.Set(ctx, gs.Params); err != nil {
		panic(err)
	}
	if err := k.TotalBurned.Set(ctx, gs.TotalBurned); err != nil {
		panic(err)
	}
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	p, err := k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}
	total, err := k.TotalBurned.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		total = math.ZeroInt()
	} else if err != nil {
		panic(err)
	}
	return &types.GenesisState{Params: p, TotalBurned: total}
}

type msgServer struct{ Keeper }

func NewMsgServerImpl(k Keeper) types.MsgServer { return &msgServer{k} }

func (m msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if msg.Authority != m.authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "expected %s got %s", m.authority, msg.Authority)
	}
	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}
	if err := m.Params.Set(ctx, msg.Params); err != nil {
		return nil, err
	}
	return &types.MsgUpdateParamsResponse{}, nil
}

func (k Keeper) ParamsQuery(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	p, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &types.QueryParamsResponse{Params: p}, nil
}

func (k Keeper) TotalBurnedQuery(ctx context.Context, _ *types.QueryTotalBurnedRequest) (*types.QueryTotalBurnedResponse, error) {
	total, err := k.TotalBurned.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		total = math.ZeroInt()
	} else if err != nil {
		return nil, err
	}
	return &types.QueryTotalBurnedResponse{TotalBurned: total}, nil
}

type queryServer struct{ Keeper }

func NewQueryServer(k Keeper) types.QueryServer { return queryServer{k} }

func (q queryServer) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return q.ParamsQuery(ctx, req)
}

func (q queryServer) TotalBurned(ctx context.Context, req *types.QueryTotalBurnedRequest) (*types.QueryTotalBurnedResponse, error) {
	return q.TotalBurnedQuery(ctx, req)
}
