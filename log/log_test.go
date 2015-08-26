package log

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"golang.org/x/net/context"

	"chain/net/http/reqid"
)

func setTestLogWriter(w io.Writer) func() {
	logWriterMu.Lock()
	old := logWriter
	logWriter = w
	logWriterMu.Unlock()

	return func() {
		logWriterMu.Lock()
		logWriter = old
		logWriterMu.Unlock()
	}
}

func TestWrite(t *testing.T) {
	examples := []struct {
		keyvals []interface{}
		want    []string
	}{
		// Basic example
		{
			keyvals: []interface{}{"msg", "hello world"},
			want: []string{
				"reqid=unknown_req_id",
				"at=log_test.go:",
				"t=",
				`msg="hello world"`,
			},
		},

		// Duplicate keys
		{
			keyvals: []interface{}{"msg", "hello world", "msg", "goodbye world"},
			want: []string{
				"reqid=unknown_req_id",
				"at=log_test.go:",
				"t=",
				`msg="hello world"`,
				`msg="goodbye world"`,
			},
		},

		// Zero log params
		{
			keyvals: nil,
			want: []string{
				"reqid=unknown_req_id",
				"at=log_test.go:",
				"t=",
			},
		},

		// Odd number of log params
		{
			keyvals: []interface{}{"k1", "v1", "k2"},
			want: []string{
				"reqid=unknown_req_id",
				"at=log_test.go:",
				"t=",
				"k1=v1",
				"k2=",
				`log-error="odd number of log params"`,
			},
		},
	}

	for i, ex := range examples {
		t.Log("Example", i)

		buf := new(bytes.Buffer)
		reset := setTestLogWriter(buf)

		Write(context.Background(), ex.keyvals...)

		read, err := ioutil.ReadAll(buf)
		if err != nil {
			t.Fatal("read buffer error:", err)
		}

		got := string(read)

		for _, w := range ex.want {
			if !strings.Contains(got, w) {
				t.Errorf("Result did not contain string:\ngot:  %s\nwant: %s", got, w)
			}
		}

		reset()
	}
}

func TestWriteRequestID(t *testing.T) {
	buf := new(bytes.Buffer)
	reset := setTestLogWriter(buf)
	defer reset()

	Write(reqid.NewContext(context.Background(), "example-request-id"))

	read, err := ioutil.ReadAll(buf)
	if err != nil {
		t.Fatal("read buffer error:", err)
	}

	got := string(read)
	want := "reqid=example-request-id"

	if !strings.Contains(got, want) {
		t.Errorf("Result did not contain string:\ngot:  %s\nwant: %s", got, want)
	}
}

func TestMessagef(t *testing.T) {
	buf := new(bytes.Buffer)
	reset := setTestLogWriter(buf)
	defer reset()

	Messagef(context.Background(), "test round %d", 0)

	read, err := ioutil.ReadAll(buf)
	if err != nil {
		t.Fatal("read buffer error:", err)
	}

	got := string(read)
	want := []string{
		"at=log_test.go:",
		`message="test round 0"`,
	}

	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("Result did not contain string:\ngot:  %s\nwant: %s", got, w)
		}
	}
}

func TestError(t *testing.T) {
	buf := new(bytes.Buffer)
	reset := setTestLogWriter(buf)
	defer reset()

	Error(context.Background(), errors.New("boo"), "failure x ", 0)

	read, err := ioutil.ReadAll(buf)
	if err != nil {
		t.Fatal("read buffer error:", err)
	}

	got := string(read)
	want := []string{
		"at=log_test.go:",
		`error="failure x 0: boo"`,
	}

	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("Result did not contain string:\ngot:  %s\nwant: %s", got, w)
		}
	}
}

func TestFormatKey(t *testing.T) {
	examples := []struct {
		key  interface{}
		want string
	}{
		{"hello", "hello"},
		{"hello world", "hello-world"},
		{"", "?"},
		{true, "true"},
		{"a b\"c\nd;e\tf龜g", "a-b-c-d-e-f龜g"},
	}

	for i, ex := range examples {
		t.Log("Example", i)
		got := formatKey(ex.key)
		if got != ex.want {
			t.Errorf("formatKey(%#v) = %q want %q", ex.key, got, ex.want)
		}
	}
}

func TestFormatValue(t *testing.T) {
	examples := []struct {
		value interface{}
		want  string
	}{
		{"hello", "hello"},
		{"hello world", `"hello world"`},
		{1.5, "1.5"},
		{true, "true"},
		{errors.New("this is an error"), `"this is an error"`},
		{[]byte{'a', 'b', 'c'}, `"[97 98 99]"`},
		{bytes.NewBuffer([]byte{'a', 'b', 'c'}), "abc"},
		{"a b\"c\nd;e\tf龜g", `"a b\"c\nd;e\tf龜g"`},
	}

	for i, ex := range examples {
		t.Log("Example", i)
		got := formatValue(ex.value)
		if got != ex.want {
			t.Errorf("formatKey(%#v) = %q want %q", ex.value, got, ex.want)
		}
	}
}
