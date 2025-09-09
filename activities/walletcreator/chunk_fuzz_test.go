package walletcreator

import (
	"strings"
	"sync"
	"testing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/petri/core/types"
	petriutil "github.com/skip-mev/ironbird/petri/core/util"
	"github.com/skip-mev/ironbird/petri/cosmos/wallet"
	"github.com/stretchr/testify/require"
)

var _ types.WalletI = &fakeWallet{}

type fakeWallet struct {
	addr string
}

func (f fakeWallet) FormattedAddress() string {
	return f.addr
}

func (f fakeWallet) KeyName() string {
	//TODO implement me
	panic("implement me")
}

func (f fakeWallet) Address() []byte {
	//TODO implement me
	panic("implement me")
}

func (f fakeWallet) FormattedAddressWithPrefix(prefix string) string {
	//TODO implement me
	panic("implement me")
}

func (f fakeWallet) PublicKey() (cryptotypes.PubKey, error) {
	//TODO implement me
	panic("implement me")
}

func (f fakeWallet) PrivateKey() (cryptotypes.PrivKey, error) {
	//TODO implement me
	panic("implement me")
}

func (f fakeWallet) Mnemonic() string {
	//TODO implement me
	panic("implement me")
}

// FuzzChunking verifies that commands contain >= 2 "cosmos*" strings
// regardless of numWallets chosen by the fuzzer.
func FuzzChunking(f *testing.F) {
	// seed a few interesting values (incl. edge cases)
	for _, n := range []int{1, 2, 10, 100, 101, 302, 1000} {
		f.Add(n)
	}

	f.Fuzz(func(t *testing.T, numWallets int) {
		// skip negatives, cap huge values
		if numWallets <= 0 {
			t.Skip("non-zero positive numWallets not meaningful")
		}
		// cap at 5k, keep it reasonable.
		if numWallets > 5_000 {
			numWallets = 5_000
		}

		walletCfg := testnet.EvmCosmosWalletConfig
		cfg := types.ChainConfig{
			BinaryName: "foo",
			Denom:      "wei",
			ChainId:    "444",
			HomeDir:    "/home/path",
		}

		addrs := make([]string, numWallets)
		wg := new(sync.WaitGroup)
		for i := 0; i < numWallets; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				w, err := wallet.NewGeneratedWallet(petriutil.RandomString(5), walletCfg)
				require.NoError(t, err)
				addrs[i] = w.FormattedAddress()
			}()
		}
		wg.Wait()

		// faucet
		w, err := wallet.NewGeneratedWallet(petriutil.RandomString(5), walletCfg)
		require.NoError(t, err)
		faucet := fakeWallet{addr: w.FormattedAddress()}

		commands := getFundWalletCommands(cfg, numWallets, faucet, addrs)

		for _, cmd := range commands {
			count := 0
			for _, arg := range cmd {
				if strings.Contains(arg, "cosmos1") {
					count++
					if count >= 2 {
						break
					}
				}
			}
			if count < 2 {
				t.Fatalf("failed to generate valid multi-send command with %d wallets. command: %s", numWallets, cmd)
			}
		}
	})
}
