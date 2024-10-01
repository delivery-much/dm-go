package request

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
)

type Response http.Response

// IsSuccessCode checks if response status code is from a success response.
func (r *Response) IsSuccessCode() bool {
	return r.StatusCode >= 200 && r.StatusCode <= 299
}

// IsFailureCode checks if response status code is from an error response.
func (r *Response) IsFailureCode() bool {
	return !r.IsSuccessCode()
}

// DecodeJSON decodes the response body into an object pointer
func (r *Response) DecodeJSON(value any) error {
	if value == nil || reflect.ValueOf(value).Kind() != reflect.Ptr {
		return errors.New("value must be a valid pointer of a struct")
	}

	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(value)
}
