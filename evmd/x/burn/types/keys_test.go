package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestBurnParamsValidate(t *testing.T) {
	require.NoError(t, DefaultParams().Validate())

	p := DefaultParams()
	p.BurnFraction = math.LegacyZeroDec()
	require.NoError(t, p.Validate())

	p.BurnFraction = math.LegacyOneDec()
	require.NoError(t, p.Validate())

	p.BurnFraction = math.LegacyMustNewDecFromStr("-0.1")
	require.ErrorIs(t, p.Validate(), ErrInvalidBurnFraction)

	p.BurnFraction = math.LegacyMustNewDecFromStr("1.1")
	require.ErrorIs(t, p.Validate(), ErrInvalidBurnFraction)
}

func TestDefaultBurnFractionIsHalf(t *testing.T) {
	require.True(t, DefaultParams().BurnFraction.Equal(math.LegacyMustNewDecFromStr("0.5")))
}
