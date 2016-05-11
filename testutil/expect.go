package testutil

import (
	"reflect"
	"runtime"
	"testing"

	"chain/cos/txscript"
	"chain/errors"
)

func ExpectEqual(t testing.TB, actual, expected interface{}, msg string) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%s: got %v, expected %v\n%s", msg, actual, expected, stackTrace())
	}
}

func ExpectScriptEqual(t testing.TB, actual, expected []byte, msg string) {
	if !reflect.DeepEqual(expected, actual) {
		expectedStr, _ := txscript.DisasmString(expected)
		actualStr, _ := txscript.DisasmString(actual)
		t.Errorf("%s: got [%s], expected [%s]\n%s", msg, actualStr, expectedStr, stackTrace())
	}
}

func ExpectError(t testing.TB, expected error, msg string, fn func() error) {
	actual := fn()
	if expected != errors.Root(actual) {
		t.Errorf("%s: got error %v, expected %v\n%s", msg, actual, expected, stackTrace())
	}
}

func FatalErr(t testing.TB, err error) {
	args := []interface{}{err}
	for _, frame := range errors.Stack(err) {
		args = append(args, "\n"+frame.String())
	}
	t.Fatal(args...)
}

func stackTrace() []byte {
	buf := make([]byte, 16384)
	len := runtime.Stack(buf, false)
	return buf[:len]
}
