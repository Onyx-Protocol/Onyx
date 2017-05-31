package compiler

import (
	"fmt"
	"strconv"
	"strings"
)

type builder struct {
	items         []*builderItem
	pendingVerify *builderItem
}

type builderItem struct {
	opcodes string
	stk     stack
}

func (b *builder) add(opcodes string, newstack stack) stack {
	if b.pendingVerify != nil {
		b.items = append(b.items, b.pendingVerify)
		b.pendingVerify = nil
	}
	item := &builderItem{opcodes: opcodes, stk: newstack}
	if opcodes == "VERIFY" {
		b.pendingVerify = item
	} else {
		b.items = append(b.items, item)
	}
	return newstack
}

func (b *builder) addRoll(stk stack, n int) stack {
	b.addInt64(stk, int64(n))
	return b.add("ROLL", stk.roll(n))
}

func (b *builder) addDup(stk stack) stack {
	return b.add("DUP", stk.dup())
}

func (b *builder) addInt64(stk stack, n int64) stack {
	s := strconv.FormatInt(n, 10)
	return b.add(s, stk.add(s))
}

func (b *builder) addNumEqual(stk stack, desc string) stack {
	return b.add("NUMEQUAL", stk.dropN(2).add(desc))
}

func (b *builder) addJumpIf(stk stack, label string) stack {
	return b.add(fmt.Sprintf("JUMPIF:$%s", label), stk.drop())
}

func (b *builder) addJumpTarget(stk stack, label string) stack {
	return b.add("$"+label, stk)
}

func (b *builder) addDrop(stk stack) stack {
	return b.add("DROP", stk.drop())
}

func (b *builder) forgetPendingVerify() {
	b.pendingVerify = nil
}

func (b *builder) addJump(stk stack, label string) stack {
	return b.add(fmt.Sprintf("JUMP:$%s", label), stk)
}

func (b *builder) addVerify(stk stack) stack {
	return b.add("VERIFY", stk.drop())
}

func (b *builder) addData(stk stack, data []byte) stack {
	var s string
	switch len(data) {
	case 0:
		s = "0"
	case 1:
		s = strconv.FormatInt(int64(data[0]), 10)
	default:
		s = fmt.Sprintf("0x%x", data)
	}
	return b.add(s, stk.add(s))
}

func (b *builder) addAmount(stk stack) stack {
	return b.add("AMOUNT", stk.add("<amount>"))
}

func (b *builder) addAsset(stk stack) stack {
	return b.add("ASSET", stk.add("<asset>"))
}

func (b *builder) addCheckOutput(stk stack, desc string) stack {
	return b.add("CHECKOUTPUT", stk.dropN(6).add(desc))
}

func (b *builder) addBoolean(stk stack, val bool) stack {
	if val {
		return b.add("TRUE", stk.add("true"))
	}
	return b.add("FALSE", stk.add("false"))
}

func (b *builder) addOps(stk stack, ops string, desc string) stack {
	return b.add(ops, stk.add(desc))
}

func (b *builder) addToAltStack(stk stack) (stack, string) {
	t := stk.top()
	return b.add("TOALTSTACK", stk.drop()), t
}

func (b *builder) addTxSigHash(stk stack) stack {
	return b.add("TXSIGHASH", stk.add("<txsighash>"))
}

func (b *builder) addFromAltStack(stk stack, alt string) stack {
	return b.add("FROMALTSTACK", stk.add(alt))
}

func (b *builder) addSwap(stk stack) stack {
	return b.add("SWAP", stk.swap())
}

func (b *builder) addCheckMultisig(stk stack, n int, desc string) stack {
	return b.add("CHECKMULTISIG", stk.dropN(n).add(desc))
}

func (b *builder) addOver(stk stack) stack {
	return b.add("OVER", stk.over())
}

func (b *builder) addPick(stk stack, n int) stack {
	b.addInt64(stk, int64(n))
	return b.add("PICK", stk.pick(n))
}

func (b *builder) addCatPushdata(stk stack, desc string) stack {
	return b.add("CATPUSHDATA", stk.dropN(2).add(desc))
}

func (b *builder) addCat(stk stack, desc string) stack {
	return b.add("CAT", stk.dropN(2).add(desc))
}

func (b *builder) opcodes() string {
	var ops []string
	for _, item := range b.items {
		ops = append(ops, item.opcodes)
	}
	return strings.Join(ops, " ")
}

// This is for producing listings like:
// 5                 |  [... <clause selector> borrower lender deadline balanceAmount balanceAsset 5]
// ROLL              |  [... borrower lender deadline balanceAmount balanceAsset <clause selector>]
// JUMPIF:$default   |  [... borrower lender deadline balanceAmount balanceAsset]
// $repay            |  [... borrower lender deadline balanceAmount balanceAsset]
// 0                 |  [... borrower lender deadline balanceAmount balanceAsset 0]
// 0                 |  [... borrower lender deadline balanceAmount balanceAsset 0 0]
// 3                 |  [... borrower lender deadline balanceAmount balanceAsset 0 0 3]
// ROLL              |  [... borrower lender deadline balanceAsset 0 0 balanceAmount]
// 3                 |  [... borrower lender deadline balanceAsset 0 0 balanceAmount 3]
// ROLL              |  [... borrower lender deadline 0 0 balanceAmount balanceAsset]
// 1                 |  [... borrower lender deadline 0 0 balanceAmount balanceAsset 1]
// 6                 |  [... borrower lender deadline 0 0 balanceAmount balanceAsset 1 6]
// ROLL              |  [... borrower deadline 0 0 balanceAmount balanceAsset 1 lender]
// CHECKOUTPUT       |  [... borrower deadline checkOutput(payment, lender)]
// VERIFY            |  [... borrower deadline]
// 1                 |  [... borrower deadline 1]
// 0                 |  [... borrower deadline 1 0]
// AMOUNT            |  [... borrower deadline 1 0 <amount>]
// ASSET             |  [... borrower deadline 1 0 <amount> <asset>]
// 1                 |  [... borrower deadline 1 0 <amount> <asset> 1]
// 6                 |  [... borrower deadline 1 0 <amount> <asset> 1 6]
// ROLL              |  [... deadline 1 0 <amount> <asset> 1 borrower]
// CHECKOUTPUT       |  [... deadline checkOutput(collateral, borrower)]
// JUMP:$_end        |  [... borrower lender deadline balanceAmount balanceAsset]
// $default          |  [... borrower lender deadline balanceAmount balanceAsset]
// 2                 |  [... borrower lender deadline balanceAmount balanceAsset 2]
// ROLL              |  [... borrower lender balanceAmount balanceAsset deadline]
// MINTIME LESSTHAN  |  [... borrower lender balanceAmount balanceAsset after(deadline)]
// VERIFY            |  [... borrower lender balanceAmount balanceAsset]
// 0                 |  [... borrower lender balanceAmount balanceAsset 0]
// 0                 |  [... borrower lender balanceAmount balanceAsset 0 0]
// AMOUNT            |  [... borrower lender balanceAmount balanceAsset 0 0 <amount>]
// ASSET             |  [... borrower lender balanceAmount balanceAsset 0 0 <amount> <asset>]
// 1                 |  [... borrower lender balanceAmount balanceAsset 0 0 <amount> <asset> 1]
// 7                 |  [... borrower lender balanceAmount balanceAsset 0 0 <amount> <asset> 1 7]
// ROLL              |  [... borrower balanceAmount balanceAsset 0 0 <amount> <asset> 1 lender]
// CHECKOUTPUT       |  [... borrower balanceAmount balanceAsset checkOutput(collateral, lender)]
// $_end             |  [... borrower lender deadline balanceAmount balanceAsset]

type (
	Step struct {
		Opcodes string `json:"opcodes"`
		Stack   string `json:"stack"`
	}
)

func (b *builder) steps() []Step {
	var result []Step
	for _, item := range b.items {
		result = append(result, Step{item.opcodes, item.stk.String()})
	}
	return result
}
