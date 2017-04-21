package vm

import (
	"fmt"
	"testing"

	"chain/errors"
	"chain/math/checked"
	"chain/testutil"
)

func TestParseOp(t *testing.T) {
	cases := []struct {
		prog    []byte
		pc      uint32
		want    Instruction
		wantErr error
	}{{
		prog: []byte{byte(OP_ADD)},
		want: Instruction{Op: OP_ADD, Len: 1},
	}, {
		prog: []byte{byte(OP_16)},
		want: Instruction{Op: OP_16, Data: []byte{16}, Len: 1},
	}, {
		prog: []byte{byte(OP_DATA_5), 1, 1, 1, 1, 1},
		want: Instruction{Op: OP_DATA_5, Data: []byte{1, 1, 1, 1, 1}, Len: 6},
	}, {
		prog: []byte{byte(OP_DATA_5), 1, 1, 1, 1, 1, 255},
		want: Instruction{Op: OP_DATA_5, Data: []byte{1, 1, 1, 1, 1}, Len: 6},
	}, {
		prog: []byte{byte(OP_PUSHDATA1), 1, 1},
		want: Instruction{Op: OP_PUSHDATA1, Data: []byte{1}, Len: 3},
	}, {
		prog: []byte{byte(OP_PUSHDATA1), 1, 1, 255},
		want: Instruction{Op: OP_PUSHDATA1, Data: []byte{1}, Len: 3},
	}, {
		prog: []byte{byte(OP_PUSHDATA2), 1, 0, 1},
		want: Instruction{Op: OP_PUSHDATA2, Data: []byte{1}, Len: 4},
	}, {
		prog: []byte{byte(OP_PUSHDATA2), 1, 0, 1, 255},
		want: Instruction{Op: OP_PUSHDATA2, Data: []byte{1}, Len: 4},
	}, {
		prog: []byte{byte(OP_PUSHDATA4), 1, 0, 0, 0, 1},
		want: Instruction{Op: OP_PUSHDATA4, Data: []byte{1}, Len: 6},
	}, {
		prog: []byte{byte(OP_PUSHDATA4), 1, 0, 0, 0, 1, 255},
		want: Instruction{Op: OP_PUSHDATA4, Data: []byte{1}, Len: 6},
	}, {
		prog:    []byte{},
		wantErr: ErrShortProgram,
	}, {
		prog:    []byte{byte(OP_0)},
		pc:      1,
		wantErr: ErrShortProgram,
	}, {
		prog:    []byte{byte(OP_DATA_1)},
		wantErr: ErrShortProgram,
	}, {
		prog:    []byte{byte(OP_PUSHDATA1)},
		wantErr: ErrShortProgram,
	}, {
		prog:    []byte{byte(OP_PUSHDATA1), 1},
		wantErr: ErrShortProgram,
	}, {
		prog:    []byte{byte(OP_PUSHDATA2)},
		wantErr: ErrShortProgram,
	}, {
		prog:    []byte{byte(OP_PUSHDATA2), 1, 0},
		wantErr: ErrShortProgram,
	}, {
		prog:    []byte{byte(OP_PUSHDATA4)},
		wantErr: ErrShortProgram,
	}, {
		prog:    []byte{byte(OP_PUSHDATA4), 1, 0, 0, 0},
		wantErr: ErrShortProgram,
	}, {
		pc:      71,
		prog:    []byte{0x6d, 0x6b, 0xaa, 0x20, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x20, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x0, 0x0, 0x4e, 0xff, 0xff, 0xff, 0xff, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30},
		wantErr: checked.ErrOverflow,
	}}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%d: %x", c.pc, c.prog), func(t *testing.T) {
			got, gotErr := ParseOp(c.prog, c.pc)

			if errors.Root(gotErr) != c.wantErr {
				t.Errorf("ParseOp(%x, %d) error = %v want %v", c.prog, c.pc, gotErr, c.wantErr)
			}

			if c.wantErr != nil {
				return
			}

			if !testutil.DeepEqual(got, c.want) {
				t.Errorf("ParseOp(%x, %d) = %+v want %+v", c.prog, c.pc, got, c.want)
			}
		})
	}
}

func TestParseProgram(t *testing.T) {
	cases := []struct {
		prog    []byte
		want    []Instruction
		wantErr error
	}{
		{
			prog: []byte{byte(OP_2), byte(OP_3), byte(OP_ADD), byte(OP_5), byte(OP_NUMEQUAL)},
			want: []Instruction{
				{Op: OP_2, Data: []byte{0x02}, Len: 1},
				{Op: OP_3, Data: []byte{0x03}, Len: 1},
				{Op: OP_ADD, Len: 1},
				{Op: OP_5, Data: []byte{0x05}, Len: 1},
				{Op: OP_NUMEQUAL, Len: 1},
			},
		},
		{
			prog: []byte{255},
			want: []Instruction{
				{Op: 255, Len: 1},
			},
		},
	}

	for _, c := range cases {
		got, gotErr := ParseProgram(c.prog)

		if errors.Root(gotErr) != c.wantErr {
			t.Errorf("ParseProgram(%x) error = %v want %v", c.prog, gotErr, c.wantErr)
		}

		if c.wantErr != nil {
			continue
		}

		if !testutil.DeepEqual(got, c.want) {
			t.Errorf("ParseProgram(%x) = %+v want %+v", c.prog, got, c.want)
		}
	}
}
