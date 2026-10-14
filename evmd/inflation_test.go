package evmd

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"
)

func TestStakeBasedInflation(t *testing.T) {
	p := minttypes.DefaultParams()
	bonded := math.LegacyNewDecWithPrec(6, 1) // 60% bonded
	p.InflationMax = math.LegacyNewDecWithPrec(24, 2)
	got := StakeBasedInflation(context.Background(), minttypes.Minter{}, p, bonded)
	require.True(t, got.Equal(math.LegacyNewDecWithPrec(144, 3))) // 0.144

	p.InflationMax = math.LegacyNewDecWithPrec(50, 2) // vote tries 50%/year
	got = StakeBasedInflation(context.Background(), minttypes.Minter{}, p, bonded)
	require.True(t, got.Equal(math.LegacyNewDecWithPrec(216, 3))) // capped: 0.36 × 0.6
}

func TestStakeBasedInflationFullBonded(t *testing.T) {
	p := minttypes.DefaultParams()
	p.InflationMax = math.LegacyNewDecWithPrec(24, 2)
	got := StakeBasedInflation(context.Background(), minttypes.Minter{}, p, math.LegacyOneDec())
	require.True(t, got.Equal(math.LegacyNewDecWithPrec(24, 2)))
}

func TestStakeBasedInflationZeroBonded(t *testing.T) {
	p := minttypes.DefaultParams()
	p.InflationMax = math.LegacyNewDecWithPrec(24, 2)
	got := StakeBasedInflation(context.Background(), minttypes.Minter{}, p, math.LegacyZeroDec())
	require.True(t, got.IsZero())
}

func TestStakeBasedInflationExactCap(t *testing.T) {
	p := minttypes.DefaultParams()
	p.InflationMax = math.LegacyNewDecWithPrec(36, 2) // exactly 3%/month
	bonded := math.LegacyNewDecWithPrec(5, 1)
	got := StakeBasedInflation(context.Background(), minttypes.Minter{}, p, bonded)
	require.True(t, got.Equal(math.LegacyNewDecWithPrec(18, 2))) // 0.36 * 0.5
}

func TestYearlyDecayStepMath(t *testing.T) {
	// Mirrors applyYearlyDecay: next = max(inflation_max - rate_change, inflation_min)
	rate := math.LegacyMustNewDecFromStr("0.24")
	step := math.LegacyMustNewDecFromStr("0.012")
	floor := math.LegacyMustNewDecFromStr("0.06")

	next := rate.Sub(step)
	require.True(t, next.Equal(math.LegacyMustNewDecFromStr("0.228")))

	next = next.Sub(step)
	require.True(t, next.Equal(math.LegacyMustNewDecFromStr("0.216")))

	// From mid-year gov set to 0.30, next yearly step is 0.288
	gov := math.LegacyMustNewDecFromStr("0.30")
	after := gov.Sub(step)
	require.True(t, after.Equal(math.LegacyMustNewDecFromStr("0.288")))

	// Floor clamp
	nearFloor := math.LegacyMustNewDecFromStr("0.065")
	clamped := nearFloor.Sub(step)
	if clamped.LT(floor) {
		clamped = floor
	}
	require.True(t, clamped.Equal(floor))

	// Step 0 pauses decay
	paused := rate.Sub(math.LegacyZeroDec())
	require.True(t, paused.Equal(rate))
}
