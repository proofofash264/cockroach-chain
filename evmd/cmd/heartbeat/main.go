// Heartbeat sidecar for standby validators.
// Once an hour: read local block, sign with consensus key, broadcast MsgHeartbeat.
//
// Example:
//
//	heartbeat \
//	  --node http://127.0.0.1:26657 \
//	  --home $HOME/.cockroachd \
//	  --chain-id cockroach_9000-1 \
//	  --operator cockroach1... \
//	  --validator cockroachvaloper1... \
//	  --from operator-key \
//	  --keyring-backend test
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cometbft/cometbft/privval"

	standbytypes "github.com/cosmos/evm/evmd/x/standby/types"
)

func main() {
	node := flag.String("node", "http://127.0.0.1:26657", "CometBFT RPC")
	home := flag.String("home", os.ExpandEnv("$HOME/.cockroachd"), "node home")
	chainID := flag.String("chain-id", "", "chain id")
	operator := flag.String("operator", "", "operator account (cockroach1...)")
	validator := flag.String("validator", "", "validator address (cockroachvaloper1...)")
	from := flag.String("from", "", "key name used to pay gas for MsgHeartbeat")
	keyring := flag.String("keyring-backend", "test", "keyring backend")
	binary := flag.String("binary", "cockroachd", "chain binary")
	once := flag.Bool("once", false, "send one heartbeat and exit")
	flag.Parse()

	if *chainID == "" || *operator == "" || *validator == "" || *from == "" {
		fmt.Fprintln(os.Stderr, "required: --chain-id --operator --validator --from")
		os.Exit(1)
	}

	keyFile := filepath.Join(*home, "config", "priv_validator_key.json")
	stateFile := filepath.Join(*home, "data", "priv_validator_state.json")

	run := func() error {
		height, hash, err := fetchBlock(*node)
		if err != nil {
			return err
		}
		pv := privval.LoadFilePV(keyFile, stateFile)
		signBytes := standbytypes.HeartbeatSignBytes(*chainID, *validator, height, hash)
		sig, err := pv.Key.PrivKey.Sign(signBytes)
		if err != nil {
			return err
		}

		// Broadcast via CLI to avoid embedding full tx wiring in the sidecar.
		args := []string{
			"tx", "standby", "heartbeat",
			*operator, *validator,
			fmt.Sprintf("%d", height),
			hex.EncodeToString(hash),
			hex.EncodeToString(sig),
			"--from", *from,
			"--keyring-backend", *keyring,
			"--chain-id", *chainID,
			"--home", *home,
			"--node", *node,
			"--yes",
			"--fees", "200000acockroach",
		}
		cmd := exec.CommandContext(context.Background(), *binary, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	if *once {
		if err := run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	for {
		if err := run(); err != nil {
			fmt.Fprintln(os.Stderr, "heartbeat failed:", err)
		}
		time.Sleep(time.Hour)
	}
}

type blockResult struct {
	Result struct {
		Block struct {
			Header struct {
				Height string `json:"height"`
			} `json:"header"`
		} `json:"block"`
		BlockID struct {
			Hash string `json:"hash"`
		} `json:"block_id"`
	} `json:"result"`
}

func fetchBlock(rpc string) (int64, []byte, error) {
	resp, err := http.Get(rpc + "/block")
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	var br blockResult
	if err := json.Unmarshal(body, &br); err != nil {
		return 0, nil, err
	}
	var height int64
	if _, err := fmt.Sscanf(br.Result.Block.Header.Height, "%d", &height); err != nil {
		return 0, nil, err
	}
	hash, err := hex.DecodeString(br.Result.BlockID.Hash)
	if err != nil {
		return 0, nil, err
	}
	return height, hash, nil
}
