package env

import (
	"net/url"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestInt(t *testing.T) {
	result := Int("nonexistent", 15)
	Parse()

	if *result != 15 {
		t.Fatalf("expected result=15, got result=%d", *result)
	}

	err := os.Setenv("int-key", "25")
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	result = Int("int-key", 15)
	Parse()

	if *result != 25 {
		t.Fatalf("expected result=25, got result=%d", *result)
	}
}

func TestIntVar(t *testing.T) {
	var result int
	IntVar(&result, "nonexistent", 15)
	Parse()

	if result != 15 {
		t.Fatalf("expected result=15, got result=%d", result)
	}

	err := os.Setenv("int-key", "25")
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	IntVar(&result, "int-key", 15)
	Parse()

	if result != 25 {
		t.Fatalf("expected result=25, got result=%d", result)
	}
}

func TestBool(t *testing.T) {
	result := Bool("nonexistent", true)
	Parse()

	if *result != true {
		t.Fatalf("expected result=true, got result=%t", *result)
	}

	err := os.Setenv("bool-key", "true")
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	result = Bool("bool-key", false)
	Parse()

	if *result != true {
		t.Fatalf("expected result=true, got result=%t", *result)
	}
}

func TestBoolVar(t *testing.T) {
	var result bool
	BoolVar(&result, "nonexistent", true)
	Parse()

	if result != true {
		t.Fatalf("expected result=true, got result=%t", result)
	}

	err := os.Setenv("bool-key", "true")
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	BoolVar(&result, "bool-key", false)
	Parse()

	if result != true {
		t.Fatalf("expected result=true, got result=%t", true)
	}
}

func TestDuration(t *testing.T) {
	result := Duration("nonexistent", 15*time.Second)

	if result != 15*time.Second {
		t.Fatalf("expected result=15s, got result=%v", result)
	}

	err := os.Setenv("duration-key", "25s")
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	result = Duration("duration-key", 15*time.Second)

	if result != 25*time.Second {
		t.Fatalf("expected result=25s, got result=%v", result)
	}
}

func TestURL(t *testing.T) {
	example := "http://example.com"
	newExample := "http://something-new.com"
	result := URL("nonexistent", example)
	Parse()

	if result.String() != example {
		t.Fatalf("expected result=%s, got result=%v", example, result)
	}

	err := os.Setenv("url-key", newExample)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	result = URL("url-key", example)
	Parse()

	if result.String() != newExample {
		t.Fatalf("expected result=%v, got result=%v", newExample, result)
	}
}

func TestURLVar(t *testing.T) {
	example := "http://example.com"
	newExample := "http://something-new.com"
	var result url.URL
	URLVar(&result, "nonexistent", example)
	Parse()

	if result.String() != example {
		t.Fatalf("expected result=%s, got result=%v", example, result)
	}

	err := os.Setenv("url-key", newExample)
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	URLVar(&result, "url-key", example)
	Parse()

	if result.String() != newExample {
		t.Fatalf("expected result=%v, got result=%v", newExample, result)
	}
}

func TestString(t *testing.T) {
	result := String("nonexistent", "default")
	Parse()

	if *result != "default" {
		t.Fatalf("expected result=default, got result=%s", *result)
	}

	err := os.Setenv("string-key", "something-new")
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	result = String("string-key", "default")
	Parse()

	if *result != "something-new" {
		t.Fatalf("expected result=something-new, got result=%s", *result)
	}
}

func TestStringVar(t *testing.T) {
	var result string
	StringVar(&result, "nonexistent", "default")
	Parse()

	if result != "default" {
		t.Fatalf("expected result=default, got result=%s", result)
	}

	err := os.Setenv("string-key", "something-new")
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	StringVar(&result, "string-key", "default")
	Parse()

	if result != "something-new" {
		t.Fatalf("expected result=something-new, got result=%s", result)
	}
}

func TestStringSlice(t *testing.T) {
	result := StringSlice("empty", "hi")
	Parse()

	exp := []string{"hi"}
	if !reflect.DeepEqual(exp, *result) {
		t.Fatalf("expected %v, got %v", exp, *result)
	}

	err := os.Setenv("string-slice-key", "hello,world")
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	result = StringSlice("string-slice-key", "hi")
	Parse()

	exp = []string{"hello", "world"}
	if !reflect.DeepEqual(exp, *result) {
		t.Fatalf("expected %v, got %v", exp, *result)
	}
}

func TestStringSliceVar(t *testing.T) {
	var result []string
	StringSliceVar(&result, "empty", "hi")
	Parse()

	exp := []string{"hi"}
	if !reflect.DeepEqual(exp, result) {
		t.Fatalf("expected %v, got %v", exp, result)
	}

	err := os.Setenv("string-slice-key", "hello,world")
	if err != nil {
		t.Fatal("unexpected error", err)
	}

	StringSliceVar(&result, "string-slice-key", "hi", "there")
	Parse()

	exp = []string{"hello", "world"}
	if !reflect.DeepEqual(exp, result) {
		t.Fatalf("expected %v, got %v", exp, result)
	}
}
