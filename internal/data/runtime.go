package data

import (
	"fmt"
	"strconv"
)

// Declaring a custom MarshalJSON() on runtime data to satisy the json.Marshaler interface.

// Declare a custom Runtime type, which has the underlying type int32
type Runtime int32

func (r Runtime) MarshalJSON() ([]byte, error) {
	// Generate a string containing the movie runtime in the required format
	jsonValue := fmt.Sprintf("%d mins", r)

	// Use the strconv.Quote() function on the string to wrap it in double quotes
	quotedJSONValue := strconv.Quote(jsonValue)

	// Convert the quoted string value to a byte slice and return it.
	return []byte(quotedJSONValue), nil
}
