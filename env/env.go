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
	return p
}

// Bool returns a new bool pointer.
// When Parse is called,
// env var name will be parsed
// and the resulting value
// will be assigned to the returned location.
// Parsing uses strconv.ParseBool.
func Bool(name string, value bool) *bool {
	p := new(bool)
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
	return p
}

// Duration returns a new time.Duration pointer.
// When Parse is called,
// env var name will be parsed
// and the resulting value
// will be assigned to the returned location.
func Duration(name string, value time.Duration) *time.Duration {
	p := new(time.Duration)
	*p = value
	funcs = append(funcs, func() bool {
		if s := os.Getenv(name); s != "" {
			v, err := time.ParseDuration(s)
			if err != nil {
				log.Println(name, err)
				return false
			}
			*p = v
		}
		return true
	})
	return p
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
	return p
}

// String returns a new string pointer.
// When Parse is called,
// env var name will be assigned
// to the returned location.
func String(name string, value string) *string {
	p := new(string)
	*p = value
	funcs = append(funcs, func() bool {
		if s := os.Getenv(name); s != "" {
			*p = s
		}
		return true
	})
	return p
}

// StringSlice returns a pointer to a slice
// of strings. It expects env var name to
// be a list of items delimited by commas.
// If env var name is missing, StringSlice
// returns a pointer to an empty slice.
func StringSlice(name string) *[]string {
	p := new([]string)
	funcs = append(funcs, func() bool {
		if s := os.Getenv(name); s != "" {
			a := strings.Split(s, ",")
			log.Println(a)
			*p = a
		}
		return true
	})
	return p
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
