package main

import (
	"fmt"
	"strings"
)

type (
	assignOp struct {
		op, translation string
	}
	assignStmt struct {
		name, op string
		expr     translatable
	}
)

func (a assignStmt) translate(stack []stackItem, context *context) ([]translation, error) {
	varDepth, err := lookup(a.name, stack)
	if err != nil {
		return nil, err
	}
	output, err := a.expr.translate(stack, context)
	if err != nil {
		return nil, err
	}
	preAssignStackItem := output[len(output)-1].stack[0]
	// After the opcodes in output, the new value is on top of the
	// runtime stack.  Need to expose the old value by moving
	// intermediate stack values to the altstack.
	for i := 0; i < varDepth; i++ {
		s := []stackItem{preAssignStackItem}
		s = append(s, stack[i+1:]...)
		output = append(output, translation{"SWAP TOALTSTACK", s})
	}
	// Stack is now ...oldValue newValue.  Must consume oldValue and combine or replace it with newValue.
	var postAssignStackItem stackItem
	if a.op == "=" {
		postAssignStackItem = preAssignStackItem
		s := []stackItem{preAssignStackItem}
		s = append(s, stack[varDepth+1:]...)
		output = append(output, translation{"NIP", s})
	} else {
		for _, op := range binaryOps {
			if op.canAssign {
				if len(a.op) == len(op.op)+1 && strings.HasPrefix(a.op, op.op) {
					postAssignStackItem = stackItem{name: fmt.Sprintf("[%s(%s, %s)]", op.translation, preAssignStackItem, stack[varDepth])}
					s := []stackItem{postAssignStackItem}
					s = append(s, stack[varDepth+1:]...)
					output = append(output, translation{op.translation, s})
					break
				}
			}
		}
	}
	for i := 0; i < varDepth; i++ {
		s := stack[varDepth-1-i:]
		output = append(output, translation{"FROMALTSTACK", s})
	}
	return output, nil
}

type ifStmt struct {
	condExpr              translatable
	consequent, alternate *block
}

func (ifstmt ifStmt) translate(stack []stackItem, context *context) ([]translation, error) {
	output, err := ifstmt.condExpr.translate(stack, context)
	if err != nil {
		return nil, err
	}
	output = append(output, translation{"IF", stack})
	t, err := ifstmt.consequent.translate(stack, context)
	if err != nil {
		return nil, err
	}
	output = append(output, t...)
	if ifstmt.alternate != nil {
		output = append(output, translation{"ELSE", stack})
		t, err = ifstmt.alternate.translate(stack, context)
		if err != nil {
			return nil, err
		}
		output = append(output, t...)
	}
	output = append(output, translation{"ENDIF", stack})
	return output, nil
}

type verifyStmt struct {
	expr expr
}

func (v verifyStmt) translate(stack []stackItem, context *context) ([]translation, error) {
	e, err := v.expr.translate(stack, context)
	if err != nil {
		return nil, err
	}
	return append(e, translation{"VERIFY", stack}), nil
}

type whileStmt struct {
	condExpr expr
	body     *block
}

// Translation of while <expr> { ...body... } is:
//   <expr> WHILE DROP <body> <expr> ENDWHILE
// This makes sure the expr is reevaluated and on the stack at the top
// of each loop iteration.
func (w whileStmt) translate(stack []stackItem, context *context) ([]translation, error) {
	cond, err := w.condExpr.translate(stack, context)
	if err != nil {
		return nil, err
	}
	result := append(cond, translation{"WHILE DROP", stack})
	t, err := w.body.translate(stack, context)
	if err != nil {
		return nil, err
	}
	result = append(result, t...)
	// Don't need to add another copy of cond here to make it appear in
	// the translation.  See the "Hark, a hack!" comment in parse.go.
	return append(result, translation{"ENDWHILE", stack}), nil
}
