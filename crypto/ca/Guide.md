A guide to crypto/ca
====================

The `crypto/ca` package contains the logic and data structures for
confidential assets: a set of features that allows blinding of
transaction amounts and/or asset IDs. Transactions that use these
features have alternate input and output data structures with newly
defined asset version 2 (which forces the transaction to have asset
version 2, and forces blocks containing v2 transactions to have block
version 2).

In contrast with an asset-version-1 transaction output,
[a v2 output](https://github.com/chain/chain-stealth/blob/d953f2f4d803122a7073e234c841314740eda80a/protocol/bc/outputv2.go#L12)
does not contain an `AssetAmount`. Instead it contains an
`AssetDescriptor` and `AssetRangeProof`, plus a `ValueDescriptor` and
`ValueRangeProof`. These new types are defined in their own files in
`crypto/ca`:
- [`asset_descriptor.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/asset_descriptor.go)
- [`asset_range_proof.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/asset_range_proof.go)
- [`value_descriptor.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/value_descriptor.go)
- [`value_range_proof.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/value_range_proof.go)

[A v2 spend input](https://github.com/chain/chain-stealth/blob/d953f2f4d803122a7073e234c841314740eda80a/protocol/bc/spend.go#L10)
is unchanged from v1, except that its embedded output commitment
(duplicated from the prevout) has v2-specific fields as described
above. [A v2 issuance input](https://github.com/chain/chain-stealth/blob/d953f2f4d803122a7073e234c841314740eda80a/protocol/bc/issuance2.go#L13)
is like a v2 spend _output_ in that it contains asset and value
descriptors and range proofs, but its asset-range-proof is different:
it’s an `IssuanceAssetRangeProof`. It also contains a list of “asset
choices,” each of which is an `IssuanceWitness`.
- [`issuance_asset_range_proof.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/issuance_asset_range_proof.go)
- [`issuance_witness.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/protocol/bc/issuance_witness.go) (part of the `protocol/bc` package)

Asset-version-2 validation happens in
[`CheckTxWellFormed`](https://github.com/chain/chain-stealth/blob/3ba5d81af52ada530a4143fc048de8ab121e67ee/protocol/validation/tx.go#L89)
in `protocol/validation/tx.go`. That function uses the entrypoint
[`VerifyConfidentialAssets`](https://github.com/chain/chain-stealth/blob/d953f2f4d803122a7073e234c841314740eda80a/crypto/ca/verification.go#L89)
defined in `crypto/ca/verification.go`.  The types of that function’s
arguments are interfaces defined in
[`transaction_interfaces.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/transaction_interfaces.go),
satisfied by the v2 types in `protocol/bc` but also by some simpler
types in `crypto/ca` for unit-testing purposes.

Descriptors and range proofs are defined in terms of ring signatures
(ordinary and borommean) and commitments:
- [`asset_commitment.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/asset_commitment.go)
- [`value_commitment.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/value_commitment.go)
- [`ring_signature.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/ring_signature.go)
- [`borromean_ring_signature.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/borromean_ring_signature.go)

...plus encrypted values and asset ids:
- [`encrypted_value.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/encrypted_value.go)
- [`encrypted_asset_id.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/encrypted_asset_id.go)

Those in turn are built up from operations on 32-byte scalars and
points on the ed25519 curve:
- [`scalar.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/scalar.go)
- [`point.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/point.go)

Ed25519 arithmetic support is supplied by the
[`crypto/ed25519/edwards25519`](https://github.com/chain/chain-stealth/tree/confidential-assets/crypto/ed25519/edwards25519)
package.

The remaining code in the `crypto/ca` package includes:
- Some utility functions, mostly related to hashing, found in [`binary.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/binary.go)
- So-called “transient issuance keys,” needed during creation of confidential issuances, implemented in [`transient_issuance_key.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/transient_issuance_key.go)
- The confidential-assets key hierarchy, implemented in [`key_derivation.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/key_derivation.go)
- Logic for “excess commitments,” used for balancing value commitments, implemented in [`excess_commitment.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/excess_commitment.go)
- High-level encryption/decryption support (analogous to the high-level entrypoint in `verification.go`) in [`encryption.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/encryption.go)
- A collection of powers of 10 expressed as ed25519 scalars (used in value range proofs) in [`decimal_exponent.go`](https://github.com/chain/chain-stealth/blob/confidential-assets/crypto/ca/decimal_exponent.go).

Creation of confidential-assets objects is spread among these functions:
- [`NewConfidentialSpend`](https://github.com/chain/chain-stealth/blob/d953f2f4d803122a7073e234c841314740eda80a/protocol/bc/txinput.go#L58)
- [`NewConfidentialIssuanceInput`](https://github.com/chain/chain-stealth/blob/d953f2f4d803122a7073e234c841314740eda80a/protocol/bc/txinput.go#L86)
- [`NewTxOutputv2`](https://github.com/chain/chain-stealth/blob/d953f2f4d803122a7073e234c841314740eda80a/protocol/bc/txoutput.go#L57)
