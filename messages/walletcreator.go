package messages

type CreateWalletsRequest struct {
	NumWallets    int
	Evm           bool
	ChainState    []byte
	ProviderState []byte
	RunnerType    string
}

type CreateWalletsResponse struct {
	Mnemonics []string
}
