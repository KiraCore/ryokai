package json

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

var testCases = []struct {
	name           string
	input          []byte
	config         Value
	expectedOutput []byte
	expectedError  error
}{
	{
		name:           "Update existing value",
		input:          []byte(`{"key1": "value1"}`),
		config:         Value{Key: "key1", Value: "updatedValue"},
		expectedOutput: []byte(`{"key1": "updatedValue"}`),
		expectedError:  nil,
	},
	{
		name:           "Nested update",
		input:          []byte(`{"nested": {"key": "value"}}`),
		config:         Value{Key: "nested.key", Value: "newValue"},
		expectedOutput: []byte(`{"nested": {"key": "newValue"}}`),
		expectedError:  nil,
	},
	{
		name:           "Nested x3 update",
		input:          []byte(`{"nested1": {"nested2": {"key": "value"}}}`),
		config:         Value{Key: "nested1.nested2.key", Value: "newValue"},
		expectedOutput: []byte(`{"nested1": {"nested2": {"key": "newValue"}}}`),
		expectedError:  nil,
	},
	{
		name:          "Key not found",
		input:         []byte(`{"key1": "value1"}`),
		config:        Value{Key: "key2", Value: "newValue"},
		expectedError: &TargetKeyNotFoundError{},
	},
	{
		name:          "Update within nested array",
		input:         []byte(`{"level1": {"level2": ["value1", "value2", "value3"]}}`),
		config:        Value{Key: "level1.level2.1", Value: "updatedValue2"},
		expectedError: &ExpectedMapError{},
	},
	{
		name:          "Non-existent nested key path",
		input:         []byte(`{"level1": {"level2": "value2"}}`),
		config:        Value{Key: "level1.level3.level4", Value: "value4"},
		expectedError: &TargetKeyNotFoundError{},
	},

	{
		name:          "Invalid JSON structure",
		input:         []byte(`{"nested": "notAnObject"}`),
		config:        Value{Key: "nested.key", Value: "newValue"},
		expectedError: &ExpectedMapError{},
	},
	{
		name:          "Intermediate key not an object",
		input:         []byte(`{"level1": "notAnObject", "level2": "value2"}`),
		config:        Value{Key: "level1.level2", Value: "updatedValue2"},
		expectedError: &ExpectedMapError{},
	},
}

func TestUpdateJSONValue(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := UpdateJSONValue(tc.input, tc.config)

			if tc.expectedError == nil {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				// Unmarshal the expected and actual output for deep equality check
				var expected, actual map[string]interface{}
				if err = json.Unmarshal(tc.expectedOutput, &expected); err != nil {
					t.Fatalf("Error unmarshaling expected output: %v", err)
				}
				if err = json.Unmarshal(output, &actual); err != nil {
					t.Fatalf("Error unmarshaling actual output: %v", err)
				}

				if !reflect.DeepEqual(expected, actual) {
					t.Errorf("Expected output to be %v, got %v", expected, actual)
				}
			} else {
				if errors.Is(tc.expectedError, err) {
					t.Errorf("Expected error containing %q, got %v", tc.expectedError, err)
				}
			}
		})
	}
}
