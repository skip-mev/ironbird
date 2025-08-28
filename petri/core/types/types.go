package types

import cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

// WalletI is an interface for a Cosmos SDK type wallet
type WalletI interface {
	// FormattedAddress should return a Bech32 formatted address using the default prefix
	FormattedAddress() string
	// KeyName should return the name of the key
	KeyName() string
	// Address should return the byte-encoded address
	Address() []byte
	// FormattedAddressWithPrefix should return a Bech32 formatted address using the given prefix
	FormattedAddressWithPrefix(prefix string) string
	// PublicKey should return the public key of the wallet
	PublicKey() (cryptotypes.PubKey, error)
	// PrivateKey should return the private key of the wallet
	PrivateKey() (cryptotypes.PrivKey, error)
	// Mnemonic should return the mnemonic of the wallet
	Mnemonic() string
}

// GasSettings is a configuration for Cosmos app gas settings
type GasSettings struct {
	Gas         int64  // Gas is the gas limit
	PricePerGas int64  // PricePerGas is the token price per gas
	GasDenom    string // GasDenom is the denomination of the gas token
}
