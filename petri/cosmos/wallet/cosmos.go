package wallet

import (
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/go-bip39"

	petritypes "github.com/skip-mev/ironbird/petri/core/types"
)

// CosmosWallet implements the types.Wallet interface and represents a valid Cosmos SDK wallet
type CosmosWallet struct {
	mnemonic     string
	privKey      cryptotypes.PrivKey
	keyName      string
	bech32Prefix string
	signingAlgo  string
}

var _ petritypes.WalletI = &CosmosWallet{}

// NewWallet creates a new CosmosWallet from a mnemonic and a keyname
func NewWallet(keyname string, mnemonic string, config petritypes.WalletConfig) (*CosmosWallet, error) {
	derivedPrivKey, err := config.DerivationFn(mnemonic, "", config.HDPath.String())
	if err != nil {
		return nil, err
	}

	privKey := config.GenerationFn(derivedPrivKey)

	return &CosmosWallet{
		mnemonic:     mnemonic,
		privKey:      privKey,
		keyName:      keyname,
		signingAlgo:  config.SigningAlgorithm,
		bech32Prefix: config.Bech32Prefix,
	}, nil
}

// NewGeneratedWallet creates a new CosmosWallet from a randomly generated mnemonic and a keyname
func NewGeneratedWallet(keyname string, config petritypes.WalletConfig) (*CosmosWallet, error) {
	entropy, err := bip39.NewEntropy(128)
	if err != nil {
		return nil, err
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return nil, err
	}

	return NewWallet(keyname, mnemonic, config)
}

// KeyName returns the keyname of the wallet
func (w *CosmosWallet) KeyName() string {
	return w.keyName
}

// FormattedAddress returns a Bech32 formatted address for the wallet, using the provided Bech32 prefix
// in WalletConfig
func (w *CosmosWallet) FormattedAddress() string {
	return types.MustBech32ifyAddressBytes(w.bech32Prefix, w.privKey.PubKey().Address())
}

// Address returns the raw address bytes for the wallet
func (w *CosmosWallet) Address() []byte {
	return w.privKey.PubKey().Address()
}

// FormattedAddressWithPrefix returns a Bech32 formatted address for the wallet, using the provided prefix
func (w *CosmosWallet) FormattedAddressWithPrefix(prefix string) string {
	return types.MustBech32ifyAddressBytes(prefix, w.privKey.PubKey().Address())
}

// PublicKey returns the public key for the wallet
func (w *CosmosWallet) PublicKey() (cryptotypes.PubKey, error) {
	return w.privKey.PubKey(), nil
}

// PrivateKey returns the private key for the wallet
func (w *CosmosWallet) PrivateKey() (cryptotypes.PrivKey, error) {
	return w.privKey, nil
}

// SigningAlgo returns the signing algorithm for the wallet
func (w *CosmosWallet) SigningAlgo() string {
	return w.signingAlgo
}

// Mnemonic returns the mnemonic for the wallet
func (w *CosmosWallet) Mnemonic() string {
	return w.mnemonic
}
