package types

import "cosmossdk.io/math"

const (
	ModuleName = "burn"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ParamsKey      = []byte{0x01}
	TotalBurnedKey = []byte{0x02}
)

func DefaultParams() Params {
	return Params{
		BurnFraction: math.LegacyMustNewDecFromStr("0.5"),
	}
}

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:      DefaultParams(),
		TotalBurned: math.ZeroInt(),
	}
}

func (p Params) Validate() error {
	if p.BurnFraction.IsNegative() || p.BurnFraction.GT(math.LegacyOneDec()) {
		return ErrInvalidBurnFraction
	}
	return nil
}
