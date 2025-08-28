## Using the Petri chain package

The `chain` package is the main entrypoint for creating Cosmos chains using Petri.
A chain is a collection of nodes that are either validators or full nodes and are fully connected to each other (for now).

### Creating a chain

The main way to create a chain is by using the `NewChain` function. 
The function accepts a `ChainConfig` struct that defines the chain's configuration.

The basic gist on how to use the CreateChain function is as such:
```go
var infraProvider provider.Provider

// create the chain
chain, err := chain.CreateChain(ctx, logger, provider, chainConfig)

if err != nil {
    panic(err)
}

err = s.chain.Init(ctx)

if err != nil {
    panic(err)
}
```

The CreateChain function only creates the nodes and their underlying workloads using the Provider. It does not
create the genesis file, start the chain or anything else. All of the initial configuration is done in the `Init` function.

### Waiting for a chain to go live

After creating a chain, you can wait for it to go live by using the `WaitForBlocks` function.

```go
err = chain.WaitForBlocks(ctx, 1) // this will wait for the chain to produce 1 block
```

### Funding wallets

To fund a wallet, you can use the `cosmosutil` package to send a `bank/send` transaction to the chain.

```go
encodingConfig := cosmosutil.EncodingConfig{
    InterfaceRegistry: encodingConfig.InterfaceRegistry,
    Codec:             encodingConfig.Codec,
    TxConfig:          encodingConfig.TxConfig,
}

interactingFaucet := cosmosutil.NewInteractingWallet(chain, chain.GetFaucetWallet(), encodingConfig)

user, err := wallet.NewGeneratedWallet("user", chainConfig.WalletConfig)

sendGasSettings := petritypes.GasSettings{
    Gas:         100000,
    PricePerGas: int64(0),
    GasDenom:    chainConfig.Denom,
}

// this will block until the bankSend transaction lands on the chain
txResp, err := s.chainClient.BankSend(ctx, *interactingFaucet, user.Address(), sdk.NewCoins(sdk.NewCoin(chain.GetConfig().Denom, sdkmath.NewInt(1000000000))), sendGasSettings, true)

if err != nil {
    panic(err)
}
```
