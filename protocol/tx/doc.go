/*
Package tx implements transaction hashing, as revised in January 2017.

The data structures used in this package are intended eventually to
replace Tx, TxData, TxInput, TxOutput, and others in the protocol/bc
package. In this new formulation, a transaction is a collection of
abstract "entries," each with a specific type (such as spend,
issuance, output, etc.), and each with its own distinct hash. The hash
of a designated "header" entry serves as the hash of the entire
transaction. The rationale for this change is that it is considerably
more extensible, and it allows future scripting tools to traverse and
access transaction data by making all components hash-addressable.

As a first step to replacing the protocol/bc types with the ones here,
hashing (of the old types) is redefined to mean "convert to the new
data structures and hash that." Thus, hash values computed now will
not be affected when we switch more fully to the new types in the
future.
*/
package tx
