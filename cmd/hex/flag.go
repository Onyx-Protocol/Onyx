package main

import (
	"flag"
	"fmt"
	"strconv"
)

// notBoolValue is like flag.boolValue but it inverts the value.
type notBoolValue bool

func flagNotBoolVar(p *bool, name string, value bool, usage string) {
	flag.CommandLine.Var(newNotBoolValue(value, p), name, usage)
}

func newNotBoolValue(val bool, p *bool) *notBoolValue {
	*p = !val
	return (*notBoolValue)(p)
}

func (b *notBoolValue) Set(s string) error {
	v, err := strconv.ParseBool(s)
	*b = notBoolValue(!v)
	return err
}

func (b *notBoolValue) Get() interface{} { return bool(*b) }

func (b *notBoolValue) String() string { return fmt.Sprintf("%v", !*b) }

func (b *notBoolValue) IsBoolFlag() bool { return true }
