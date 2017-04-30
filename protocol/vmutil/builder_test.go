package vmutil

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"chain/protocol/vm"
)

func TestAddInt64(t *testing.T) {
	cases := []struct {
		num     int64
		wantHex string
	}{
		{0, "00"},
		{1, "51"},
		{15, "5f"},
		{16, "60"},
		{17, "0111"},
		{255, "01ff"},
		{256, "020001"},
		{258, "020201"},
		{65535, "02ffff"},
		{65536, "03000001"},
		{-1, "08ffffffffffffffff"},
		{-2, "08feffffffffffffff"},
		{-65536, "080000ffffffffffff"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("adding %d", c.num), func(t *testing.T) {
			b := NewBuilder()
			b.AddInt64(c.num)
			prog, err := b.Build()
			if err != nil {
				t.Fatal(err)
			}
			want, err := hex.DecodeString(c.wantHex)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(prog, want) {
				t.Errorf("got %x, want %x", prog, want)
			}
		})
	}
}

func TestAddJump(t *testing.T) {
	cases := []struct {
		name    string
		wantHex string
		fn      func(t *testing.T, b *Builder)
	}{
		{
			"single jump single target not yet defined",
			"630600000061",
			func(t *testing.T, b *Builder) {
				target := b.NewJumpTarget()
				b.AddJump(target)
				b.AddOp(vm.OP_NOP)
				b.SetJumpTarget(target)
			},
		},
		{
			"single jump single target already defined",
			"616300000000",
			func(t *testing.T, b *Builder) {
				target := b.NewJumpTarget()
				b.SetJumpTarget(target)
				b.AddOp(vm.OP_NOP)
				b.AddJump(target)
			},
		},
		{
			"two jumps single target not yet defined",
			"630c00000061630c00000061",
			func(t *testing.T, b *Builder) {
				target := b.NewJumpTarget()
				b.AddJump(target)
				b.AddOp(vm.OP_NOP)
				b.AddJump(target)
				b.AddOp(vm.OP_NOP)
				b.SetJumpTarget(target)
			},
		},
		{
			"two jumps single target already defined",
			"616300000000616300000000",
			func(t *testing.T, b *Builder) {
				target := b.NewJumpTarget()
				b.SetJumpTarget(target)
				b.AddOp(vm.OP_NOP)
				b.AddJump(target)
				b.AddOp(vm.OP_NOP)
				b.AddJump(target)
			},
		},
		{
			"two jumps single target, one not yet defined, one already defined",
			"630600000061616306000000",
			func(t *testing.T, b *Builder) {
				target := b.NewJumpTarget()
				b.AddJump(target)
				b.AddOp(vm.OP_NOP)
				b.SetJumpTarget(target)
				b.AddOp(vm.OP_NOP)
				b.AddJump(target)
			},
		},
		{
			"two jumps, two targets, not yet defined",
			"630c00000061630d0000006161",
			func(t *testing.T, b *Builder) {
				target1 := b.NewJumpTarget()
				b.AddJump(target1)
				b.AddOp(vm.OP_NOP)
				target2 := b.NewJumpTarget()
				b.AddJump(target2)
				b.AddOp(vm.OP_NOP)
				b.SetJumpTarget(target1)
				b.AddOp(vm.OP_NOP)
				b.SetJumpTarget(target2)
			},
		},
		{
			"two jumps, two targets, already defined",
			"6161616301000000616302000000",
			func(t *testing.T, b *Builder) {
				b.AddOp(vm.OP_NOP)
				target1 := b.NewJumpTarget()
				b.SetJumpTarget(target1)
				b.AddOp(vm.OP_NOP)
				target2 := b.NewJumpTarget()
				b.SetJumpTarget(target2)
				b.AddOp(vm.OP_NOP)
				b.AddJump(target1)
				b.AddOp(vm.OP_NOP)
				b.AddJump(target2)
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := NewBuilder()
			c.fn(t, b)
			prog, err := b.Build()
			if err != nil {
				t.Fatal(err)
			}
			want, err := hex.DecodeString(c.wantHex)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(prog, want) {
				t.Errorf("got %x, want %x", prog, want)
			}
		})
	}
}
