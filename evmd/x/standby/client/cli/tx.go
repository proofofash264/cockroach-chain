package cli

import (
	"encoding/hex"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/cosmos/evm/evmd/x/standby/types"
)

func NewTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Standby transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(NewHeartbeatCmd())
	return cmd
}

func NewHeartbeatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "heartbeat [operator] [validator] [height] [block-hash-hex] [signature-hex]",
		Short: "Submit a standby validator heartbeat",
		Args:  cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			height, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			hash, err := hex.DecodeString(args[3])
			if err != nil {
				return err
			}
			sig, err := hex.DecodeString(args[4])
			if err != nil {
				return err
			}
			msg := &types.MsgHeartbeat{
				Operator:         args[0],
				ValidatorAddress: args[1],
				Height:           height,
				BlockHash:        hash,
				Signature:        sig,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
