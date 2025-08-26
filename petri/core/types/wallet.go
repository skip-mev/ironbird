package types

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
)

// WalletConfig is a configuration for a Cosmos SDK type wallet
type WalletConfig struct {
	DerivationFn     hd.DeriveFn     // DerivationFn is the function to derive a seed from a mnemonic
	GenerationFn     hd.GenerateFn   // GenerateFn is the function to derive a private key from a seed
	Bech32Prefix     string          // Bech32Prefix is the default prefix that should be used for formatting addresses
	HDPath           *hd.BIP44Params // HDPath is the default HD path to use for deriving keys
	SigningAlgorithm string          // SigningAlgorithm is the default signing algorithm to use
}

func (c WalletConfig) ValidateBasic() error {
	if c.DerivationFn == nil {
		return fmt.Errorf("derivation function cannot be nil")
	}

	if c.GenerationFn == nil {
		return fmt.Errorf("generation function cannot be nil")
	}

	if c.Bech32Prefix == "" {
		return fmt.Errorf("bech32 prefix cannot be empty")
	}

	if c.HDPath == nil {
		return fmt.Errorf("HD path cannot be nil")
	}

	if c.SigningAlgorithm == "" {
		return fmt.Errorf("signing algorithm cannot be empty")
	}

	return nil
}
