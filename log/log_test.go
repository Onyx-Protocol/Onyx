package log

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"chain/errors"
)

func TestSetOutput(t *testing.T) {
	var buf bytes.Buffer
	want := "foobar"
	SetOutput(&buf)
	Printf(context.Background(), want)
	SetOutput(os.Stdout)
	got := buf.String()
	if !strings.Contains(got, want) {
		t.Errorf("log = %q; should contain %q", got, want)
	}
}

func TestNoExtraFormatDirectives(t *testing.T) {
	buf := new(bytes.Buffer)
	SetOutput(buf)
	SetPrefix("foo", "bar")
	Printkv(context.Background(), "baz", 1)
	SetOutput(os.Stdout)
	got := buf.String()
	if strings.Contains(got, "%") {
		t.Errorf("log line appears to contain format directive: %q", got)
	}
}

func TestPrefix(t *testing.T) {
	buf := new(bytes.Buffer)
	SetOutput(buf)
	SetPrefix("foo", "bar")
	Printkv(context.Background(), "baz", 1)
	SetOutput(os.Stdout)

	got := buf.String()
	wantPrefix := "foo=bar "
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("output = %q want prefix %q", got, wantPrefix)
	}

	SetPrefix()
	if procPrefix != nil {
		t.Errorf("procPrefix = %q want nil", procPrefix)
	}
}

func TestAddPrefixkv0(t *testing.T) {
	got := prefix(context.Background())
	var want []byte = nil
	if !reflect.DeepEqual(got, want) {
		t.Errorf(`prefix(context.Background()) = %#v want %#v`, got, want)
	}
}

func TestAddPrefixkv1(t *testing.T) {
	ctx := context.Background()
	ctx1 := AddPrefixkv(ctx, "a", "b")
	got := prefix(ctx1)
	want := []byte("a=b ")
	if !bytes.Equal(got, want) {
		t.Errorf(`prefix(AddPrefixkv(bg, "a", "b")) = %q want %q`, got, want)
	}
}

func TestAddPrefixkv2(t *testing.T) {
	ctx := context.Background()
	ctx1 := AddPrefixkv(ctx, "a", "b")
	ctx2 := AddPrefixkv(ctx1, "c", "d")
	got := prefix(ctx2)
	want := []byte("a=b c=d ")
	if !bytes.Equal(got, want) {
		t.Errorf(`prefix(AddPrefixkv(AddPrefixkv(bg, "a", "b"), "c", "d")) = %q want %q`, got, want)
	}
}

func TestAddPrefixkvAppendTwice(t *testing.T) {
	ctx := context.Background()
	ctx1 := AddPrefixkv(ctx, "a", "b")
	ctx2a := AddPrefixkv(ctx1, "c", "d")
	ctx2b := AddPrefixkv(ctx1, "e", "f")
	gota := prefix(ctx2a)
	wanta := []byte("a=b c=d ")
	if !bytes.Equal(gota, wanta) {
		t.Errorf(`prefix(AddPrefixkv(AddPrefixkv(bg, "a", "b"), "c", "d")) = %q want %q`, gota, wanta)
	}
	gotb := prefix(ctx2b)
	wantb := []byte("a=b e=f ")
	if !bytes.Equal(gotb, wantb) {
		t.Errorf(`prefix(AddPrefixkv(AddPrefixkv(bg, "a", "b"), "e", "f")) = %q want %q`, gotb, wantb)
	}
}

func TestPrintkv(t *testing.T) {
	examples := []struct {
		keyvals []interface{}
		want    []string
	}{
		// Basic example
		{
			keyvals: []interface{}{"msg", "hello world"},
			want: []string{
				"at=log_test.go:",
				"t=",
				`msg="hello world"`,
			},
		},

		// Duplicate keys
		{
			keyvals: []interface{}{"msg", "hello world", "msg", "goodbye world"},
			want: []string{
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
				"at=log_test.go:",
				"t=",
			},
		},

		// Odd number of log params
		{
			keyvals: []interface{}{"k1", "v1", "k2"},
			want: []string{
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
		SetOutput(buf)

		Printkv(context.Background(), ex.keyvals...)

		read, err := ioutil.ReadAll(buf)
		if err != nil {
			SetOutput(os.Stdout)
			t.Fatal("read buffer error:", err)
		}

		got := string(read)

		for _, w := range ex.want {
			if !strings.Contains(got, w) {
				t.Errorf("Result did not contain string:\ngot:  %s\nwant: %s", got, w)
			}
		}
		if !strings.HasSuffix(got, "\n") {
			t.Errorf("log output should end with a newline")
		}

		SetOutput(os.Stdout)
	}
}

func TestMessagef(t *testing.T) {
	buf := new(bytes.Buffer)
	SetOutput(buf)
	defer SetOutput(os.Stdout)

	Printf(context.Background(), "test round %d", 0)

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

func TestPrintkvStack(t *testing.T) {
	buf := new(bytes.Buffer)
	SetOutput(buf)
	defer SetOutput(os.Stdout)

	root := errors.New("boo")
	wrapped := errors.Wrap(root)
	Printkv(context.Background(), KeyError, wrapped)

	read, err := ioutil.ReadAll(buf)
	if err != nil {
		t.Fatal("read buffer error:", err)
	}

	got := string(read)
	want := []string{
		"at=log_test.go:",
		"error=boo",

		// stack trace
		"TestPrintkvStack\n",
		"/go",
	}

	t.Logf("output:\n%s", got)
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("output %q did not contain %q", got, w)
		}
	}
}

func TestError(t *testing.T) {
	buf := new(bytes.Buffer)
	SetOutput(buf)
	defer SetOutput(os.Stdout)

	root := errors.New("boo")
	wrapped := errors.Wrap(root)
	Error(context.Background(), wrapped, "failure x ", 0)

	read, err := ioutil.ReadAll(buf)
	if err != nil {
		t.Fatal("read buffer error:", err)
	}

	got := string(read)
	want := []string{
		"at=log_test.go:",
		`error="failure x 0: boo"`,

		// stack trace
		"TestError\n",
		"/go",
	}

	t.Logf("output:\n%s", got)
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("output %q did not contain %q", got, w)
		}
	}
}

func TestRawStack(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	defer SetOutput(os.Stdout)

	stack := []byte("this\nis\na\nraw\nstack")
	Printkv(context.Background(), "message", "foo", "stack", stack)

	got := buf.String()
	if !strings.HasSuffix(got, "\n"+string(stack)+"\n") {
		t.Logf("output:\n%s", got)
		t.Errorf("output did not contain %q", stack)
	}
}

func TestIsStackVal(t *testing.T) {
	cases := []struct {
		v interface{}
		w bool
	}{
		{[]byte("foo"), true},
		{new(runtime.Frames), true},
		{"line1", false},
		{[...]byte{'x'}, false},
		{[]string{}, false},
	}
	for _, test := range cases {
		if g := isStackVal(test.v); g != test.w {
			t.Errorf("isStackVal(%#v) = %v want %v", test.v, g, test.w)
		}
	}
}

func TestWriteRawStack(t *testing.T) {
	pc := make([]uintptr, 10)
	n := runtime.Callers(1, pc)
	if n < 2 {
		t.Fatalf("runtime.Callers failed")
	}

	cases := []struct {
		v interface{}
		w string
	}{
		{[]byte("foo\nbar"), "foo\nbar\n"},
		{runtime.CallersFrames(pc[:2]), ": chain/log.TestWriteRawStack\n"},
		{1, ""}, // int is not a valid stack val
	}

	for _, test := range cases {
		var buf bytes.Buffer
		writeRawStack(&buf, test.v)
		if g := buf.String(); !strings.Contains(g, test.w) {
			t.Errorf("writeRawStack(%#v) = %q, must contain %q", test.v, g, test.w)
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
