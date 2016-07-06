package blockchain

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestReadBytesMax(t *testing.T) {
	var buf bytes.Buffer
	_, err := WriteUvarint(&buf, math.MaxUint32)
	if err != nil {
		t.Fatal(err)
	}

	want := fmt.Errorf("cannot read %d bytes; max is %d", math.MaxUint32, 10)
	_, err = ReadBytes(&buf, 10)
	if !reflect.DeepEqual(err, want) {
		t.Fatalf("err: got=%#v, want=%#v", err, want)
	}
}
