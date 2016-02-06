package testutil

import (
	"reflect"
	"runtime"
	"testing"

	"chain/errors"
	"chain/fedchain/txscript"
)

func ExpectEqual(t *testing.T, actual, expected interface{}, msg string) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%s: got %v, expected %v\n%s", msg, actual, expected, stackTrace())
	}
}

func ExpectScriptEqual(t *testing.T, actual, expected []byte, msg string) {
	if !reflect.DeepEqual(expected, actual) {
		expectedStr, _ := txscript.DisasmString(expected)
		actualStr, _ := txscript.DisasmString(actual)
		t.Errorf("%s: got [%s], expected [%s]\n%s", msg, actualStr, expectedStr, stackTrace())
	}
}

func ExpectError(t *testing.T, expected error, msg string, fn func() error) {
	actual := fn()
	if expected != errors.Root(actual) {
		t.Errorf("%s: got error %v, expected %v\n%s", msg, actual, expected, stackTrace())
	}
}

func FatalErr(t *testing.T, err error) {
	t.Log(err)
	for _, frame := range errors.Stack(err) {
		t.Log(frame)
	}
	t.FailNow()
}

func stackTrace() []byte {
	buf := make([]byte, 16384)
	len := runtime.Stack(buf, false)
	return buf[:len]
}
