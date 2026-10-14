package types

import (
	"fmt"

	"cosmossdk.io/math"
)

const (
	ModuleName = "standby"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

var (
	ParamsKey       = []byte{0x01}
	RecentHashesKey = []byte{0x02}
	OnlineKey       = []byte{0x03}
	DayKey          = []byte{0x04}
	TotalPaidKey    = []byte{0x05}
)

func DefaultParams() Params {
	return Params{
		MaxStandby:      100,
		MinSelfStake:    math.NewInt(5_000).Mul(math.NewIntWithDecimal(1, 18)),
		HeartbeatWindow: 100,
	}
}

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		TotalPaid: math.ZeroInt(),
	}
}

func (p Params) Validate() error {
	if p.MaxStandby == 0 {
		return fmt.Errorf("max_standby must be > 0")
	}
	if p.MinSelfStake.IsNegative() {
		return fmt.Errorf("min_self_stake must be non-negative")
	}
	if p.HeartbeatWindow == 0 {
		return fmt.Errorf("heartbeat_window must be > 0")
	}
	return nil
}

func HeartbeatSignBytes(chainID, valoper string, height int64, hash []byte) []byte {
	return []byte(fmt.Sprintf("cockroach-heartbeat/v1|%s|%s|%d|%X", chainID, valoper, height, hash))
}
