package httperror

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"chain/log"
)

func TestLogSkip(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)

	formatter := Formatter{
		Default:     Info{500, "CH000", "Internal server error"},
		IsTemporary: func(Info, error) bool { return false },
		Errors:      map[error]Info{},
	}
	formatter.Log(context.Background(), errors.New("an unmapped error"))

	logStr := string(buf.Bytes())
	if len(logStr) == 0 {
		t.Error("expected error to be logged")
	}
	if strings.Contains(logStr, "at=httperror.go") {
		t.Errorf("expected httperror stack frames to be skipped but got:\n%s", logStr)
	}
	if !strings.Contains(logStr, "status=500") {
		t.Errorf("expected status code of default error info but got:\n%s", logStr)
	}
	t.Log(logStr)
}
