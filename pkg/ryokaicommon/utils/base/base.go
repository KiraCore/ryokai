package base

import "strconv"

// isBool checks if the given string is a boolean value ("true" or "false").
func IsBool(value string) bool {
	_, err := strconv.ParseBool(value)

	return err == nil
}

// isNumber checks if the given string is a numeric value.
func IsNumber(value string) bool {
	_, err := strconv.ParseInt(value, 0, 64)

	return err == nil
}
