package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/evmd/config"
)

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	config.SetBech32Prefixes(cfg)
	m.Run()
}

func TestParseAddressHexAndBech32SameAccount(t *testing.T) {
	hexAddr := "0x00000000000000000000000000000000000000Ab"
	fromHex, err := ParseAddress(hexAddr)
	require.NoError(t, err)

	bech32 := sdk.AccAddress(common.HexToAddress(hexAddr).Bytes()).String()
	require.True(t, len(bech32) > 0)
	require.Contains(t, bech32, "cockroach")

	fromBech32, err := ParseAddress(bech32)
	require.NoError(t, err)
	require.True(t, fromHex.Equals(fromBech32))
}

func TestParseAddressInvalid(t *testing.T) {
	_, err := ParseAddress("0xnotanaddress")
	require.Error(t, err)

	_, err = ParseAddress("cosmos1invalid")
	require.Error(t, err)
}
