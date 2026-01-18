// Package response handles structured response formatting for the Fizzy CLI.
package response

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/robzolkos/fizzy-cli/internal/errors"
	"github.com/toon-format/toon-go"
)

// prettyPrint controls whether JSON output is indented.
var prettyPrint bool

const (
	FormatJSON = "json"
	FormatTOON = "toon"
)

var outputFormat = FormatJSON

// SetPrettyPrint enables or disables pretty-printed JSON output.
func SetPrettyPrint(enabled bool) {
	prettyPrint = enabled
}

// SetOutputFormat sets the output format (json or toon).
func SetOutputFormat(format string) {
	outputFormat = format
}

// Response represents the JSON response envelope.
type Response struct {
	Success    bool                   `json:"success" toon:"success"`
	Data       interface{}            `json:"data,omitempty" toon:"data,omitempty"`
	Error      *ErrorDetail           `json:"error,omitempty" toon:"error,omitempty"`
	Pagination *Pagination            `json:"pagination,omitempty" toon:"pagination,omitempty"`
	Location   string                 `json:"location,omitempty" toon:"location,omitempty"`
	Meta       map[string]interface{} `json:"meta,omitempty" toon:"meta,omitempty"`
}

// ErrorDetail represents an error in the response.
type ErrorDetail struct {
	Code    string      `json:"code" toon:"code"`
	Message string      `json:"message" toon:"message"`
	Status  int         `json:"status,omitempty" toon:"status,omitempty"`
	Details interface{} `json:"details,omitempty" toon:"details,omitempty"`
}

// Pagination represents pagination info in the response.
type Pagination struct {
	HasNext bool   `json:"has_next" toon:"has_next"`
	NextURL string `json:"next_url,omitempty" toon:"next_url,omitempty"`
}

// Success creates a successful response with data.
func Success(data interface{}) *Response {
	return &Response{
		Success: true,
		Data:    data,
		Meta:    createMeta(),
	}
}

// SuccessWithLocation creates a successful response with location.
func SuccessWithLocation(data interface{}, location string) *Response {
	return &Response{
		Success:  true,
		Data:     data,
		Location: location,
		Meta:     createMeta(),
	}
}

// SuccessWithPagination creates a successful response with pagination.
func SuccessWithPagination(data interface{}, hasNext bool, nextURL string) *Response {
	resp := &Response{
		Success: true,
		Data:    data,
		Meta:    createMeta(),
	}
	if hasNext || nextURL != "" {
		resp.Pagination = &Pagination{
			HasNext: hasNext,
			NextURL: nextURL,
		}
	}
	return resp
}

// Error creates an error response from a CLIError.
func Error(err *errors.CLIError) *Response {
	resp := &Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    err.Code,
			Message: err.Message,
		},
		Meta: createMeta(),
	}
	if err.Status != 0 {
		resp.Error.Status = err.Status
	}
	return resp
}

// ErrorFromError creates an error response from a generic error.
func ErrorFromError(err error) *Response {
	if cliErr, ok := err.(*errors.CLIError); ok {
		return Error(cliErr)
	}
	return Error(errors.NewError(err.Error()))
}

func createMeta() map[string]interface{} {
	return map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
}

// Print outputs the response in the configured format to stdout.
func (r *Response) Print() {
	resp := sanitizeResponse(r)
	switch outputFormat {
	case FormatTOON:
		data, err := toon.Marshal(resp)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling response: %v\n", err)
			return
		}
		fmt.Print(string(data))
	default:
		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		if prettyPrint {
			encoder.SetIndent("", "  ")
		}
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(resp); err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling response: %v\n", err)
			return
		}
		fmt.Print(buf.String())
	}
}

func sanitizeResponse(r *Response) *Response {
	if r == nil {
		return r
	}
	resp := *r
	resp.Data = sanitizeData(resp.Data)
	if resp.Error != nil {
		details := sanitizeData(resp.Error.Details)
		if isEmptySlice(details) {
			resp.Error.Details = nil
		} else {
			resp.Error.Details = details
		}
	}
	return &resp
}

func sanitizeData(value interface{}) interface{} {
	if value == nil {
		return nil
	}
	if isSlice(value) {
		return pruneEmptyArrays(value)
	}
	cleaned := pruneEmptyArrays(value)
	if isEmptySlice(cleaned) {
		return nil
	}
	return cleaned
}

func pruneEmptyArrays(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		for key, val := range v {
			cleaned := pruneEmptyArrays(val)
			if isEmptySlice(cleaned) {
				delete(v, key)
				continue
			}
			v[key] = cleaned
		}
		return v
	case []interface{}:
		for i, item := range v {
			v[i] = pruneEmptyArrays(item)
		}
		return v
	default:
		return value
	}
}

func isSlice(value interface{}) bool {
	if value == nil {
		return false
	}
	rv := reflect.ValueOf(value)
	kind := rv.Kind()
	return kind == reflect.Slice || kind == reflect.Array
}

func isEmptySlice(value interface{}) bool {
	if value == nil {
		return false
	}
	rv := reflect.ValueOf(value)
	kind := rv.Kind()
	if kind == reflect.Slice || kind == reflect.Array {
		return rv.Len() == 0
	}
	return false
}

// PrintAndExit prints the response and exits with appropriate code.
func (r *Response) PrintAndExit() {
	r.Print()
	if r.Success {
		os.Exit(errors.ExitSuccess)
	}
	// Try to get exit code from the error
	if r.Error != nil {
		switch r.Error.Code {
		case "AUTH_ERROR":
			os.Exit(errors.ExitAuthFailure)
		case "FORBIDDEN":
			os.Exit(errors.ExitForbidden)
		case "NOT_FOUND":
			os.Exit(errors.ExitNotFound)
		case "VALIDATION_ERROR":
			os.Exit(errors.ExitValidation)
		case "NETWORK_ERROR":
			os.Exit(errors.ExitNetwork)
		case "INVALID_ARGS":
			os.Exit(errors.ExitInvalidArgs)
		default:
			os.Exit(errors.ExitError)
		}
	}
	os.Exit(errors.ExitError)
}

// ExitCode returns the appropriate exit code for this response.
func (r *Response) ExitCode() int {
	if r.Success {
		return errors.ExitSuccess
	}
	if r.Error != nil {
		switch r.Error.Code {
		case "AUTH_ERROR":
			return errors.ExitAuthFailure
		case "FORBIDDEN":
			return errors.ExitForbidden
		case "NOT_FOUND":
			return errors.ExitNotFound
		case "VALIDATION_ERROR":
			return errors.ExitValidation
		case "NETWORK_ERROR":
			return errors.ExitNetwork
		case "INVALID_ARGS":
			return errors.ExitInvalidArgs
		}
	}
	return errors.ExitError
}
