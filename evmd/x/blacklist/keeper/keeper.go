package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/core/store"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/cosmos/evm/evmd/x/blacklist/types"
)

type Keeper struct {
	authority string
	Frozen    collections.Map[sdk.AccAddress, string]
	Schema    collections.Schema
}

func NewKeeper(storeService storetypes.KVStoreService, authority string) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		authority: authority,
		Frozen:    collections.NewMap(sb, collections.NewPrefix(0), "frozen", sdk.AccAddressKey, collections.StringValue),
	}
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

func (k Keeper) HasFrozen(ctx context.Context, addr sdk.AccAddress) bool {
	ok, err := k.Frozen.Has(ctx, addr)
	return err == nil && ok
}

func (k Keeper) InitGenesis(ctx context.Context, gs *types.GenesisState) {
	for _, e := range gs.Entries {
		addr, err := types.ParseAddress(e.Address)
		if err != nil {
			panic(err)
		}
		if err := k.Frozen.Set(ctx, addr, e.Reason); err != nil {
			panic(err)
		}
	}
}

func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	gs := &types.GenesisState{}
	err := k.Frozen.Walk(ctx, nil, func(addr sdk.AccAddress, reason string) (bool, error) {
		gs.Entries = append(gs.Entries, &types.FrozenEntry{Address: addr.String(), Reason: reason})
		return false, nil
	})
	if err != nil {
		panic(err)
	}
	return gs
}

type msgServer struct{ Keeper }

func NewMsgServerImpl(k Keeper) types.MsgServer { return &msgServer{k} }

func (m msgServer) FreezeAddress(ctx context.Context, msg *types.MsgFreezeAddress) (*types.MsgFreezeAddressResponse, error) {
	if msg.Authority != m.authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "expected %s", m.authority)
	}
	addr, err := types.ParseAddress(msg.Address)
	if err != nil {
		return nil, err
	}
	if err := m.Frozen.Set(ctx, addr, msg.Reason); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("address_frozen", sdk.NewAttribute("address", addr.String())))
	return &types.MsgFreezeAddressResponse{}, nil
}

func (m msgServer) UnfreezeAddress(ctx context.Context, msg *types.MsgUnfreezeAddress) (*types.MsgUnfreezeAddressResponse, error) {
	if msg.Authority != m.authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "expected %s", m.authority)
	}
	addr, err := types.ParseAddress(msg.Address)
	if err != nil {
		return nil, err
	}
	if err := m.Frozen.Remove(ctx, addr); err != nil {
		return nil, err
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("address_unfrozen", sdk.NewAttribute("address", addr.String())))
	return &types.MsgUnfreezeAddressResponse{}, nil
}

func (k Keeper) IsFrozenQuery(ctx context.Context, req *types.QueryIsFrozenRequest) (*types.QueryIsFrozenResponse, error) {
	addr, err := types.ParseAddress(req.Address)
	if err != nil {
		return nil, err
	}
	reason, err := k.Frozen.Get(ctx, addr)
	if errors.Is(err, collections.ErrNotFound) {
		return &types.QueryIsFrozenResponse{Frozen: false}, nil
	}
	if err != nil {
		return nil, err
	}
	return &types.QueryIsFrozenResponse{Frozen: true, Reason: reason}, nil
}

func (k Keeper) FrozenAddresses(ctx context.Context, _ *types.QueryFrozenAddressesRequest) (*types.QueryFrozenAddressesResponse, error) {
	var entries []*types.FrozenEntry
	err := k.Frozen.Walk(ctx, nil, func(addr sdk.AccAddress, reason string) (bool, error) {
		entries = append(entries, &types.FrozenEntry{Address: addr.String(), Reason: reason})
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryFrozenAddressesResponse{Entries: entries}, nil
}

type queryServer struct{ Keeper }

func NewQueryServer(k Keeper) types.QueryServer { return queryServer{k} }

func (q queryServer) IsFrozen(ctx context.Context, req *types.QueryIsFrozenRequest) (*types.QueryIsFrozenResponse, error) {
	return q.IsFrozenQuery(ctx, req)
}

func (q queryServer) FrozenAddresses(ctx context.Context, req *types.QueryFrozenAddressesRequest) (*types.QueryFrozenAddressesResponse, error) {
	return q.Keeper.FrozenAddresses(ctx, req)
}

var _ = sdkerrors.ErrUnauthorized
