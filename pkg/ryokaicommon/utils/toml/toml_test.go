package toml

import (
	"testing"
)

var testCases = []struct {
	testCaseName      string
	oldConfigStr      string
	newTomlValue      Value
	expectedConfigStr string
	expectError       bool
}{
	{
		testCaseName: "Update existing string variable in a tag",
		oldConfigStr: `
[tag]
variable = "old value"
`,
		newTomlValue: Value{Tag: "tag", Name: "variable", Value: "new value"},
		expectedConfigStr: `
[tag]
variable = "new value"
`,
		expectError: false,
	},
	{
		testCaseName: "Update existing number variable in a tag",
		oldConfigStr: `
[tag]
variable = 8000
`,
		newTomlValue: Value{Tag: "tag", Name: "variable", Value: "8010"},
		expectedConfigStr: `
[tag]
variable = 8010
`,
		expectError: false,
	},
	{
		testCaseName: "Update existing boolean variable in a tag",
		oldConfigStr: `
[tag]
variable = false
`,
		newTomlValue: Value{Tag: "tag", Name: "variable", Value: "true"},
		expectedConfigStr: `
[tag]
variable = true
`,
		expectError: false,
	},
	{
		testCaseName: "Update existing string unquoted variable in a tag",
		oldConfigStr: `
[tag]
variable = text
`,
		newTomlValue: Value{Tag: "tag", Name: "variable", Value: "new"},
		expectedConfigStr: `
[tag]
variable = "new"
`,
		expectError: false,
	},
	{
		testCaseName:      "Update existing variable in base section",
		oldConfigStr:      `variable = "old base value"`,
		newTomlValue:      Value{Tag: "", Name: "variable", Value: "new base value"},
		expectedConfigStr: `variable = "new base value"`,
		expectError:       false,
	},
	{
		testCaseName:      "Update with empty value",
		oldConfigStr:      `variable = "some value"`,
		newTomlValue:      Value{Tag: "", Name: "variable", Value: ""},
		expectedConfigStr: `variable = ""`,
		expectError:       false,
	},
	{
		testCaseName: "Update complex toml file",
		oldConfigStr: `
basevar = text
[tag1]
variable = 8000
[tag2]
variable2 = true
`,
		newTomlValue: Value{Tag: "tag1", Name: "variable", Value: "8010"},
		expectedConfigStr: `
basevar = text
[tag1]
variable = 8010
[tag2]
variable2 = true
`,
		expectError: false,
	},
	{
		testCaseName: "Update complex toml file with base",
		oldConfigStr: `
basevar = text
[tag1]
variable = 8000
[tag2]
variable2 = true
`,
		newTomlValue: Value{Tag: "", Name: "basevar", Value: "new"},
		expectedConfigStr: `
basevar = "new"
[tag1]
variable = 8000
[tag2]
variable2 = true
`,
		expectError: false,
	},
	{
		testCaseName:      "No content",
		oldConfigStr:      ``,
		newTomlValue:      Value{Tag: "tag", Name: "variable", Value: "value"},
		expectedConfigStr: ``,
		expectError:       true,
	},
}

// TestSetTomlVar runs the table-driven tests
func TestSetTomlVar(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.testCaseName, func(t *testing.T) {
			// Assuming SetTomlVar is modified to accept string configuration as input and output
			// and simulate file operations internally for the sake of these test cases.
			updatedConfig, err := SetTomlVar(tc.newTomlValue, tc.oldConfigStr)

			if tc.expectError {
				if err == nil {
					t.Errorf("[%s] expected error but got none", tc.testCaseName)
				}
			} else {
				if err != nil {
					t.Errorf("[%s] unexpected error: %v", tc.testCaseName, err)
				}
				if updatedConfig != tc.expectedConfigStr {
					t.Errorf("[%s] expected updated config to be %q, got %q", tc.testCaseName, tc.expectedConfigStr, updatedConfig)
				}
			}
		})
	}
}
