package messages

// CreateWalletsRequest represents the request to create wallets
type CreateWalletsRequest struct {
	NumWallets    int
	GaiaEVM       bool
	ChainState    []byte
	ProviderState []byte
	RunnerType    string
}

// CreateWalletsResponse represents the response from creating wallets
type CreateWalletsResponse struct {
	Mnemonics     []string
	ProviderState []byte
	ChainState    []byte
}
