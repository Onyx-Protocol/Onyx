package vmutil

import (
	"encoding/binary"
	"reflect"

	"chain/errors"
	"chain/protocol/vm"
)

type Builder struct {
	items       []item
	jumpCounter int
	opt         bool
}

type item interface {
	bytes() []byte
}

type (
	int64Item    int64
	pushdataItem []byte
	rawdataItem  []byte
	opItem       vm.Op

	jumpItem struct {
		isIf      bool
		targetNum int
	}

	jumpTargetItem int
)

func NewBuilder(optimize bool) *Builder {
	return &Builder{
		opt: optimize,
	}
}

// AddInt64 adds a pushdata instruction for an integer value.
func (b *Builder) AddInt64(n int64) *Builder {
	b.items = append(b.items, int64Item(n))
	return b
}

// AddData adds a pushdata instruction for a given byte string.
func (b *Builder) AddData(data []byte) *Builder {
	b.items = append(b.items, pushdataItem(data))
	return b
}

// AddRawBytes simply appends the given bytes to the program. (It does
// not introduce a pushdata opcode.)
func (b *Builder) AddRawBytes(data []byte) *Builder {
	b.items = append(b.items, rawdataItem(data))
	return b
}

// AddOp adds the given opcode to the program.
func (b *Builder) AddOp(op vm.Op) *Builder {
	b.items = append(b.items, opItem(op))
	b.optimize()
	return b
}

// NewJumpTarget allocates a number that can be used as a jump target
// in AddJump and AddJumpIf. Call SetJumpTarget to associate the
// number with a program location.
func (b *Builder) NewJumpTarget() int {
	b.jumpCounter++
	return b.jumpCounter
}

// AddJump adds a JUMP opcode whose target is the given target
// number. The actual program location of the target does not need to
// be known yet, as long as SetJumpTarget is called before Build.
func (b *Builder) AddJump(target int) *Builder {
	b.items = append(b.items, jumpItem{false, target})
	return b
}

// AddJump adds a JUMPIF opcode whose target is the given target
// number. The actual program location of the target does not need to
// be known yet, as long as SetJumpTarget is called before Build.
func (b *Builder) AddJumpIf(target int) *Builder {
	b.items = append(b.items, jumpItem{true, target})
	return b
}

// SetJumpTarget associates the given jump-target number with the
// current position in the program. It is legal for SetJumpTarget to
// be called at the end of the program, causing jumps using that
// target to fall off the end. There must be a call to SetJumpTarget
// for every jump target used before any call to Build.
func (b *Builder) SetJumpTarget(target int) *Builder {
	b.items = append(b.items, jumpTargetItem(target))
	return b
}

var ErrUnresolvedJump = errors.New("unresolved jump target")

// Build produces the bytecode of the program. It first resolves any
// jumps in the program by filling in the addresses of their
// targets. This requires SetJumpTarget to be called prior to Build
// for each jump target used (in a call to AddJump or AddJumpIf). If
// any target's address hasn't been set in this way, this function
// produces ErrUnresolvedJump. There are no other error conditions.
func (b *Builder) Build() ([]byte, error) {
	var result []byte
	jumps := make(map[int]int)
	jumpTargets := make(map[int]uint32)
	for _, it := range b.items {
		switch j := it.(type) {
		case jumpItem:
			jumps[len(result)] = j.targetNum
		case jumpTargetItem:
			jumpTargets[int(j)] = uint32(len(result))
		}
		result = append(result, it.bytes()...)
	}
	for jloc, targetNum := range jumps {
		addr, ok := jumpTargets[targetNum]
		if !ok {
			return nil, errors.Wrapf(ErrUnresolvedJump, "target %d", targetNum)
		}
		binary.LittleEndian.PutUint32(result[jloc+1:jloc+5], addr)
	}
	return result, nil
}

func (i int64Item) bytes() []byte {
	return vm.PushdataInt64(int64(i))
}

func (i pushdataItem) bytes() []byte {
	return vm.PushdataBytes(i)
}

func (i rawdataItem) bytes() []byte {
	return i
}

func (i opItem) bytes() []byte {
	return []byte{byte(i)}
}

func (i jumpItem) bytes() []byte {
	var b [5]byte
	if i.isIf {
		b[0] = byte(vm.OP_JUMPIF)
	} else {
		b[0] = byte(vm.OP_JUMP)
	}
	return b[:]
}

func (i jumpTargetItem) bytes() []byte {
	return []byte{}
}

var optimizations = []struct {
	before, after []item
}{
	{
		[]item{int64Item(0), opItem(vm.OP_ROLL)}, []item{},
	}, {
		[]item{int64Item(0), opItem(vm.OP_PICK)}, []item{opItem(vm.OP_DUP)},
	}, {
		[]item{int64Item(1), opItem(vm.OP_ROLL)}, []item{opItem(vm.OP_SWAP)},
	}, {
		[]item{int64Item(1), opItem(vm.OP_PICK)}, []item{opItem(vm.OP_OVER)},
	}, {
		[]item{int64Item(2), opItem(vm.OP_ROLL)}, []item{opItem(vm.OP_ROT)},
	}, {
		[]item{opItem(vm.OP_TRUE), opItem(vm.OP_VERIFY)}, []item{},
	}, {
		[]item{opItem(vm.OP_SWAP), opItem(vm.OP_SWAP)}, []item{},
	}, {
		[]item{opItem(vm.OP_OVER), opItem(vm.OP_OVER)}, []item{opItem(vm.OP_2DUP)},
	}, {
		[]item{opItem(vm.OP_SWAP), opItem(vm.OP_OVER)}, []item{opItem(vm.OP_TUCK)},
	}, {
		[]item{opItem(vm.OP_DROP), opItem(vm.OP_DROP)}, []item{opItem(vm.OP_2DROP)},
	}, {
		[]item{opItem(vm.OP_SWAP), opItem(vm.OP_DROP)}, []item{opItem(vm.OP_NIP)},
	}, {
		[]item{int64Item(5), opItem(vm.OP_ROLL), int64Item(5), opItem(vm.OP_ROLL)}, []item{opItem(vm.OP_2ROT)},
	}, {
		[]item{int64Item(3), opItem(vm.OP_PICK), int64Item(3), opItem(vm.OP_PICK)}, []item{opItem(vm.OP_2OVER)},
	}, {
		[]item{int64Item(3), opItem(vm.OP_ROLL), int64Item(3), opItem(vm.OP_ROLL)}, []item{opItem(vm.OP_2SWAP)},
	}, {
		[]item{int64Item(2), opItem(vm.OP_PICK), int64Item(2), opItem(vm.OP_PICK), int64Item(2), opItem(vm.OP_PICK)}, []item{opItem(vm.OP_3DUP)},
	}, {
		[]item{int64Item(1), opItem(vm.OP_ADD)}, []item{opItem(vm.OP_1ADD)},
	}, {
		[]item{int64Item(1), opItem(vm.OP_SUB)}, []item{opItem(vm.OP_1SUB)},
	}, {
		[]item{opItem(vm.OP_EQUAL), opItem(vm.OP_VERIFY)}, []item{opItem(vm.OP_EQUALVERIFY)},
	}, {
		[]item{opItem(vm.OP_SWAP), opItem(vm.OP_TXSIGHASH), opItem(vm.OP_ROT)}, []item{opItem(vm.OP_TXSIGHASH), opItem(vm.OP_SWAP)},
	},
}

func (b *Builder) optimize() {
	if !b.opt {
		return
	}
	looping := true
	for looping {
		looping = false
		for _, o := range optimizations {
			if len(b.items) < len(o.before) {
				continue
			}
			if !reflect.DeepEqual(o.before, b.items[len(b.items)-len(o.before):]) {
				continue
			}
			b.items = append(b.items[:len(b.items)-len(o.before)], o.after...)
			looping = true
		}
		if !looping {
			// a few extra optimizations here that don't fit the static
			// patterns in "optimizations" above
			if len(b.items) >= 3 && b.items[len(b.items)-3] == int64Item(1) && b.items[len(b.items)-1] == opItem(vm.OP_ADD) {
				// 1 <x> ADD => <x> 1ADD
				addend := b.items[len(b.items)-2]
				b.items = b.items[:len(b.items)-3]
				b.items = append(b.items, addend)
				b.items = append(b.items, opItem(vm.OP_1ADD))
				looping = true
				continue
			}
			if len(b.items) >= 2 && b.items[len(b.items)-2] == opItem(vm.OP_SWAP) {
				if op, ok := b.items[len(b.items)-1].(opItem); ok {
					switch vm.Op(op) {
					case vm.OP_EQUAL, vm.OP_ADD, vm.OP_BOOLAND, vm.OP_BOOLOR, vm.OP_MIN, vm.OP_MAX:
						// SWAP <op> => <op> (where <op> is commutative)
						b.items = b.items[:len(b.items)-2]
						b.items = append(b.items, op)
						looping = true
						continue
					}
				}
			}
			if len(b.items) >= 4 && b.items[len(b.items)-4] == opItem(vm.OP_DUP) && b.items[len(b.items)-3] == int64Item(2) && b.items[len(b.items)-2] == opItem(vm.OP_PICK) {
				if op, ok := b.items[len(b.items)-1].(opItem); ok {
					switch vm.Op(op) {
					case vm.OP_EQUAL, vm.OP_ADD, vm.OP_BOOLAND, vm.OP_BOOLOR, vm.OP_MIN, vm.OP_MAX:
						// DUP 2 PICK <op> => 2DUP <op> (where <op> is commutative)
						b.items = b.items[:len(b.items)-4]
						b.items = append(b.items, opItem(vm.OP_2DUP), op)
						looping = true
						continue
					}
				}
			}
		}
	}
}
