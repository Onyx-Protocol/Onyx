package txscript

import (
	"errors"
	"fmt"
)

// scriptStack represents the execution stack of a script. Each stack frame
// contains a script, an alternate data stack, a conditional stack and its
// own program counter. The frame at the highest index is the current stack
// frame. When a frame is finished, it's popped off and execution begins at
// the program counter of the previous stack frame.
type scriptStack struct {
	frames []*stackFrame
}

// Reset returns the script stack to an empty stack.
func (ss *scriptStack) Reset() {
	ss.frames = nil
}

// Peek returns the current top of the stack.
func (ss *scriptStack) Peek() *stackFrame {
	return ss.frames[len(ss.frames)-1]
}

// Push pushes a new stack frame onto the stack.
func (ss *scriptStack) Push(f *stackFrame) {
	if len(f.script) == 0 {
		return
	}
	ss.frames = append(ss.frames, f)
}

// Pop removes and returns the top stack frame.
func (ss *scriptStack) Pop() *stackFrame {
	f := ss.Peek()
	ss.frames = ss.frames[:len(ss.frames)-1]
	return f
}

// Depth returns number of stack frames on the execution stack.
func (ss scriptStack) Depth() int32 {
	return int32(len(ss.frames))
}

// nextFrame will pop all completed stack frames from the top of the stack
// until it finds an unfinished stack frame or all remaining stack frames
// have been popped.
func (ss *scriptStack) nextFrame() (done bool, err error) {
	for len(ss.frames) > 0 && ss.Peek().done() {
		finishedFrame := ss.Pop()

		// Conditionals cannot span stack frames.
		if len(finishedFrame.condStack) != 0 {
			return false, ErrStackMissingEnd
		}
	}
	return len(ss.frames) == 0, nil
}

// disasm is a helper function to produce the output for the engine's
// disassembly methods.  It produces the opcode prefixed by the program counter
// at the provided position in the script.  It does no error checking and
// leaves that to the caller to provide a valid offset.
func (ss scriptStack) disasm(frame, off int) string {
	return fmt.Sprintf("%02x:%s", frame, ss.frames[frame].disasm(off))
}

// empty returns true iff there are no frames on the execution stack.
func (ss scriptStack) empty() bool {
	return len(ss.frames) == 0
}

// validPC returns an error if the current script position is valid for
// execution, nil otherwise.
func (ss *scriptStack) validPC() error {
	if len(ss.frames) == 0 {
		return errors.New("zero stack frames")
	}
	err := ss.Peek().validPC()
	return err
}

// curPC returns the current position of the program counter. If the pc
// is currently in an invalid position, curPC will return an error.
func (ss *scriptStack) curPC() (frame int, pc int, err error) {
	err = ss.validPC()
	if err != nil {
		return 0, 0, err
	}
	return len(ss.frames) - 1, ss.Peek().pc, nil
}

// stackFrame represents a single stack frame on the execution stack. It
// encompasses a script and its current state.
type stackFrame struct {
	script    []parsedOpcode
	condStack []int // conditional stack
	pc        int
}

// clone makes a shallow copy of the stackFrame. It's used in the
// implementation of OP_WHILE.
func (f *stackFrame) clone() *stackFrame {
	clone := *f
	return &clone
}

// step increments the frame's program counter.
func (f *stackFrame) step() {
	f.pc = f.pc + 1
}

// done returns true if this frame has finished execution. A frame has finished
// execution if its program counter moves pass the last instruction.
func (f stackFrame) done() bool {
	return f.pc >= len(f.script)
}

// validPC returns an error if the current script position is valid for
// execution, nil otherwise.
func (f stackFrame) validPC() error {
	if f.done() {
		return fmt.Errorf("past frame's script: pc=%v of %04d", f.pc, len(f.script))
	}
	return nil
}

// disasm is a helper function to produce the output for the engine's
// disassembly methods.  It produces the opcode prefixed by the program counter
// at the provided position in the script.  It does no error checking and
// leaves that to the caller to provide a valid offset.
func (f stackFrame) disasm(off int) string {
	return fmt.Sprintf("%04x: %s", off, f.script[off].print(false))
}

// opcode returns the parsed opcode at the provided offset within the
// script.
func (f stackFrame) opcode(off int) *parsedOpcode {
	return &f.script[off]
}
