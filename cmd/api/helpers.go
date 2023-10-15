package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	"greenlight.example.com/internal/validator"
)

// Retrieve the "id" url param from the current request context, then convert it to
// an integer and return it.  If the operation isnt successfulm return 0 and an error
func (app *application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}
	return id, nil
}

// define and envelope type
type envelope map[string]any

// writeJSON() takes the destination http.ResponseWriter, the HTTP statuscode to send, the
// data to encode to JSON, and a header map containing any additional HTTP headers we want
// to include in the response.
// We use a data parameter with the envelope type instead of any.
func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	// Encode the data to JSON, returning any error if there was one
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	// Append a newline to make it easier to view in terminal applications
	js = append(js, '\n')

	// At this point we know we wont encounter any more errors before writing the
	// the response so we can add any headers that we want to include.
	for key, value := range headers {
		w.Header()[key] = value
	}

	// Add the "Content-Type: application/json" header, then write the status code and
	// JSON response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

// readJSON() limits the request body and attempts to decode the body contents into the
// receiving destination struct, returning a variety of error types to help the client
// understand if the decoding failed.
func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Use http.MaxBytesReader() to limit the size of the request body to 1MB.
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	// Initialise the JSON decoder, and call the DisallowUnknownFields() method on it
	// before decoding.  This returns and error if fields are discovered which does not
	// map to the target destination.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// Decode the request body to the destination
	err := dec.Decode(dst)
	if err != nil {
		// If there is an error during decoding, start the triage...
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		// Use the errors.As function to check whether the error has the type
		// *json.SyntaxError.  If it does then return a plain-english error message
		// which includes the location of the problem
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)

		// In some circumstances Decode() may also return an io.ErrUnexpectedEOF error
		// for syntax errors in the JSON. So we check for this using errors.Is() and
		// return a generic error message. There is an open issue regarding this at
		// https://github.com/golang/go/issues/25956.
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly JSON")

		// Likewise, catch any *json.UnmarshalTypeError errors. These occur when the
		// JSON value is the wrong type for the target destination. If the error relates
		// to a specific field, then we include that in our error message to make it
		// easier for the client to debug.
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

			// An io.EOF error will be returned by Decode() if the request body is empty. We
			// check for this with errors.Is() and return a plain-english error message
			// instead.
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

			// If the JSON contains a field which cannot be mapped to the target destination
			// then Decode() will now return an error message in the format "json: unknown
			// field "<name>"". We check for this, extract the field name from the error,
			// and interpolate it into our custom error message.
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)

			// Use the errors.As() function to check whether the error has the type
			// *http.MaxBytesError. If it does, then it means the request body exceeded our
			// size limit of 1MB and we return a clear error message.
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)

			// A json.InvalidUnmarshalError error will be returned if we pass something
			// that is not a non-nil pointer to Decode(). We catch this and panic,
			// rather than returning an error to our handler. At the end of this chapter
			// we'll talk about panicking versus returning errors, and discuss why it's an
			// appropriate thing to do in this specific situation.
		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		// For anything else, return the error message as-is.
		default:
			return err
		}
	}

	// Call Decode() again, using a pointer to an empty anonymous struct as the
	// destination. If the request body only contained a single JSON value this will
	// return an io.EOF error. So if we get anything else, we know that there is
	// additional data in the request body and we return our own custom error message.
	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contains a single JSON value")
	}

	return nil
}

// The readString() helper returns a string value from the query string, or the provided
// default value if no matching key could be found
func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	// Extract the value for a given key from the query string.  If no key exists this
	// will return the empty string "".
	s := qs.Get(key)

	// If not key exists(or the value is empty) then return the default value.
	if s == "" {
		return defaultValue
	}
	// Otherwise return the string.
	return s
}

// the readCSV() helper reads a string value from the query string and then splits it
// into a slice on the comma character.  If no matching key could be found, it returns
// the provided default value.
func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	//Extract the value from the query string
	csv := qs.Get(key)

	// If no key exists(or the value is empty) then return the default value.
	if csv == "" {
		return defaultValue
	}

	// Otherwise parse the value into a []string slice and return it.
	return strings.Split(csv, ",")
}

// The readInt() helper reads a string value from the query string and converts it to an
// integer before returning.  If the value could not be converted to an integer, then we
// record an error message in the provided Validator instance.
func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	//Extract the value from the query string.
	s := qs.Get(key)

	// If no key exists (or the value is empty), then return the default value
	if s == "" {
		return defaultValue
	}
	// Try to convert the value to an int.  If this fails, add an error message to
	// the validator instance and return the default version.
	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}
	// Otherwise, return the converted integer value
	return i
}
