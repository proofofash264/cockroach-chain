# Cockroach Chain — manual local setup

Milestone C is in this tree: branding, stake-based mint, yearly decay, 10% commission
guard, `x/burn`, `x/blacklist`, `x/standby`. BABYROACH is deferred.

Detailed test cases and how to run them: **[TEST.md](TEST.md)**.

| Item | Value |
| --- | --- |
| Binary | `cockroachd` (built from `evmd/cmd/evmd`) |
| Default home | `~/.cockroachd` |
| Bech32 prefix | `cockroach` → accounts `cockroach1...`, validators `cockroachvaloper1...` |
| Base denom | `acockroach` (18 decimals), display `COCKROACH` |
| Local chain-id example | `cockroach_9000-1` (EVM chain ID `9000` is a placeholder only) |

---

## 1. Build the binary

From the repo root:

```text
make build
```

You should get `./build/cockroachd`.

Or from `evmd/`:

```text
go build -o ../build/cockroachd ./cmd/evmd
```

Confirm:

```text
./build/cockroachd version
```

---

## 2. Initialize the node home

Pick a home directory (example: `~/.cockroachd`) and a chain-id
(example: `cockroach_9000-1`).

If you already have an old home from testing, delete that folder first so genesis is clean.

```text
./build/cockroachd init <moniker> --chain-id cockroach_9000-1 --home ~/.cockroachd
```

This creates:

- `~/.cockroachd/config/genesis.json`
- `~/.cockroachd/config/app.toml`
- `~/.cockroachd/config/config.toml`
- validator key material under `config/` and `data/`

---

## 3. Edit `genesis.json` by hand

Open:

```text
~/.cockroachd/config/genesis.json
```

Set the fields below. Names follow SDK v0.54 / Cosmos EVM. If a path is missing or
named differently, start a throwaway node once and run
`cockroachd q <module> params` to see the live shape, then match that in genesis.

### 3.1 Denom (must be consistent everywhere)

**Bank metadata** — set `app_state.bank.denom_metadata` to one entry:

- `base`: `acockroach`
- `display` / `name` / `symbol`: `COCKROACH`
- `denom_units`:
  - `{ "denom": "acockroach", "exponent": 0 }`
  - `{ "denom": "COCKROACH", "exponent": 18 }`

Also set:

- `app_state.staking.params.bond_denom` → `acockroach`
- `app_state.mint.params.mint_denom` → `acockroach`
- `app_state.evm.params.evm_denom` → `acockroach`
- `app_state.gov.params.min_deposit[0].denom` → `acockroach`
- `app_state.gov.params.expedited_min_deposit[0].denom` → `acockroach`

Leave deposit **amounts** as placeholders for now; change them when you know a USD target.

### 3.2 Staking rewards (mint)

Under `app_state.mint`:

- `minter.inflation` → `"0.000000000000000000"`
- `params.inflation_max` → `"0.240000000000000000"` (2% per month at launch)
- `params.inflation_rate_change` → `"0.012000000000000000"` (yearly step down)
- `params.inflation_min` → `"0.060000000000000000"` (floor 0.5% per month)
- `params.blocks_per_year` → `"6307200"` (5 second blocks)

Under `app_state.distribution.params`:

- `community_tax` → `"0.000000000000000000"`

Under `app_state.staking.params`:

- `min_commission_rate` → `"0.100000000000000000"`
- `unbonding_time` → `"604800s"` (7 days)
- `max_validators` → `100`

### 3.3 Burn module

Under `app_state.burn` (module is registered in this binary):

- `params.burn_fraction` → `"0.500000000000000000"`
- `total_burned` → `"0"`

Under `app_state.gov.params`:

- `burn_vote_veto` → `true`

### 3.4 Standby module

Under `app_state.standby`:

- `params.max_standby` → `100`
- `params.min_self_stake` → `"5000000000000000000000"` (5,000 × 10^18)
- `params.heartbeat_window` → `"100"`
- `total_paid` → `"0"`

### 3.5 Blacklist module

Under `app_state.blacklist`:

- `entries` → `[]` (empty at genesis)

### 3.6 Slashing and evidence

Under `app_state.slashing.params`:

- `signed_blocks_window` → `"10000"`
- `min_signed_per_window` → `"0.500000000000000000"`
- `downtime_jail_duration` → `"600s"`
- `slash_fraction_downtime` → `"0.000100000000000000"`
- `slash_fraction_double_sign` → `"0.050000000000000000"`

Under consensus evidence (often `consensus_params.evidence` in this stack):

- `max_age_duration` → `"604800s"`
- `max_age_num_blocks` → `"120960"`

### 3.7 Governance timing

Under `app_state.gov.params`:

- `voting_period` → `"259200s"` (3 days)
- `expedited_voting_period` → `"86400s"` (1 day)
- `quorum` → `"0.334000000000000000"`
- `threshold` → `"0.500000000000000000"`

### 3.8 Fee market (EIP-1559)

Under `app_state.feemarket.params`:

- `no_base_fee` → `false`
- `base_fee` → `"10000000000"`
- `min_gas_price` → `"10000000000"`
- `min_gas_multiplier` → `"0.500000000000000000"`

Under consensus block params:

- `max_gas` → `"40000000"` (must be positive for EIP-1559)

Save the file.

Validate:

```text
./build/cockroachd genesis validate-genesis --home ~/.cockroachd
```

If validate complains about an unknown field, open the error, compare with
`cockroachd q <module> params` on a running local node, and fix that path only.

---

## 4. Edit `app.toml` by hand

Open:

```text
~/.cockroachd/config/app.toml
```

Set at least:

- `minimum-gas-prices = "0acockroach"`  
  (on-chain floor is the fee market; this avoids a node halt on empty min gas)

Under `[json-rpc]`:

- `enable = true`
- HTTP listen (default `0.0.0.0:8545` is fine for local)
- WebSocket if you need it (default often `8546`)

Under EVM settings (name may be `[evm]` depending on template):

- set `evm-chain-id` to the numeric part of your chain-id (example `9000` for
  `cockroach_9000-1`)

Under `[api]` / `[grpc]` enable REST and gRPC if you want Keplr / explorers later
(defaults ports: REST `1317`, gRPC `9090`).

Save the file.

---

## 5. Edit `config.toml` by hand (optional for single node)

Open:

```text
~/.cockroachd/config/config.toml
```

For a single local node you can leave most defaults. Common checks:

- `[rpc] laddr` → CometBFT RPC (default `tcp://127.0.0.1:26657`)
- P2P settings if you later join multiple nodes

Save the file.

---

## 6. Create keys and fund genesis accounts

Create keys (example: `test` keyring for local only):

```text
./build/cockroachd keys add alice --keyring-backend test --home ~/.cockroachd
./build/cockroachd keys add bob --keyring-backend test --home ~/.cockroachd
```

Show an address:

```text
./build/cockroachd keys show alice -a --keyring-backend test --home ~/.cockroachd
```

Add a liquid genesis balance (amounts are in `acockroach`, 18 decimals). Example:
1,000,000 COCKROACH = `1000000000000000000000000acockroach`.

```text
./build/cockroachd genesis add-genesis-account <alice-address> \
  1000000000000000000000000acockroach \
  --home ~/.cockroachd
```

Repeat for other accounts.

### Vesting buckets (when you need them)

Same command, with vesting flags (no contract):

```text
./build/cockroachd genesis add-genesis-account <address> <total>acockroach \
  --vesting-amount <vesting>acockroach \
  --vesting-end-time <unix-end> \
  --vesting-start-time <unix-start> \
  --home ~/.cockroachd
```

Use cliff schedules by omitting start or setting start = cliff unlock, per the SDK
vesting account rules you choose for team / private sale / ecosystem.

---

## 7. Create the genesis validator (commission must be exactly 10%)

Self-bond some of alice’s tokens. **Both** rate and max rate must be `0.10` or the
chain rejects the gentx / create-validator.

```text
./build/cockroachd genesis gentx alice 100000000000000000000000acockroach \
  --chain-id cockroach_9000-1 \
  --commission-rate 0.10 \
  --commission-max-rate 0.10 \
  --commission-max-change-rate 0.00 \
  --keyring-backend test \
  --home ~/.cockroachd
```

Collect and validate:

```text
./build/cockroachd genesis collect-gentxs --home ~/.cockroachd
./build/cockroachd genesis validate-genesis --home ~/.cockroachd
```

---

## 8. Start the node

```text
./build/cockroachd start --home ~/.cockroachd
```

Leave this running. In another terminal, use the same `--home` and `--chain-id` for
queries and txs.

---

## 9. Manual smoke checks

### Params

```text
./build/cockroachd q mint params --home ~/.cockroachd
./build/cockroachd q burn params --home ~/.cockroachd
./build/cockroachd q standby params --home ~/.cockroachd
./build/cockroachd q staking params --home ~/.cockroachd
./build/cockroachd q distribution params --home ~/.cockroachd
```

Expect mint `inflation_max` ≈ `0.24`, burn `burn_fraction` ≈ `0.5`,
distribution `community_tax` ≈ `0`, staking `min_commission_rate` ≈ `0.1`.

### Delegate and rewards

Find your validator operator address (`cockroachvaloper1...`), then:

```text
./build/cockroachd tx staking delegate <valoper> 1000000000000000000000acockroach \
  --from alice --keyring-backend test --home ~/.cockroachd \
  --chain-id cockroach_9000-1 -y
```

Wait several blocks, then:

```text
./build/cockroachd q distribution rewards <alice-address> --home ~/.cockroachd
```

Rough check: for 1,000 COCKROACH delegated at rate `0.24`, after `N` blocks the
delegator’s claimable share is about  
`1000 × 0.24 × 0.90 × N / 6307200` COCKROACH (validator keeps 10% commission).

### Commission guard

Try creating a validator with `--commission-rate 0.05` or `0.15`, or rate `0.10` with
max `0.15`. Those should fail. Only `0.10` / `0.10` should succeed.

### Burn address

```text
./build/cockroachd q auth module-account burn --home ~/.cockroachd
```

Note the `cockroach1...` address. Send `acockroach` to it from alice (CLI or MetaMask
using the `0x` form from `debug addr`). After the next block:

```text
./build/cockroachd q burn total-burned --home ~/.cockroachd
```

Total burned should rise; supply should fall by that amount.

### Address freeze

1. Write a gov proposal JSON whose message `@type` is  
   `/cockroach.blacklist.v1.MsgFreezeAddress`, authority = gov module account,  
   `address` = either `0x...` or `cockroach1...`, plus a `reason`.
2. Submit and vote until it passes (use expedited for a short local voting period if
   you lowered those times in genesis for testing).
3. Confirm that address cannot send (Cosmos or EVM) but can still receive.
4. Unfreeze with `/cockroach.blacklist.v1.MsgUnfreezeAddress` the same way.

### Yearly decay (local only)

On a **throwaway** home, set `app_state.mint.params.blocks_per_year` to `"100"` before
start. After height 100, `inflation_max` should move `0.24` → `0.228`; at 200 →
`0.216`; it should stop at `0.06`.

---

## 10. Standby heartbeat (manual)

1. Build the sidecar:
   ```text
   cd evmd
   go build -o ../build/heartbeat ./cmd/heartbeat
   ```
2. Run a validator that is **not** in the bonded set, with self-stake ≥ 5,000 COCKROACH,
   not jailed.
3. Once per hour (or `--once` for a single try), run the sidecar against the local RPC.
   It reads the latest block, signs with the consensus key under
   `config/priv_validator_key.json`, and broadcasts `MsgHeartbeat` via:
   ```text
   cockroachd tx standby heartbeat ...
   ```
4. After a UTC day boundary, qualifying standby validators should receive liquid
   `acockroach` (not distribution rewards).

---

## 11. Before any public testnet

Walk the PDF checklist on a local multi-node net by hand:

- Reward math for 1,000 COCKROACH delegated  
- Commission fixed at 10% (CLI and staking precompile)  
- Decay steps and gov changes to `inflation_max` / `inflation_rate_change`  
- Fee burn 50/50 and burn-address burns  
- Freeze / unfreeze for both address forms  
- Standby payouts for 24h / 18h / low self-stake / jailed cases  

Mainnet genesis must be generated fresh (never reuse testnet state). Pick a real unused
EVM chain ID from chainlist.org before that genesis.

---

## Open decisions (not code)

- Final EVM chain ID  
- `gov` min deposit amounts in `acockroach`  
- Gas prices once token market value is known  
- Public RPC / explorer domains  
