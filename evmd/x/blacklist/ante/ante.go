package ante

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/cosmos/evm/evmd/x/blacklist/keeper"
)

type FreezeDecorator struct {
	K keeper.Keeper
}

func (d FreezeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, sim bool, next sdk.AnteHandler) (sdk.Context, error) {
	for _, msg := range tx.GetMsgs() {
		if ethMsg, ok := msg.(*evmtypes.MsgEthereumTx); ok {
			sender := ethMsg.GetFrom()
			if d.K.HasFrozen(ctx, sender) {
				return ctx, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "sender address is frozen")
			}
		}
	}
	if sigTx, ok := tx.(authsigning.SigVerifiableTx); ok {
		signers, err := sigTx.GetSigners()
		if err != nil {
			return ctx, err
		}
		for _, s := range signers {
			if d.K.HasFrozen(ctx, sdk.AccAddress(s)) {
				return ctx, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "signer address is frozen")
			}
		}
	}
	return next(ctx, tx, sim)
}

func SendRestriction(k keeper.Keeper) func(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) (sdk.AccAddress, error) {
	return func(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) (sdk.AccAddress, error) {
		if k.HasFrozen(ctx, from) {
			return to, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "sender address is frozen")
		}
		return to, nil
	}
}
