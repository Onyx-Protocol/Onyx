package reqid

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"chain/log"
)

func TestPrintkvRequestID(t *testing.T) {
	buf := new(bytes.Buffer)
	log.SetOutput(buf)
	defer log.SetOutput(os.Stdout)

	log.Printkv(NewContext(context.Background(), "example-request-id"))

	got := buf.String()
	want := "reqid=example-request-id"
	if !strings.Contains(got, want) {
		t.Errorf("Result did not contain string:\ngot:  %s\nwant: %s", got, want)
	}
}
