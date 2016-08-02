/*

Command p2cc parses Chain's high-level p2c contract descriptions and
compiles them to ChainOS opcodes.

Output includes the program listing in symbolic opcode form, annotated
with illustrations of stack changes; plus the same program as hex
bytes; plus pkscript hex bytes made from the program's "contracthash."

Input file may be named on the command line.  In its absence, stdin is
read.  Output is to stdout.

The code produced assumes that, immediately prior to evaluation of any
contract clauses, the stack looks like:

  clauseParamN clauseParamN-1 ... clauseParam1 [clauseSelector] [contract] contractParamN contractParamN-1 ... contractParam1

The contract is present iff the contract pkscript is in
contracthash form.  The clauseSelector (a number from 1 through N) is
present iff the contract has more than one clause.  The clauseParams
correspond to whichever clause is selected.

The language understood by p2cc looks like this:

  contract name(param1, param2, ..., paramN) {
    clause name1(param11, param12, ..., param1N) {
      decl1
      decl2
      ...
      declN
      statement1
      statement2
      ...
      statementN
      [expr]
    }
    clause name2(...) { ... }
  }

Each param is an identifier followed by an optional type, one of num,
bool, or bytes.  (Types, which are also inferred from expressions, are
used to choose between e.g. EQUAL and NUMEQUAL at compile time.)

Each decl looks like:

  var name expr;

creating a variable named name, scoped to the clause in which it
appears, with an initial value given by expr.

Each statement is one of:

  if expr { ...statements... } [else { ...statements... }]

or

  while expr { ...statements... }

or

  var OP expr;

(where OP is an assignment operator, see below) or

  verify expr;

Valid assignment operators are =, *=, /=, %=, <<=, >>=, &=, &^=, +=,
-=, |=, ^=, &&=, ||=.

Expressions are one of:

  (expr)
  OP expr [for unary operators -, !, and ^]
  name(arg1, arg2, ..., argN) for various predefined function calls (see below)
  name (where name is a variable or contract or clause parameter)
  literal (one of base10-int, 'string', or 0x... hex bytes)

Function calls are:

  asset()
  amount()
  program()
  time()
  circulation(assetID)
  abs(num)
  hash256(bytes)
  checkpredicate(bytes)
  size(bytes)
  min(num1, num2)
  max(num1, num2)
  checksig(signature, pubkey)
  cat(bytes1, bytes2)
  catpushdata(bytes1, bytes2)
  left(bytes, num)
  right(bytes, num)
  reserveoutput(amount, assetID, bytes)
  findoutput(amount, assetID, bytes)
  substr(bytes, begin, size)

A "function call" may also be

  name(val1, val2, ...)

where name is the name of a contract (which must be one of the ones
parsed from the current input) and the number of vals matches the
number of contract parameters for name.  In this case, the compiler
will emit code to construct the pkscript for the given contract and
parameters, for use as the final argument to RESERVEOUTPUT and
FINDOUTPUT.

Caveat: The contracthash is needed for this to work, and when a
contract refers to itself in this way, the contracthash is not known
at translation time.  But it is known at runtime, so the emitted code
performs manipulations on PROGRAM, therefore it assumes that the
contract was invoked with p2c contracthash style (and not p2c inline
style).

Comments are introduced with # and continue to the end of the line.

*/
package main
