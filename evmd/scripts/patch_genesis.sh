#!/usr/bin/env bash
# Apply Cockroach Chain v1 genesis parameters after `cockroachd init` and before gentxs.
# Usage: ./scripts/patch_genesis.sh [path-to-genesis.json]
set -euo pipefail

G="${1:-$HOME/.cockroachd/config/genesis.json}"
D="acockroach"

if [[ ! -f "$G" ]]; then
  echo "genesis not found: $G" >&2
  exit 1
fi

jq --arg d "$D" '
# Denom everywhere
.app_state.staking.params.bond_denom = $d
| .app_state.mint.params.mint_denom = $d
| .app_state.evm.params.evm_denom = $d
| .app_state.gov.params.min_deposit[0].denom = $d
| .app_state.gov.params.expedited_min_deposit[0].denom = $d
| .app_state.bank.denom_metadata = [{
  "description": "Native staking and gas token",
  "base": $d, "display": "COCKROACH", "name": "COCKROACH", "symbol": "COCKROACH",
  "denom_units": [{"denom": $d, "exponent": 0}, {"denom": "COCKROACH", "exponent": 18}]
}]
# Staking rewards and yearly decay
| .app_state.mint.minter.inflation = "0.000000000000000000"
| .app_state.mint.params.inflation_max = "0.240000000000000000"
| .app_state.mint.params.inflation_rate_change = "0.012000000000000000"
| .app_state.mint.params.inflation_min = "0.060000000000000000"
| .app_state.mint.params.blocks_per_year = "6307200"
| .app_state.distribution.params.community_tax = "0.000000000000000000"
| .app_state.staking.params.min_commission_rate = "0.100000000000000000"
# Burn
| .app_state.burn.params.burn_fraction = "0.500000000000000000"
| .app_state.burn.total_burned = "0"
| .app_state.gov.params.burn_vote_veto = true
# Standby validators
| .app_state.standby.params.max_standby = 100
| .app_state.standby.params.min_self_stake = "5000000000000000000000"
| .app_state.standby.params.heartbeat_window = "100"
| .app_state.standby.total_paid = "0"
# Blacklist starts empty
| .app_state.blacklist.entries = []
# Unbonding and slashing
| .app_state.staking.params.unbonding_time = "604800s"
| .app_state.staking.params.max_validators = 100
| .app_state.slashing.params.signed_blocks_window = "10000"
| .app_state.slashing.params.min_signed_per_window = "0.500000000000000000"
| .app_state.slashing.params.downtime_jail_duration = "600s"
| .app_state.slashing.params.slash_fraction_downtime = "0.000100000000000000"
| .app_state.slashing.params.slash_fraction_double_sign = "0.050000000000000000"
| .consensus_params.evidence.max_age_duration = "604800s"
| .consensus_params.evidence.max_age_num_blocks = "120960"
# Governance
| .app_state.gov.params.voting_period = "259200s"
| .app_state.gov.params.expedited_voting_period = "86400s"
| .app_state.gov.params.quorum = "0.334000000000000000"
| .app_state.gov.params.threshold = "0.500000000000000000"
# Gas fees
| .app_state.feemarket.params.no_base_fee = false
| .app_state.feemarket.params.base_fee = "10000000000"
| .app_state.feemarket.params.min_gas_price = "10000000000"
| .app_state.feemarket.params.min_gas_multiplier = "0.500000000000000000"
| .consensus_params.block.max_gas = "40000000"
' "$G" > "$G.tmp" && mv "$G.tmp" "$G"

echo "Patched $G"
echo "Next: cockroachd genesis validate-genesis --home <home>"
