package messages

type CreateWalletsRequest struct {
	NumWallets    int
	GaiaEVM       bool
	ChainState    []byte
	ProviderState []byte
	RunnerType    string
}

type CreateWalletsResponse struct {
	Mnemonics []string
}
