package errors_test

import (
	"fmt"

	"chain/errors"
)

var ErrInvalidKey = errors.New("invalid key")

func ExampleSub() error {
	sig, err := sign()
	if err != nil {
		return errors.Sub(ErrInvalidKey, err)
	}
	fmt.Println(sig)
	return nil
}

func ExampleSub_return() ([]byte, error) {
	sig, err := sign()
	return sig, errors.Sub(ErrInvalidKey, err)
}

func sign() ([]byte, error) { return nil, nil }
