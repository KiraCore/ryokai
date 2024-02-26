package json

import (
	"encoding/json"
	"fmt"
	"strings"
)

type (
	// Value represents a JSON value to be updated, identified by a dot-separated key path.
	Value struct {
		Value any
		Key   string
	}

	TargetKeyNotFoundError struct {
		Key string
	}

	ExpectedMapError struct {
		Key string
	}
)

// UpdateJSONValue updates a value in a JSON object based on a dot-separated key path.
// It unmarshal the input byte slice into a map, updates the value at the specified path,
// and then marshals the map back into a byte slice.
// Returns the updated JSON as a byte slice, or an error if the update cannot be performed.
func UpdateJSONValue(input []byte, config Value) ([]byte, error) {
	var mapRepresentationJSON map[string]any

	err := json.Unmarshal(input, &mapRepresentationJSON)
	if err != nil {
		return nil, fmt.Errorf("can't unmarshal JSON: %w", err)
	}

	keys := strings.Split(config.Key, ".")

	err = setNested(mapRepresentationJSON, keys, config.Value)
	if err != nil {
		return nil, err
	}

	result, err := json.Marshal(mapRepresentationJSON)
	if err != nil {
		return nil, fmt.Errorf("can't marshal JSON back: %w", err)
	}

	return result, nil
}

// setNested is a helper function that recursively navigates through a map based on a slice of keys,
// updating the value at the final key. It supports nested maps as intermediate steps in the path.
// Returns an error if any key in the path does not exist or does not lead to a map.
func setNested(mapRepresentationJSON map[string]any, keys []string, value any) error {
	for keyIndex, key := range keys[:len(keys)-1] {
		nested, keyExists := mapRepresentationJSON[key]
		if !keyExists {
			return &TargetKeyNotFoundError{Key: strings.Join(keys[:keyIndex+1], ".")}
		}

		nestedMap, keyExists := nested.(map[string]any)
		if !keyExists {
			return &ExpectedMapError{Key: strings.Join(keys[:keyIndex+1], ".")}
		}

		mapRepresentationJSON = nestedMap
	}

	lastKey := keys[len(keys)-1]
	mapRepresentationJSON[lastKey] = value

	return nil
}

func (e *TargetKeyNotFoundError) Error() string {
	return fmt.Sprintf("target key does not exist: %s", e.Key)
}

func (e *ExpectedMapError) Error() string {
	return fmt.Sprintf("expected map for key: %s", e.Key)
}
