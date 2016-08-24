package main

import (
	"fmt"
	"strings"
)

type (
	assignStmt struct {
		name, op string
		expr     translatable
	}
)

func (a assignStmt) translate(stk stack, context *context) (*translation, error) {
	varDepth := stk.lookup(a.name)
	if varDepth < 0 {
		return nil, fmt.Errorf("unknown variable %s", a.name)
	}
	output, err := a.expr.translate(stk, context)
	if err != nil {
		return nil, err
	}
	preAssignStackItem := output.finalStackTop()
	// After the opcodes in output, the new value is on top of the
	// runtime stack.  Need to expose the old value by moving
	// intermediate stack values to the altstack.
	for i := 0; i < varDepth; i++ {
		s := stk.dropN(i + 1)
		s = s.push(preAssignStackItem)
		output = output.add("SWAP TOALTSTACK", s)
	}
	// Stack is now ...oldValue newValue.  Must consume oldValue and combine or replace it with newValue.
	var postAssignStackItem typedName
	if a.op == "=" {
		postAssignStackItem = preAssignStackItem
		s := stk.dropN(varDepth + 1)
		s = s.push(preAssignStackItem)
		output = output.add("NIP", s)
	} else {
		for _, op := range binaryOps {
			if op.canAssign {
				if len(a.op) == len(op.op)+1 && strings.HasPrefix(a.op, op.op) {
					postAssignStackItem = typedName{name: fmt.Sprintf("[%s(%s, %s)]", op.translation, preAssignStackItem, stk[varDepth])}
					s := stk.dropN(varDepth + 1)
					s = s.push(postAssignStackItem)
					output = output.add(op.translation, s)
					break
				}
			}
		}
	}
	for i := 0; i < varDepth; i++ {
		output = output.add("FROMALTSTACK", stk.dropN(varDepth-1-i))
	}
	return output, nil
}

type ifStmt struct {
	condExpr              translatable
	consequent, alternate *block
}

func (ifstmt ifStmt) translate(stk stack, context *context) (*translation, error) {
	output, err := ifstmt.condExpr.translate(stk, context)
	if err != nil {
		return nil, err
	}
	output = output.add("IF", stk)
	t, err := ifstmt.consequent.translate(stk, context)
	if err != nil {
		return nil, err
	}
	output = output.addMany(t.steps)
	if ifstmt.alternate != nil {
		output = output.add("ELSE", stk)
		t, err = ifstmt.alternate.translate(stk, context)
		if err != nil {
			return nil, err
		}
		output = output.addMany(t.steps)
	}
	output = output.add("ENDIF", stk)
	return output, nil
}

type verifyStmt struct {
	expr expr
}

func (v verifyStmt) translate(stk stack, context *context) (*translation, error) {
	e, err := v.expr.translate(stk, context)
	if err != nil {
		return nil, err
	}
	return e.add("VERIFY", stk), nil
}

type whileStmt struct {
	condExpr expr
	body     *block
}

// Translation of while <expr> { ...body... } is:
//   <expr> WHILE DROP <body> <expr> ENDWHILE
// This makes sure the expr is reevaluated and on the stack at the top
// of each loop iteration.
func (w whileStmt) translate(stk stack, context *context) (*translation, error) {
	cond, err := w.condExpr.translate(stk, context)
	if err != nil {
		return nil, err
	}
	result := cond.add("WHILE DROP", stk)
	t, err := w.body.translate(stk, context)
	if err != nil {
		return nil, err
	}
	result = result.addMany(t.steps)
	// Don't need to add another copy of cond here to make it appear in
	// the translation.  See the "Hark, a hack!" comment in parse.go.
	return result.add("ENDWHILE", stk), nil
}
