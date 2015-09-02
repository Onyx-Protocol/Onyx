package pg

import "errors"

// ErrUserInputNotFound indicates that a query returned no results.
// It is equivalent to sql.ErrNoRows, except that ErrUserInputNotFound
// also indicates the query was based on user-provided parameters,
// and the lack of results should be communicated back to the user.
//
// In contrast, we use sql.ErrNoRows to represent an internal error;
// this indicates a bug in our code
// and only a generic "internal error" message
// should be communicated back to the user.
var ErrUserInputNotFound = errors.New("pg: user input not found")
