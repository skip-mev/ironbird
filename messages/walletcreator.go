package messages

type CreateWalletsRequest struct {
	WorkflowID    string
	NumWallets    int
	IsEvmChain    bool
	ChainState    []byte
	ProviderState []byte
	RunnerType    string
}

type CreateWalletsResponse struct {
	Mnemonics []string
}
