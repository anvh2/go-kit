package errors

import (
	"bytes"
	"strings"
)

// Errors is a type of []error
// This is used to pass multiple errors when using parallel or concurrent methods
// and yet subscribe to the error interface
type Errors []error

// Prints all errors from asynchronous tasks separated by space
func (e Errors) Error() string {
	b := bytes.NewBufferString("")

	for _, err := range e {
		b.WriteString(err.Error())
		b.WriteString(" ")
	}

	return strings.TrimSpace(b.String())
}

func (e Errors) Final() error {
	if len(e) == 0 {
		return nil
	}
	return e
}
