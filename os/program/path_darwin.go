package program

import "errors"

func path() (name string, err error) {
	return "", errors.New("program: Path not implemented on darwin")
}
