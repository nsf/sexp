package sexp

import (
	"fmt"
)

// This error structure is Parse* functions family specific, it returns information
// about errors encountered during parsing. Location can be decoded using the
// context you passed in as an argument. If the context was nil, then the location
// is simply a byte offset from the beginning of the input stream.
type Error struct {
	Location SourceLoc
	message  string
}

// Satisfy the built-in error interface. Returns the error message (without
// source location).
func (e *Error) Error() string {
	return e.message
}

func new_error(location SourceLoc, format string, args ...interface{}) *Error {
	return &Error{
		Location: location,
		message:  fmt.Sprintf(format, args...),
	}
}

func panic_if_error(err error) {
	if err != nil {
		panic(err)
	}
}

func number_suffix(n int) string {
	if n >= 10 && n <= 20 {
		return "th"
	}
	switch n % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	}
	return "th"
}

func the_list_has_n_children(n int) string {
	switch n {
	case 0:
		return "the list has no children"
	case 1:
		return "the list has 1 child only"
	}
	return fmt.Sprintf("the list has %d children only", n)
}

func the_list_has_n_siblings(n int) string {
	switch n {
	case 0:
		return "the list has no siblings"
	case 1:
		return "the list has 1 sibling only"
	}
	return fmt.Sprintf("the list has %d siblings only", n)
}
