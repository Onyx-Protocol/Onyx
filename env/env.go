// Package env provides a convenient way to convert environment
// variables into Go data. It is similar in design to package
// flag.
package env

import (
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var funcs []func() bool

// Int returns a new int pointer.
// When Parse is called,
// env var name will be parsed
// and the resulting value
// will be assigned to the returned location.
func Int(name string, value int) *int {
	p := new(int)
	IntVar(p, name, value)
	return p
}

// IntVar defines an int var with the specified
// name and default value. The argument p points
// to an int variable in which to store the
// value of the environment var.
func IntVar(p *int, name string, value int) {
	*p = value
	funcs = append(funcs, func() bool {
		if s := os.Getenv(name); s != "" {
			v, err := strconv.Atoi(s)
			if err != nil {
				log.Println(name, err)
				return false
			}
			*p = v
		}
		return true
	})
}

// Bool returns a new bool pointer.
// When Parse is called,
// env var name will be parsed
// and the resulting value
// will be assigned to the returned location.
// Parsing uses strconv.ParseBool.
func Bool(name string, value bool) *bool {
	p := new(bool)
	BoolVar(p, name, value)
	return p
}

// BoolVar defines a bool var with the specified
// name and default value. The argument p points
// to a bool variable in which to store the value
// of the environment variable.
func BoolVar(p *bool, name string, value bool) {
	*p = value
	funcs = append(funcs, func() bool {
		if s := os.Getenv(name); s != "" {
			v, err := strconv.ParseBool(s)
			if err != nil {
				log.Println(name, err)
				return false
			}
			*p = v
		}
		return true
	})
}

// Duration returns the value of the named environment variable,
// interpreted as a time.Duration (using time.ParseDuration).
// If there is an error parsing the value, it prints a
// diagnostic message to the log and calls os.Exit(1).
// If name isn't in the environment, it returns value.
func Duration(name string, value time.Duration) time.Duration {
	if s := os.Getenv(name); s != "" {
		var err error
		value, err = time.ParseDuration(s)
		if err != nil {
			log.Println(name, err)
			os.Exit(1)
		}
	}
	return value
}

// URL returns a new url.URL pointer.
// When Parse is called,
// env var name will be parsed
// and the resulting value
// will be assigned to the returned location.
// URL panics if there is an error parsing
// the given default value.
func URL(name string, value string) *url.URL {
	p := new(url.URL)
	URLVar(p, name, value)
	return p
}

// URLVar defines a url.URL variable with
// the specified name ande default value.
// The argument p points to a url.URL variable
// in which to store the value of the environment
// variable.
func URLVar(p *url.URL, name string, value string) {
	v, err := url.Parse(value)
	if err != nil {
		panic(err)
	}
	*p = *v
	funcs = append(funcs, func() bool {
		if s := os.Getenv(name); s != "" {
			v, err := url.Parse(s)
			if err != nil {
				log.Println(name, err)
				return false
			}
			*p = *v
		}
		return true
	})
}

// String returns a new string pointer.
// When Parse is called,
// env var name will be assigned
// to the returned location.
func String(name string, value string) *string {
	p := new(string)
	StringVar(p, name, value)
	return p
}

// StringVar defines a string with the
// specified name and default value. The
// argument p points to a string variable in
// which to store the value of the environment
// var.
func StringVar(p *string, name string, value string) {
	*p = value
	funcs = append(funcs, func() bool {
		if s := os.Getenv(name); s != "" {
			*p = s
		}
		return true
	})
}

// StringSlice returns a pointer to a slice
// of strings. It expects env var name to
// be a list of items delimited by commas.
// If env var name is missing, StringSlice
// returns a pointer to a slice of the value
// strings.
func StringSlice(name string, value ...string) *[]string {
	p := new([]string)
	StringSliceVar(p, name, value...)
	return p
}

// StringSliceVar defines a new string slice
// with the specified name. The argument p
// points to a string slice variable in which
// to store the value of the environment var.
func StringSliceVar(p *[]string, name string, value ...string) {
	*p = value
	funcs = append(funcs, func() bool {
		if s := os.Getenv(name); s != "" {
			a := strings.Split(s, ",")
			*p = a
		}
		return true
	})
}

// Parse parses known env vars
// and assigns the values to the variables
// that were previously registered.
// If any values cannot be parsed,
// Parse prints an error message for each one
// and exits the process with status 1.
func Parse() {
	ok := true
	for _, f := range funcs {
		ok = f() && ok
	}
	if !ok {
		os.Exit(1)
	}
}
