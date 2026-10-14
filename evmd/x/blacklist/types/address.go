package types

import (
	"strings"

	errorsmod "cosmossdk.io/errors"
	"github.com/ethereum/go-ethereum/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func ParseAddress(s string) (sdk.AccAddress, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		if !common.IsHexAddress(s) {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, s)
		}
		return sdk.AccAddress(common.HexToAddress(s).Bytes()), nil
	}
	return sdk.AccAddressFromBech32(s)
}
