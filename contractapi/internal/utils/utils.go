package utils

import (
	"strconv"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

// ValidateErrorsToString converts errors from JSON schema output into readable string
func ValidateErrorsToString(resErrors []gojsonschema.ResultError) string {
	toReturn := ""

	for i, v := range resErrors {
		toReturn += strconv.Itoa(i+1) + ". " + v.String() + "\n"
	}

	return strings.Trim(toReturn, "\n")
}

// StringInSlice returns whether string exists in string slice
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
