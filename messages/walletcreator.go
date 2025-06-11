package messages

type CreateWalletsRequest struct {
	WorkflowID    string
	NumWallets    int
	Evm           bool
	ChainState    []byte
	ProviderState []byte
	RunnerType    string
}

type CreateWalletsResponse struct {
	Mnemonics []string
}
