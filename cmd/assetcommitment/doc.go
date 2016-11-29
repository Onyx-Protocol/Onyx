/*

Command assetcommitment creates blinded and non-blinded asset commitments.

Usage:

	assetcommitment <assetid >nonblinded
	assetcommitment -key KEY_HEX <assetcommitment >blinded

The first form produces an unblinded asset commitment from the given asset ID.

The second form produces a blinded asset commitment from a previous
commitment (and its associated cumulative blinding factor) and an
asset-encoding key, which must be a 32-byte value encoded as 64 hex
bytes. It should be derived from a record encryption key.

In both cases, the output is a JSON object with two fields: H, the
hex-encoded asset commitment; and C, the cumulative blinding
factor. This structure is suitable as input to the second form.

*/
package main
