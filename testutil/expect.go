package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"chain/errors"
)

var wd, _ = os.Getwd()

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
