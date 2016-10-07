/*
Package vm implements the VM described in Chain Protocol 1.

The VM is for verifying transaction inputs and blocks. Accordingly
there are two main entrypoints: VerifyTxInput and VerifyBlockHeader,
both in vm.go. Each constructs a disposable VM object to perform its
computation.

For VerifyTxInput, the program to execute comes from the input
commitment: either the prevout's control program, if it's a spend
input; or the issuance program, if it's an issuance. For
VerifyBlockHeader, the program to execute is the previous block's
consensus program.  In all cases, the VM's data stack is first
populated with witness data from the current object (transaction input
or block).

The program is interpreted byte-by-byte by the main loop in
virtualMachine.run(). Most bytes are opcodes in one of the following categories:
  - bitwise
  - control
  - crypto
  - introspection
  - numeric
  - pushdata
  - splice
  - stack
Each category has a corresponding .go file implementing those opcodes.

Each instruction incurs some cost when executed. These costs are
deducted from (and in some cases refunded to) a predefined run
limit. Costs are tallied in two conceptual phases: "before" the
instruction runs and "after." In practice, "before" charges are
applied on the fly in the body of each opcode's implementation, and
"after" charges are deferred until the instruction finishes, at which
point the VM main loop applies the deferred charges. As such,
functions that have associated costs (chiefly stack pushing and
popping) include a "deferred" flag as an argument.
*/
package vm
