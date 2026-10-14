package types

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestStandbyDefaultParams(t *testing.T) {
	p := DefaultParams()
	require.Equal(t, uint32(100), p.MaxStandby)
	require.Equal(t, uint64(100), p.HeartbeatWindow)
	require.True(t, p.MinSelfStake.Equal(math.NewInt(5_000).Mul(math.NewIntWithDecimal(1, 18))))
	require.NoError(t, p.Validate())
}

func TestStandbyParamsValidate(t *testing.T) {
	p := DefaultParams()
	p.MaxStandby = 0
	require.Error(t, p.Validate())

	p = DefaultParams()
	p.HeartbeatWindow = 0
	require.Error(t, p.Validate())

	p = DefaultParams()
	p.MinSelfStake = math.NewInt(-1)
	require.Error(t, p.Validate())
}

func TestHeartbeatSignBytes(t *testing.T) {
	hash := []byte{0xde, 0xad, 0xbe, 0xef}
	got := HeartbeatSignBytes("cockroach_9000-1", "cockroachvaloper1abc", 42, hash)
	require.Equal(t,
		"cockroach-heartbeat/v1|cockroach_9000-1|cockroachvaloper1abc|42|DEADBEEF",
		string(got),
	)
}

func TestStandbyDailyRewardMath(t *testing.T) {
	// PDF example: 50,000 tokens, rate 0.24, 24 hours online
	// reward = tokens * rate * hours / (24 * 365)
	tokens := math.NewInt(50_000)
	rate := math.LegacyMustNewDecFromStr("0.24")
	hours := int64(24)
	reward := rate.MulInt(tokens).MulInt64(hours).QuoInt64(24 * 365).TruncateInt()
	require.Equal(t, int64(32), reward.Int64()) // 32.876... truncates to 32

	// 18 of 24 hours = 75% of a full day
	hours = 18
	partial := rate.MulInt(tokens).MulInt64(hours).QuoInt64(24 * 365).TruncateInt()
	require.True(t, partial.LT(reward))
	require.Equal(t, int64(24), partial.Int64()) // 24.657... truncates to 24
}
