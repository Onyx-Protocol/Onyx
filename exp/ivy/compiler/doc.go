/*
Package ivy provides a compiler for Chain's Ivy contract language.

A contract is a means to lock some payment in the output of a
transaction. It contains a number of clauses, each describing a way to
unlock, or redeem, the payment in a subsequent transaction.  By
executing the statements in a clause, using contract arguments
supplied by the payer and clause arguments supplied by the redeemer,
nodes in a Chain network can determine whether a proposed spend is
valid.

The language definition is in flux, but here's what's implemented as
of late May 2017.

  program = contract*

  contract = "contract" identifier "(" [params] ")" "locks" identifier "{" clause+ "}"

    The identifier after "locks" is a name for the value locked by
    the contract. It must be unlocked or re-locked (with "unlock"
    or "lock") in every clause.

  clause = "clause" identifier "(" [params] ")" ["requires" requirements] "{" statement+ "}"

    The requirements are blockchain values that must be present in
    the spending transaction in order to spend the value locked by
    the earlier transaction. Each such value must be re-locked
    (with "lock") in its clause.

  statement = verify | unlock | lock

  verify = "verify" expr

    Verifies that boolean expression expr produces a true result.

  unlock = "unlock" expr

    Expr must evaluate to the contract value. This unlocks that
    value for any use.

  lock = "lock" expr "with" expr

    The first expr must be a blockchain value (i.e., one named
    with "locks" or "requires"). The second expr must be a
    program. This unlocks expr and re-locks it with the new
    program.

  requirements = requirement | requirements "," requirement

  requirement = identifier ":" expr "of" expr

    The first expr must be an amount, the second must be an
    asset. This denotes that the named value must have the given
    quantity and asset type.

  params = param | params "," param

  param = idlist ":" identifier

    The identifiers in idlist are individual parameter names. The
    identifier after the colon is their type. Available types are:

      Amount; Asset; Boolean; Hash; Integer; Program; PublicKey;
      Signature; String; Time

  idlist = identifier | idlist "," identifier

  expr = unary_expr | binary_expr | call_expr | identifier | "(" expr ")" | literal

  unary_expr = unary_op expr

  binary_expr = expr binary_op expr

  call_expr = expr "(" [args] ")"

    If expr is the name of an Ivy contract, then calling it (with
    the appropriate arguments) produces a program suitable for use
    in "lock" statements.

    Otherwise, expr should be one of these builtin functions:

      sha3(x)
        SHA3-256 hash of x.
      sha256(x)
        SHA-256 hash of x.
      size(x)
        Size in bytes of x.
      abs(x)
        Absolute value of x.
      min(x, y)
        The lesser of x and y.
      max(x, y)
        The greater of x and y.
      checkTxSig(pubkey, signature)
        Whether signature matches both the spending
        transaction and pubkey.
      concat(x, y)
        The concatenation of x and y.
      concatpush(x, y)
        The concatenation of x with the bytecode sequence
        needed to push y on the ChainVM stack.
      before(x)
        Whether the spending transaction is happening before
        time x.
      after(x)
        Whether the spending transaction is happening after
        time x.
      checkTxMultiSig([pubkey1, pubkey2, ...], [sig1, sig2, ...])
        Like checkTxSig, but for M-of-N signature checks.
        Every sig must match both the spending transaction and
        one of the pubkeys. There may be more pubkeys than
        sigs, but they are only checked left-to-right so must
        be supplied in the same order as the sigs. The square
        brackets here are literal and must appear as shown.

  unary_op = "-" | "~"

  binary_op = ">" | "<" | ">=" | "<=" | "==" | "!=" | "^" | "|" |
        "+" | "-" | "&" | "<<" | ">>" | "%" | "*" | "/"

  args = expr | args "," expr

  literal = int_literal | str_literal | hex_literal

*/
package compiler
