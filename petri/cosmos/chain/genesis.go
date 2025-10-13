package chain

import (
	"fmt"

	"github.com/tidwall/sjson"
)

// GenesisKV is used in ModifyGenesis to specify which keys have to be modified
// in the resulting genesis

type GenesisKV struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// GenesisModifier represents a type of function that takes in genesis formatted in bytes
// and returns a modified genesis file in the same format

type GenesisModifier func([]byte) ([]byte, error)

var _ GenesisModifier = ModifyGenesis(nil)

// NewGenesisKV is a function for creating a GenesisKV object
func NewGenesisKV(key string, value interface{}) GenesisKV {
	return GenesisKV{
		Key:   key,
		Value: value,
	}
}

// ModifyGenesis is a function that is a GenesisModifier and takes in GenesisKV
// to specify which fields of the genesis file should be modified
func ModifyGenesis(genesisKV []GenesisKV) func([]byte) ([]byte, error) {
	return func(genbz []byte) ([]byte, error) {
		out := genbz
		var err error
		for idx, kv := range genesisKV {
			out, err = sjson.SetBytes(out, kv.Key, kv.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to set value (index:%d) in genesis json: %v", idx, err)
			}
		}
		return out, nil
	}
}
