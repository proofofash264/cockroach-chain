package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrInvalidBurnFraction = errorsmod.Register(ModuleName, 2, "burn_fraction must be between 0 and 1")
	ErrInvalidAuthority    = errorsmod.Register(ModuleName, 3, "invalid authority")
)
