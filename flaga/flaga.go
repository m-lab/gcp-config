package flaga

import (
	"fmt"
	"strings"
)

// Strings accepts comma separated values, and tracks whether the flag was provided.
type Strings struct {
	Values   []string
	Assigned bool
}

// Get retrieves the value contained in the flag.
func (sa Strings) Get() []string {
	return sa.Values
}

// Set accepts a string parameter, attempts to split it and then appends all
// values to the associated Strings.Values.
func (sa *Strings) Set(s string) error {
	sa.Assigned = true
	f := strings.Split(s, ",")
	(*sa).Values = append((*sa).Values, f...)
	return nil
}

// String reports the Strings as a Go value.
func (sa Strings) String() string {
	return fmt.Sprintf("%#v", sa.Get())
}

// Contains returns true when the given val equals one of the included values.
func (sa Strings) Contains(val string) bool {
	for _, value := range sa.Values {
		if value == val {
			return true
		}
	}
	return false
}

// String contains a single value and whether the value was assigned.
type String struct {
	Value    string
	Assigned bool
}

// Get retrieves the value contained in the flag.
func (e String) Get() string {
	return e.Value
}

// Set accepts a string parameter, sets the Value and indicates that  is true.
func (e *String) Set(s string) error {
	e.Assigned = true
	e.Value = s
	return nil
}

// String reports the String as a Go value.
func (e String) String() string {
	return e.Get()
}
