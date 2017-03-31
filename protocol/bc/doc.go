/*
Package bc provides the fundamental blockchain data structures used in
the Chain Protocol.

This package is in transition from a set of "old" data structures
(TxData, TxInput, TxOutput, etc.) to a new data model based on
"entries," each with a specific type (such as spend, issuance, output,
etc.), and each with its own distinct hash. The hash of a designated
"header" entry serves as the hash of the entire transaction. The
rationale for this change is that it is considerably more extensible,
and it allows future scripting tools to traverse and access
transaction data by making all components hash-addressable.

Hashing and validation (of the old types) are redefined to mean
"convert to the new data structures and hash/validate that."

Soon the old structures will be retired entirely.

These changes will be made in a compatible way; in particular, block
and transaction hashes will not change.
*/
package bc
