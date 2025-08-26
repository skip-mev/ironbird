package chain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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

func safeSet(data map[string]interface{}, value interface{}, path []interface{}) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}

	current := interface{}(data)
	for i, component := range path {
		isLast := i == len(path)-1

		switch comp := component.(type) {
		case string:
			m := current.(map[string]interface{})
			if isLast {
				m[comp] = value
				return nil
			}
			if m[comp] == nil {
				if i+1 < len(path) && isInt(path[i+1]) {
					m[comp] = []interface{}{}
				} else {
					m[comp] = map[string]interface{}{}
				}
			}
			current = m[comp]

		case int:
			arr := current.([]interface{})
			for len(arr) <= comp {
				arr = append(arr, nil)
			}
			if i > 0 {
				parent := data
				for j := 0; j < i-1; j++ {
					parent = parent[path[j].(string)].(map[string]interface{})
				}
				parent[path[i-1].(string)] = arr
			}
			if isLast {
				arr[comp] = value
				return nil
			}
			if arr[comp] == nil {
				if i+1 < len(path) && isInt(path[i+1]) {
					arr[comp] = []interface{}{}
				} else {
					arr[comp] = map[string]interface{}{}
				}
			}
			current = arr[comp]
		}
	}
	return nil
}

func isInt(v interface{}) bool {
	_, ok := v.(int)
	return ok
}

// ModifyGenesis is a function that is a GenesisModifier and takes in GenesisKV
// to specify which fields of the genesis file should be modified
func ModifyGenesis(genesisKV []GenesisKV) func([]byte) ([]byte, error) {
	return func(genbz []byte) ([]byte, error) {
		g := make(map[string]interface{})
		if err := json.Unmarshal(genbz, &g); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
		}

		for idx, values := range genesisKV {
			splitPath := strings.Split(values.Key, ".")

			path := make([]interface{}, len(splitPath))
			for i, component := range splitPath {
				if v, err := strconv.Atoi(component); err == nil {
					path[i] = v
				} else {
					path[i] = component
				}
			}

			if err := safeSet(g, values.Value, path); err != nil {
				return nil, fmt.Errorf("failed to set value (index:%d) in genesis json: %v\n", idx, err)
			}
		}

		out, err := json.Marshal(g)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
		}
		return out, nil
	}
}
