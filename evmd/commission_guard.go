package evmd

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var fixedCommission = math.LegacyNewDecWithPrec(10, 2) // 0.10

// CommissionGuard embeds MultiStakingHooks so every other hook is a no-op.
type CommissionGuard struct {
	stakingtypes.MultiStakingHooks
	sk *stakingkeeper.Keeper
}

func (h CommissionGuard) AfterValidatorCreated(ctx context.Context, valAddr sdk.ValAddress) error {
	v, err := h.sk.GetValidator(ctx, valAddr)
	if err != nil {
		return err
	}
	c := v.Commission
	if !c.Rate.Equal(fixedCommission) || !c.MaxRate.Equal(fixedCommission) {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest,
			"commission rate and max rate must both be exactly 0.10")
	}
	return nil
}
