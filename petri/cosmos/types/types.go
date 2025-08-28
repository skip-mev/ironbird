package types

import "math/big"

type Coin struct {
	Amount big.Int
	Denom  string
}
