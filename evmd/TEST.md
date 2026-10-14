# Cockroach Chain — test cases and how to run them

This covers every Milestone C change: stake-based inflation, yearly decay, commission
guard, `x/burn`, `x/blacklist`, `x/standby` (+ heartbeat), and branding/genesis params.

There are two layers:

1. **Automated unit tests** — run with `go test` (no node required).
2. **Manual / local-node cases** — run against a live `cockroachd` (see [SETUP.md](SETUP.md)).

---

## How to run automated tests

From the **repo root**:

```text
make test-evmd
```

That runs all `evmd` package tests with `-race` and `-tags=test`.

From **`evmd/`** (faster, targeted):

```text
# All Cockroach custom unit tests in one go
go test -count=1 ./inflation_test.go ./inflation.go \
  ./x/burn/types/ ./x/blacklist/types/ ./x/standby/types/

# Or package by package:
go test -count=1 -v .
go test -count=1 -v ./x/burn/types/
go test -count=1 -v ./x/blacklist/types/
go test -count=1 -v ./x/standby/types/
```

Verbose single function:

```text
cd evmd
go test -count=1 -v -run TestStakeBasedInflation .
go test -count=1 -v -run TestParseAddressHexAndBech32SameAccount ./x/blacklist/types/
```

Expect: `PASS` / `ok` with no failures.

---

## A. Stake-based inflation (`evmd/inflation.go`)

| ID | Case | Expected |
| --- | --- | --- |
| A1 | Bonded ratio 60%, `inflation_max` 0.24 | Chain inflation = **0.144** |
| A2 | Bonded ratio 60%, `inflation_max` 0.50 (above hard cap) | Still **0.216** (= 0.36 × 0.6); never pays above 3%/month |
| A3 | Bonded ratio 100%, rate 0.24 | Inflation = **0.24** |
| A4 | Bonded ratio 0%, rate 0.24 | Inflation = **0** |
| A5 | Bonded ratio 50%, rate exactly 0.36 | Inflation = **0.18** |

**Automated:** `TestStakeBasedInflation`, `TestStakeBasedInflationFullBonded`,
`TestStakeBasedInflationZeroBonded`, `TestStakeBasedInflationExactCap`

**Manual (local node):**

1. Genesis: `inflation_max=0.24`, `community_tax=0`, `blocks_per_year=6307200`.
2. Delegate 1,000 COCKROACH; wait `N` blocks.
3. Delegator rewards ≈ `1000 × 0.24 × 0.90 × N / 6307200`.
4. Validator commission for that stake ≈ the 10% column.
5. Gov sets `inflation_max` to `0.30` → per-block rewards rise ~25%.
6. Gov sets `inflation_max` to `0.50` → proposal may pass, **paid** rate stays capped at 0.36.

Reference table (1,000 COCKROACH delegated at 0.24):

| Period | Gross | Delegator (90%) | Validator (10%) |
| --- | --- | --- | --- |
| 1 block | ~0.00003805 | ~0.00003425 | ~0.00000381 |
| 1 day | 0.6575 | 0.5918 | 0.0658 |
| 1/12 year | 20.00 | 18.00 | 2.00 |
| 1 year | 240.00 | 216.00 | 24.00 |

---

## B. Yearly decay (`applyYearlyDecay` in `evmd/app.go`)

| ID | Case | Expected |
| --- | --- | --- |
| B1 | At height = `blocks_per_year` with max 0.24, step 0.012 | `inflation_max` → **0.228** |
| B2 | Next year boundary | → **0.216** |
| B3 | Mid-year gov sets max to 0.30; next step | → **0.288** |
| B4 | `inflation_rate_change` = 0 | No further decay |
| B5 | Step would go below `inflation_min` 0.06 | Stops at **0.06** |
| B6 | Already at floor | No-op (no event / no change) |

**Automated:** `TestYearlyDecayStepMath` (formula only)

**Manual:** On a throwaway genesis set `blocks_per_year` to `"100"`, start node, query
`mint params` at heights 100 and 200.

```text
./build/cockroachd q mint params --home ~/.cockroachd
```

---

## C. Commission guard (`evmd/commission_guard.go`)

| ID | Case | Expected |
| --- | --- | --- |
| C1 | Create validator rate 0.10, max 0.10 | **Accepted** |
| C2 | Rate 0.05 | **Rejected** |
| C3 | Rate 0.15 | **Rejected** |
| C4 | Rate 0.10, max 0.15 | **Rejected** |
| C5 | Same checks via EVM staking precompile (MetaMask) | Same accept/reject |
| C6 | Genesis gentx without 0.10/0.10 | Gentx / start should fail or reject |

**Automated:** none (needs staking keeper / live tx). Use manual.

**Manual:**

```text
./build/cockroachd tx staking create-validator ... \
  --commission-rate 0.05 --commission-max-rate 0.05 ...
# expect error: commission rate and max rate must both be exactly 0.10
```

Also:

```text
./build/cockroachd q distribution commission <valoper>
```

Should show **0.10**.

---

## D. Burn module (`evmd/x/burn`)

| ID | Case | Expected |
| --- | --- | --- |
| D1 | Default `burn_fraction` | **0.5** |
| D2 | `burn_fraction` in [0, 1] | Validate OK |
| D3 | `burn_fraction` &lt; 0 or &gt; 1 | Validate error |
| D4 | After a block with fees `F`, fraction 0.5 | Supply ↓ by ~F/2; stakers get ~F/2 |
| D5 | Send 100 COCKROACH to burn module account | EndBlock burns 100; `TotalBurned` += 100 |
| D6 | Gov sets `burn_fraction` to 0.7 | Applies from next block |
| D7 | Non-`acockroach` sent to burn address | Not burned (stays locked) |

**Automated:** `TestBurnParamsValidate`, `TestDefaultBurnFractionIsHalf`

**Manual:**

```text
./build/cockroachd q burn params --home ~/.cockroachd
./build/cockroachd q auth module-account burn --home ~/.cockroachd
# send to that address, then:
./build/cockroachd q burn total-burned --home ~/.cockroachd
```

---

## E. Blacklist / freeze (`evmd/x/blacklist`)

| ID | Case | Expected |
| --- | --- | --- |
| E1 | Parse `0x...` and matching `cockroach1...` | Same 20-byte account |
| E2 | Invalid hex / bech32 | Error |
| E3 | Frozen account signs Cosmos tx (Keplr) | Rejected |
| E4 | Frozen account signs EVM tx (MetaMask) | Rejected |
| E5 | Native send from frozen (incl. authz / precompile path) | Rejected |
| E6 | Incoming transfer to frozen address | **Allowed** |
| E7 | Unfreeze | Sends work again |
| E8 | Freeze via gov with either address form | Both accepted |

**Automated:** `TestParseAddressHexAndBech32SameAccount`, `TestParseAddressInvalid`

**Manual:** Submit `/cockroach.blacklist.v1.MsgFreezeAddress` via gov, vote, then try
sends both ways; then unfreeze.

```text
./build/cockroachd q blacklist is-frozen <address> --home ~/.cockroachd
```

(Exact query command name may follow AutoCLI; if missing, use gRPC/REST or the typed
query once registered.)

---

## F. Standby + heartbeat (`evmd/x/standby`, `cmd/heartbeat`)

| ID | Case | Expected |
| --- | --- | --- |
| F1 | Default params | max 100, window self-stake 5000×10^18, window 100 |
| F2 | Invalid params (max 0, window 0, negative stake) | Validate error |
| F3 | Sign bytes format | `cockroach-heartbeat/v1\|chain\|valoper\|height\|HASH` |
| F4 | 50k tokens, rate 0.24, 24h online | ~32 COCKROACH that day (truncate) |
| F5 | Same with 18 heartbeats | ~75% of full day |
| F6 | Self-stake 4,999 | **Nothing** paid |
| F7 | Jailed validator heartbeat | Rejected; nothing paid |
| F8 | Heartbeat height older than window / wrong hash / wallet sig | Rejected |
| F9 | Active (bonded) validator heartbeat | Accepted, **no** standby mint |
| F10 | 120 qualifying validators | Only top **100** by stake paid |
| F11 | Gov changes min_self_stake / max_standby | Applies from **next** day |

**Automated:** `TestStandbyDefaultParams`, `TestStandbyParamsValidate`,
`TestHeartbeatSignBytes`, `TestStandbyDailyRewardMath`

**Manual:**

1. Run a non-bonded validator with ≥ 5,000 self-stake.
2. Build sidecar: `go build -o ../build/heartbeat ./cmd/heartbeat` (from `evmd/`).
3. Send heartbeats for several UTC hours (`--once` or hourly).
4. After UTC day rollover, check operator + delegator balances for liquid mint
   (not distribution rewards).

```text
./build/cockroachd tx standby heartbeat \
  <operator> <valoper> <height> <hash-hex> <sig-hex> \
  --from <key> --home ~/.cockroachd --chain-id <id> -y
```

---

## G. Branding / genesis config (no Go unit tests)

| ID | Case | Expected |
| --- | --- | --- |
| G1 | Bech32 accounts | Prefix `cockroach1...` |
| G2 | Binary / home | `cockroachd`, `~/.cockroachd` |
| G3 | Denoms in genesis | `acockroach` on bank/staking/mint/evm/gov |
| G4 | `community_tax` | `0` |
| G5 | Unbonding | 7 days (`604800s`) |
| G6 | Fee market on | `no_base_fee=false`, positive `max_gas` |

**Manual:** After [SETUP.md](SETUP.md) genesis edits:

```text
./build/cockroachd q staking params
./build/cockroachd q mint params
./build/cockroachd keys show alice -a --keyring-backend test --home ~/.cockroachd
```

---

## Suggested run order

1. **Unit tests** (this file, section “How to run”) — confirm green.
2. **Build** — `make build` → `./build/cockroachd`.
3. **Manual setup** — [SETUP.md](SETUP.md) through start.
4. **Smoke** — A (rewards), C (commission), D (burn), E (freeze), B (decay on throwaway), F (standby).
5. **Before public testnet** — full PDF 4-node checklist (all rows above on multi-node).

---

## What is not automated yet

These need a running app / multi-node harness (future work):

- Full `CommissionGuard` against live `MsgCreateValidator`
- `BurnFees` / `BurnAddressBalance` EndBlock integration
- Ante freeze decorator end-to-end
- Standby `payDay` with real validators and heartbeats
- BABYROACH (out of Milestone C)

Unit tests cover pure math, params validation, address parsing, and heartbeat sign bytes.
