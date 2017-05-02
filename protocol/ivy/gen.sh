#!/bin/sh

pigeon grammar.peg | goimports > grammar.go
