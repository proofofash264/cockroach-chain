package types

const (
	ModuleName = "blacklist"
	StoreKey   = ModuleName
	RouterKey  = ModuleName
)

func DefaultGenesis() *GenesisState {
	return &GenesisState{Entries: nil}
}
