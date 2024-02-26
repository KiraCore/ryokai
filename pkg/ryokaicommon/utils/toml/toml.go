// Package toml provides utilities for manipulating TOML (Tom's Obvious, Minimal Language) configurations
// within Go applications. It offers a straightforward API for updating variables within a TOML configuration
// string, handling both base and tagged sections. The package is designed to facilitate the dynamic
// modification of TOML configurations, allowing for the programmatic setting of configuration values
// without the need for direct file manipulation or external TOML parsing libraries and
// with the possibility of saving original comments
//
// Example Usage:
// Given a TOML configuration string, the package can be
// used to update the value of a configuration variable as follows:
//
//	tomlValue := toml.Value{
//	    Tag:   "server",
//	    Name:  "port",
//	    Value: "8080",
//	}
//	updatedConfig, err := toml.SetTomlVar(tomlValue, configStr)
//	if err != nil {
//	    // Handle error
//	}
//
// This utility is particularly useful for applications that need to modify configuration values on the fly,
// based on dynamic conditions or user input, without relying on external configuration management tools.
package toml

import (
	"fmt"
	"strings"

	"github.com/KiraCore/ryokai/pkg/ryokaicommon/utils/base"
)

type (
	VariableNotFoundError struct {
		VariableName string
		Tag          string
	}

	// Value represents a TOML configuration variable, including its tag (section),
	// name, and value. The tag is optional for variables in the base section.
	Value struct {
		Tag   string
		Name  string
		Value string
	}
)

func (e *VariableNotFoundError) Error() string {
	return fmt.Sprintf("variable '%s' not found in tag '%s'", e.VariableName, e.Tag)
}

// SetTomlVar updates a specific variable within a TOML configuration string.
// It takes a Value struct containing the tag, name, and new value of the variable
// to be updated, along with the original TOML configuration string.
// It returns the updated TOML configuration string or an error
// if the variable could not be found or updated.
func SetTomlVar(tomlValue Value, configStr string) (string, error) {
	tag := strings.TrimSpace(tomlValue.Tag)
	name := strings.TrimSpace(tomlValue.Name)
	value := strings.TrimSpace(tomlValue.Value)

	tag = formatTag(tag)

	lines := strings.Split(configStr, "\n")

	foundLineIndex, err := searchLine(tag, name, lines)
	if err != nil {
		return "", err
	}

	value = formatValue(value)

	lines[foundLineIndex] = fmt.Sprintf("%s = %s", name, value)

	return strings.Join(lines, "\n"), nil
}

// formatTag prepares the tag for insertion into the TOML configuration string.
func formatTag(tag string) string {
	if tag != "" {
		return "[" + tag + "]"
	}

	return tag
}

// searchLine searches for the line index within the TOML configuration string
// where the specified variable (within a tag, if provided) can be updated.
func searchLine(tag, name string, lines []string) (int, error) {
	var (
		tagFound    = tag == ""
		withinScope = tag == ""
	)

	for lineIndex, line := range lines {
		trimmedCurrentLine := strings.TrimSpace(line)

		if !tagFound && strings.Contains(trimmedCurrentLine, tag) {
			tagFound = true
			withinScope = true

			continue
		}

		if withinScope && strings.HasPrefix(trimmedCurrentLine, name+" =") {
			return lineIndex, nil
		}
	}

	// C-like conventions
	return -1, &VariableNotFoundError{
		VariableName: name,
		Tag:          tag,
	}
}

// formatValue formats the given value for insertion into the TOML configuration string.
// It quotes strings if necessary and leaves boolean and numeric values unquoted.
func formatValue(value string) string {
	switch {
	case value == "" || strings.Contains(value, " "):
		// If the value is empty or contains spaces, quote it
		return fmt.Sprintf("\"%s\"", value)
	case base.IsBool(value) || base.IsNumber(value):
		// If the value is a boolean or a number, return as is
		return value
	default:
		// Otherwise, wrap in quotes to ensure it's treated as a string
		return fmt.Sprintf("\"%s\"", value)
	}
}
