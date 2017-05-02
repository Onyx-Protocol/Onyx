/*
Package ivy provides a compiler for Chain's Ivy contract language.

A contract is a means to lock some payment in the output of a
transaction. It contains a number of clauses, each describing a way to
unlock, or redeem, the payment in a subsequent transaction.  By
executing the statements in a clause, using contract arguments
supplied by the payer and clause arguments supplied by the redeemer,
nodes in a Chain network can determine whether a proposed spend is
valid.
*/
package ivy
