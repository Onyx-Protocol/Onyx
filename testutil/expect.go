package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"chain/cos/vm"
	"chain/errors"
)

var wd, _ = os.Getwd()

func ExpectEqual(t testing.TB, actual, expected interface{}, msg string) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%s: got %v, expected %v\n%s", msg, actual, expected, stackTrace())
	}
}

func ExpectScriptEqual(t testing.TB, actual, expected []byte, msg string) {
	if !reflect.DeepEqual(expected, actual) {
		expectedStr, _ := vm.Decompile(expected)
		actualStr, _ := vm.Decompile(actual)
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
		file := frame.File
		if rel, err := filepath.Rel(wd, file); err == nil && !strings.HasPrefix(rel, "../") {
			file = rel
		}
		funcname := frame.Func[strings.IndexByte(frame.Func, '.')+1:]
		s := fmt.Sprintf("\n%s:%d: %s", file, frame.Line, funcname)
		args = append(args, s)
	}
	t.Fatal(args...)
}

func stackTrace() []byte {
	buf := make([]byte, 16384)
	len := runtime.Stack(buf, false)
	return buf[:len]
}
