package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgFreezeAddress{},
		&MsgUnfreezeAddress{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgFreezeAddress{}, "cockroach/x/blacklist/MsgFreezeAddress", nil)
	cdc.RegisterConcrete(&MsgUnfreezeAddress{}, "cockroach/x/blacklist/MsgUnfreezeAddress", nil)
}
