package evmd

import (
	"context"

	"cosmossdk.io/math"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

// maxStakeRate is 0.36 / year = 3% / month. Governance cannot pay above this.
var maxStakeRate = math.LegacyNewDecWithPrec(36, 2)

// StakeBasedInflation pays a fixed yearly rate on bonded tokens.
// params.InflationMax is the governance-controlled stake rate (0.24 = 2%/month).
func StakeBasedInflation(
	_ context.Context,
	_ minttypes.Minter,
	params minttypes.Params,
	bondedRatio math.LegacyDec,
) math.LegacyDec {
	rate := params.InflationMax
	if rate.GT(maxStakeRate) {
		rate = maxStakeRate
	}
	return rate.Mul(bondedRatio)
}
