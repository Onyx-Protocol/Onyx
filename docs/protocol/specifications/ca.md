# Confidential Assets

* [Introduction](#introduction)
* [Usage](#usage)
  * [Confidential issuance](#confidential-issuance)
  * [Simple transfer](#simple-transfer)
  * [Multi-party transaction](#multi-party-transaction)
* [Cryptographic primitives](#cryptographic-primitives)
  * [Elliptic curve](#elliptic-curve)
  * [Scalar](#scalar)
  * [Points](#points)
  * [Hash functions](#hash-functions)
  * [Ring Signature](#ring-signature)
  * [Borromean Ring Signature](#borromean-ring-signature)
* [Keys](#keys)
  * [2D Key](#2d-key)
  * [Record Encryption Key](#record-encryption-key)
  * [Intermediate Encryption Key](#intermediate-encryption-key)
  * [Asset ID Encryption Key](#asset-id-encryption-key)
  * [Value Encryption Key](#value-encryption-key)
  * [Asset ID Blinding Factor](#asset-id-blinding-factor)
  * [Value Blinding Factor](#value-blinding-factor)
  * [Transient Issuance Key](#transient-issuance-key)
* [Commitments](#commitments)
  * [Asset ID Point](#asset-id-point)
  * [Asset ID Commitment](#asset-id-commitment)
  * [Value Commitment](#value-commitment)
  * [Excess Factor](#excess-factor)
  * [Excess Commitment](#excess-commitment)
  * [Validate Value Commitments Balance](#validate-value-commitments-balance)
* [Proofs](#proofs)
  * [Asset Range Proof](#asset-range-proof)
  * [Issuance Asset Range Proof](#issuance-asset-range-proof)
  * [Issuance Proof](#issuance-proof)
  * [Value Range Proof](#value-range-proof)
  * [Value Proof](#value-proof)
  * [Validate Issuance](#validate-issuance)
  * [Validate Destination](#validate-destination)
  * [Validate Assets Flow](#validate-assets-flow)
* [Encryption](#encryption)
  * [Encrypted Payload](#encrypted-payload)
  * [Encrypted Value](#encrypted-value)
  * [Encrypted Asset ID](#encrypted-asset-id)
  * [Encrypt Issuance](#encrypt-issuance)
  * [Encrypt Output](#encrypt-output)
  * [Decrypt Output](#decrypt-output)
* [Security](#security)
* [Test vectors](#test-vectors)
* [Glossary](#glossary)



## Introduction

In Chain Protocol 2, asset IDs and amounts in transaction inputs and outputs can be kept private. These details are encrypted homomorphically: the network can verify that transactions are balanced (and, consequently, that the ledger is consistent) without learning exact asset IDs or amounts. Designated recipients and auditors share the encryption key that gives them “read access” to the asset ID and amount of a particular output.

Blinded amounts can be selectively revealed to the network in order to make them visible to smart contracts. The asset ID and the amount can be hidden or revealed independently enabling fine-grained privacy control.

Amounts can have *absolute privacy*, independent of the structure and history of any given transaction. Asset IDs are *relatively private*, meaning their privacy set is the union of the sets of possible asset IDs committed to by transaction inputs.

[[NOTES: Define absolute privacy. Define privacy set.]]

Cryptographic proofs for blinded asset IDs and blinded amounts require relatively large amounts of data (typically 3 to 5 KB). However, almost 80% of that space can be reused to encrypt a confidential message addressed to a designated recipient of the transaction output. The protocol specifies algorithms for encrypting and decrypting this data, along with parameters that allow the user creating the transaction to tune the size of the proofs trading off some of the privacy for lower bandwidth requirements (e.g. blinding a 32-bit amount requires half as much data compared to blinding a full-resolution 64-bit integer).

Present scheme is *perfectly binding*, but only *computationally hiding*. (In fact, [it is impossible](http://crypto.stackexchange.com/questions/41822/why-cant-the-commitment-schemes-have-both-information-theoretic-hiding-and-bind) for a commitment scheme to be both perfectly binding and perfectly hiding at the same time.) This means that breaking elliptic curve discrete logarithm problem (ECDLP) will not compromise the integrity of the commitments, that bind the value perfectly and do not allow manipulations using any amount of computational resources. However, breaking ECDLP can compromise commitments’ hiding property, which rests on discovery of the blinding factor being computationally hard (which is the case only until a powerful quantum computer is made or there is a breakthrough in solving ECDLP with classical computers).

## Usage

In this section we will provide a brief overview of various ways to use confidential assets.

### Confidential issuance

1. Issuer chooses asset ID and an amount to issue.
2. Issuer generates issuance [REK](#record-encryption-key) unique for this issuance.
3. Issuer chooses a set of other asset IDs to add into the *issuance anonymity set*.
4. Issuer sorts the union of the issuance asset ID and anonymity set lexicographically.
5. For each asset ID, where issuance program does not check an issuance key, issuer [creates a transient issuance key](#transient-issuance-key).
6. Issuer provides arguments for each issuance program of each asset ID.
7. Issuer [encrypts issuance](#encrypt-issuance): generates asset ID and value commitments and provides necessary range proofs. Issuer uses ID of the [Nonce](blockchain.md#nonce) as `nonce` and [issuance program](blockchain.md#program) as `message`.
8. Issuer remembers values `(AC,c,f)` to help complete the transaction. Once outputs are fully or partially specified, these values can be discarded.
9. Issuer proceeds with the rest of the transaction creation. See [simple transfer](#simple-transfer) and [multi-party transaction](#multi-party-transaction) for details.

### Simple transfer

1. Recipient generates the following parameters and sends them privately to the sender:
    * amount and asset ID to be sent,
    * control program,
    * [REK1](#record-encryption-key).
2. Sender composes a transaction with an unencrypted output with a given control program.
3. Sender adds necessary amount of inputs to satisfy the output:
    1. For each unspent output, sender pulls values `(assetid,value,AC,c,f)` from its DB.
    2. If the unspent output is not confidential, then:
        * `AC=(A,O)`, [nonblinded asset ID commitment](#asset-id-commitment)
        * `c=0`
        * `f=0`
4. If sender needs to add a change output:
    1. Sender [encrypts](#encrypt-output) the first output.
    2. Sender [balances blinding factors](#balance-blinding-factors) to create an excess factor `q`.
    3. Sender adds unencrypted change output.
    4. Sender generates a [REK2](#record-encryption-key) for the change output.
    5. Sender [encrypts](#encrypt-output) the change output with an additional excess factor `q`.
5. If sender does not need a change output:
    1. Sender [balances blinding factors](#balance-blinding-factors) of the inputs to create an excess factor `q`.
    2. Sender [encrypts](#encrypt-output) the first output with an additional excess factor `q`.
6. Sender stores values `(assetid,value,AC,c,f)` in its DB with its change output for later spending.
7. Sender publishes the transaction.
8. Recipient receives the transaction and identifies its output via its control program.
9. Recipient uses its [REK1](#record-encryption-key) to [decrypt output](#decrypt-output).
10. Recipient stores resulting `(assetid,value,H,c,f)` in its DB with its change output for later spending.
11. Recipient separately stores decrypted plaintext payload with the rest of the reference data. It is not necessary for spending.


### Multi-party transaction

1. All parties communicate out-of-band payment details (how much is being paid for what), but not cryptographic material or control programs.
2. Each party:
    1. Generates cleartext outputs: (amount, asset ID, control program, [REK](#record-encryption-key)). These include both requested payment ("I want to receive €10") and the change ("I send back to myself $142").
    2. [Encrypts](#encrypt-output) each output.
    3. [Balances blinding factors](#balance-blinding-factors) to create an excess factor `q[i]`.
    4. For each output, stores values `(assetid,value,AC,c,f)` associated with that output in its DB for later spending.
    5. Sends `q[i]` to the party that finalizes transaction.
3. Party that finalizes transaction:
    1. Receives `{q[i]}` values from all other parties (including itself).
    2. Sums all excess factors: `qsum = ∑q[i]`
    3. [Creates excess commitment](#create-excess-commitment) out of `qsum`: `QG=qsum·G, QJ=qsum·J` and signs it.
    4. Finalizes transaction by placing the [excess commitment](#excess-commitment) in the transaction.
    5. Publishes the transaction.
4. If the amounts and blinding factors are balanced, transaction is valid and included in the blockchain. Parties can not see each other’s outputs, but only their own.


## Cryptographic primitives

### Elliptic curve

**The elliptic curve** is edwards25519 as defined by [[RFC7748](https://tools.ietf.org/html/rfc7748)].

`L` is the **order of edwards25519** as defined by \[[RFC8032](https://tools.ietf.org/html/rfc8032)\]. 

L = 2<sup>252</sup>+27742317777372353535851937790883648493.

### Scalar

A _scalar_ is an integer in the range from `0` to `L-1` where `L` is the order of [edwards25519](#elliptic-curve) subgroup.
Scalars are encoded as little-endian 32-byte integers.

### Points

#### Point

A point is a two-dimensional point on [edwards25519](#elliptic-curve).
Points are encoded according to [RFC8032](https://tools.ietf.org/html/rfc8032).

#### Point Pair

A vector of two elliptic curve [points](#point). Point pair is encoded as 64-byte string composed of 32-byte encodings of each point.

#### Point operations

Elliptic curve *points* support two operations:

1. Addition/subtraction of points (`A+B`, `A-B`)
2. Scalar multiplication (`a·B`).

These operations are defined as in \[[RFC8032](https://tools.ietf.org/html/rfc8032)\].

*Point pairs* support the same operations defined as:

1. Sum of two pairs is a pair of sums:

        (A,B) + (C,D) == (A+C, B+D)

2. Multiplication of a pair by a [scalar](#scalar) is a pair of scalar multiplications of each point:

        x·(A,B) == (x·A,x·B)


#### Zero point

_Zero point_ `O` is a representation of the _point at infinity_, identity element in the [edwards25519](#elliptic-curve) subgroup. It is encoded as follows:

    O = 0x0100000000000000000000000000000000000000000000000000000000000000

#### Generators

**Primary generator point** (`G`) is the elliptic curve point specified as "B" in Section 5.1 of [[RFC8032](https://tools.ietf.org/html/rfc8032)].

Generator `G` has the following 32-byte encoding:

    G = 0x5866666666666666666666666666666666666666666666666666666666666666

**Secondary generator point** (`J`) is the elliptic curve point defined as decoded hash of the primary generator `G`:

    J = 8·Decode(SHA3-256(Encode(G)))

Generator `J` has the following 32-byte encoding:

    J = 0x00c774b875ed4e395ebb0782b4d93db838d3c4c0840bc970570517555ca71b77


### Hash functions

#### Hash256

`Hash256` is a secure hash function that takes a variable-length binary string `x` as input and outputs a 256-bit string.

    Hash256(x) = SHAKE128("ChainCA-256" || x, 32)

#### StreamHash

`StreamHash` is a secure extendable-output hash function that takes a variable-length binary string `x` as input
and outputs a variable-length hash string depending on a number of bytes (`n`) requested.

    StreamHash(x, n) = SHAKE128("ChainCA-stream" || x, n)

#### ScalarHash

`ScalarHash` is a secure hash function that takes a variable-length binary string `x` as input and outputs a [scalar](#scalar):

1. For the input string `x` compute a 512-bit hash `h`:

        h = SHAKE128("ChainCA-scalar" || x, 64)

2. Interpret `h` as a little-endian integer and reduce modulo subgroup [order](#elliptic-curve) `L`:

        s = h mod L

3. Return the resulting scalar `s`.


### Ring Signature

Ring signature is a variable-length string representing a signature of knowledge of a one private key among an ordered set of public keys (specified separately). In other words, ring signature implements an OR function of the public keys: “I know the private key for A, or B or C”. Ring signatures are used in [asset range proofs](#asset-range-proof).

The ring signature is encoded as a string of `n+1` 32-byte elements where `n` is the number of public keys provided separately (typically stored or imputed from the data structure containing the ring signature):

    {e, s[0], s[1], ..., s[n-1]}

Each 32-byte element is an integer coded using little endian convention. I.e., a 32-byte string `x` `x[0],...,x[31]` represents the integer `x[0] + 2^8 · x[1] + ... + 2^248 · x[31]`.

Ring signature described below supports proving knowledge of multiple discrete logarithms at once.

#### Create Ring Signature

**Inputs:**

1. `msg`: the string to be signed.
2. `M`: number of discrete logarithms to prove per signature (1 for normal signature, 2 for dlog equality proof).
3. `{B[u]}`: `M` base [points](#point) to validate the signature.
4. `{P[i,u]}`: `n·M` [points](#point) representing the public keys.
5. `j`: the index of the designated public key, so that `P[j] == p·B`.
6. `p`: the secret [scalar](#scalar) representing a private key for the public keys `P[u,j]`.

**Output:** `{e0, s[0], ..., s[n-1]}`: the ring signature, `n+1` 32-byte elements.

**Algorithm:**

1. Let `counter = 0`.
2. Let the `msghash` be a hash of the input non-secret data: `msghash = Hash256("RS" || byte(48+M) || B || P[0] || ... || P[n-1] || msg)`.
3. Calculate a sequence of: `n-1` 32-byte random values, 64-byte `nonce` and 1-byte `mask`: `{r[i], nonce, mask} = StreamHash(uint64le(counter) || msghash || p || uint64le(j), 32·(n-1) + 64 + 1)`, where:
    * `counter` is encoded as a 64-bit little-endian integer,
    * `p` is encoded as a 256-bit little-endian integer,
    * `j` is encoded as a 64-bit little-endian integer.
4. Calculate `k = nonce mod L`, where `nonce` is interpreted as a 64-byte little-endian integer and reduced modulo subgroup order `L`.
5. Calculate the initial e-value, let `i = j+1 mod n`:
    1. For each `u` from 0 to `M-1`: calculate `R[u,i]` as the [point](#point) `k·B[u]`.
    2. Define `w[j]` as `mask` with lower 4 bits set to zero: `w[j] = mask & 0xf0`.
    3. Calculate `e[i] = ScalarHash("e" || R[0,i] || ... || R[M-1,i] || msghash || uint64le(i) || w[j])` where `i` is encoded as a 64-bit little-endian integer.
6. For `step` from `1` to `n-1` (these steps are skipped if `n` equals 1):
    1. Let `i = (j + step) mod n`.
    2. Calculate the forged s-value `s[i] = r[step-1]`.
    3. Define `z[i]` as `s[i]` with the most significant 4 bits set to zero.
    4. Define `w[i]` as a most significant byte of `s[i]` with lower 4 bits set to zero: `w[i] = s[i][31] & 0xf0`.
    5. Let `i’ = i+1 mod n`.
    6. For each `u` from 0 to `M-1`:
        1. Calculate point `R[u,i’] = z[i]·B[u] - e[i]·P[i,u]`.
    7. Calculate `e[i’] = ScalarHash("e" || R[0,i’] || ... || R[M-1,i’] || msghash || uint64le(i’) || w[i])` where `i’` is encoded as a 64-bit little-endian integer.
7. Calculate the non-forged `z[j] = k + p·e[j] mod L` and encode it as a 32-byte little-endian integer.
8. If `z[j]` is greater than 2<sup>252</sup>–1, then increment the `counter` and try again from the beginning. The chance of this happening is below 1 in 2<sup>124</sup>.
9. Define `s[j]` as `z[j]` with 4 high bits set to high 4 bits of the `mask`.
10. Return the ring signature `{e[0], s[0], ..., s[n-1]}`, total `n+1` 32-byte elements.


#### Validate Ring Signature

**Inputs:**

1. `msg`: the string being signed.
2. `M`: number of discrete logarithms to prove per signature (1 for normal signature, 2 for dlog equality proof).
3. `{B[u]}`: `M` base [points](#point) to validate the signature.
4. `{P[i,u]}`: `n·M` [points](#point) representing the public keys.
5. `e[0], s[0], ... s[n-1]`: ring signature consisting of `n+1` 32-byte elements.


**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. Let the `msghash` be a hash of the input non-secret data: `msghash = Hash256("RS" || byte(48+M) || B || P[0] || ... || P[n-1] || msg)`.
2. For each `i` from `0` to `n-1`:
    1. Define `z[i]` as `s[i]` with the most significant 4 bits set to zero (see note below).
    2. Define `w[i]` as a most significant byte of `s[i]` with lower 4 bits set to zero: `w[i] = s[i][31] & 0xf0`.
    3. For each `u` from 0 to `M-1`:
        1. Calculate point `R[u,i+1] = z[i]·B[u] - e[i]·P[u,i]`.
    4. Calculate `e[i+1] = ScalarHash("e" || R[0,i+1] || ... || R[M-1,i+1] || msghash || i+1 || w[i])` where `i+1` is encoded as a 64-bit little-endian integer.
3. Return true if `e[0]` equals `e[n]`, otherwise return false.

Note: when the s-values are decoded as little-endian integers we must set their 4 most significant bits to zero in order to restore the original scalar as produced while [creating the range proof](#create-asset-range-proof). During signing the non-forged s-value has its 4 most significant bits set to random bits to make it indistinguishable from the forged s-values.





### Borromean Ring Signature

Borromean ring signature ([Maxwell2015](https://github.com/Blockstream/borromean_paper)) is a data structure representing several [ring signatures](#ring-signature) compactly joined with an AND function of the ring signatures: “I know the private key for (A or B) and (C or D)”. Borromean ring signatures are used in [value range proofs](#value-range-proof) that prove the range of multiple digits at once.

The borromean ring signature is encoded as a sequence of 32-byte elements.

    {e, s[i,j]...}

Where:

* `i` is in range `0..n` where `n` is the number of rings.
* `j` is in range `0..m` where `m` is the number of signatures per ring.

Example: a [value range proof](#value-range-proof) for a 4-bit mantissa has 9 elements in its borromean ring signature:

    {
      e,
      s[0,0], s[0,1], s[0,2], s[0,3], # base-4 digit at position 0 (proof for the lower two bits)
      s[1,0], s[1,1], s[1,2], s[1,3], # base-4 digit at position 1 (proof for the higher two bits)
    }

#### Create Borromean Ring Signature

**Inputs:**

1. `msg`: the string to be signed.
2. `n`: number of rings.
3. `m`: number of signatures in each ring.
4. `M`: number of discrete logarithms to prove per signature (1 for normal signature, 2 for dlog equality proof).
5. `{B[i,u]}`: `n·M` base [points](#point) to validate the signature.
6. `{P[i,j,u]}`: `n·m·M` [points](#point) representing public keys.
7. `{p[i]}`: the list of `n` [scalars](#scalar) representing private keys.
8. `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[i,j] == p[i]·B[i]`.
9. `{payload[i]}`: sequence of `n·m` random 32-byte elements.

**Output:** `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.

**Algorithm:**

1. Let the `msghash` be a hash of the input non-secret data: `msghash = Hash256("BRS" || byte(48+M) || uint64le(n) || uint64le(m) || {B[i]} || {P[i,j]} || msg)` where `n` and `m` are encoded as 64-bit little-endian integers.
2. Let `counter = 0`.
3. Let `cnt` byte contain lower 4 bits of `counter`: `cnt = counter & 0x0f`.
4. Calculate a sequence of `n·m` 32-byte random overlay values: `{o[i]} = StreamHash("O" || uint64le(counter) || msghash || {p[i]} || {uint64le(j[i])}, 32·n·m)`, where:
    * `counter` is encoded as a 64-bit little-endian integer,
    * private keys `{p[i]}` are encoded as concatenation of 256-bit little-endian integers,
    * secret indexes `{j[i]}` are encoded as concatenation of 64-bit little-endian integers.
5. Define `r[i] = payload[i] XOR o[i]` for all `i` from 0 to `n·m - 1`.
6. For `t` from `0` to `n-1` (each ring):
    1. Let `j = j[t]`
    2. Let `x = r[m·t + j]` interpreted as a little-endian integer.
    3. Define `k[t]` as the lower 252 bits of `x`.
    4. Define `mask[t]` as the higher 4 bits of `x`.
    5. Define `w[t,j]` as a byte with lower 4 bits set to zero and higher 4 bits equal `mask[t]`.
    6. Calculate the initial e-value for the ring:
        1. Let `j’ = j+1 mod m`.
        2. For each `u` from 0 to `M-1`:
            1. Calculate `R[t,j’,u]` as the point `k[t]·B[t,u]`.
        3. Calculate `e[t,j’] = ScalarHash("e" || byte(cnt), R[t,j’,0] || ... || R[t,j’,M-1] || msghash || uint64le(t) || uint64le(j’) || w[t,j])` where `t` and `j’` are encoded as 64-bit little-endian integers.
    7. If `j ≠ m-1`, then for `i` from `j+1` to `m-1`:
        1. Calculate the forged s-value: `s[t,i] = r[m·t + i]`.
        2. Define `z[t,i]` as `s[t,i]` with 4 most significant bits set to zero.
        3. Define `w[t,i]` as a most significant byte of `s[t,i]` with lower 4 bits set to zero: `w[t,i] = s[t,i][31] & 0xf0`.
        4. Let `i’ = i+1 mod m`.
        5. For each `u` from 0 to `M-1`:
            1. Calculate point `R[t,i’,u] = z[t,i]·B[t,u] - e[t,i]·P[t,i,u]`.
        6. Calculate `e[t,i’] = ScalarHash("e" || byte(cnt), R[t,i’,0] || ... || R[t,i’,M-1] || msghash || uint64le(t) || uint64le(i’) || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
7. Calculate the shared e-value `e0` for all the rings:
    1. Calculate `E` as concatenation of all `e[t,0]` values encoded as 32-byte little-endian integers: `E = e[0,0] || ... || e[n-1,0]`.
    2. Calculate `e0 = ScalarHash(E)`.
    3. If `e0` is greater than 2<sup>252</sup>–1, then increment the `counter` and try again from step 3. The chance of this happening is below 1 in 2<sup>124</sup>.
8. For `t` from `0` to `n-1` (each ring):
    1. Let `j = j[t]`.
    2. Let `e[t,0] = e0`.
    3. If `j` is not zero, then for `i` from `0` to `j-1`:
        1. Calculate the forged s-value: `s[t,i] = r[m·t + i]`.
        2. Define `z[t,i]` as `s[t,i]` with 4 most significant bits set to zero.
        3. Define `w[t,i]` as a most significant byte of `s[t,i]` with lower 4 bits set to zero: `w[t,i] = s[t,i][31] & 0xf0`.
        4. Let `i’ = i+1 mod m`.
        5. For each `u` from 0 to `M-1`:
            1. Calculate point `R[t,i’,u] = z[t,i]·B[t,u] - e[t,i]·P[t,i,u]`. If `i` is zero, use `e0` in place of `e[t,0]`.
        6. Calculate `e[t,i’] = ScalarHash("e" || byte(cnt), R[t,i’,0] || ... || R[t,i’,M-1] || msghash || uint64le(t) || uint64le(i’) || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
    4. Calculate the non-forged `z[t,j] = k[t] + p[t]·e[t,j] mod L` and encode it as a 32-byte little-endian integer.
    5. If `z[t,j]` is greater than 2<sup>252</sup>–1, then increment the `counter` and try again from step 3. The chance of this happening is below 1 in 2<sup>124</sup>.
    6. Define `s[t,j]` as `z[t,j]` with 4 high bits set to `mask[t]` bits.
9. Set top 4 bits of `e0` to the lower 4 bits of `counter`.
10. Return the [borromean ring signature](#borromean-ring-signature):
    * `{e,s[t,j]}`: `n·m+1` 32-byte elements.



#### Validate Borromean Ring Signature

**Inputs:**

1. `msg`: the string to be signed.
2. `n`: number of rings.
3. `m`: number of signatures in each ring.
4. `M`: number of discrete logarithms to prove per signature (1 for normal signature, 2 for dlog equality proof).
5. `{B[i,u]}`: `n·M` base [points](#point) to validate the signature.
6. `{P[i,j,u]}`: `n·m·M` [points](#point) representing public keys.
7. `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. Let the `msghash` be a hash of the input non-secret data: `msghash = SHA3-256("BRS" || byte(48+M) || uint64le(n) || uint64le(m) || {B[i]} || {P[i,j]} || msg)` where `n` and `m` are encoded as 64-bit little-endian integers.
2. Define `E` to be an empty binary string.
3. Set `cnt` byte to the value of top 4 bits of `e0`: `cnt = e0[31] >> 4`.
4. Set top 4 bits of `e0` to zero.
5. For `t` from `0` to `n-1` (each ring):
    1. Let `e[t,0] = e0`.
    2. For `i` from `0` to `m-1` (each item):
        1. Calculate `z[t,i]` as `s[t,i]` with the most significant 4 bits set to zero.
        2. Calculate `w[t,i]` as a most significant byte of `s[t,i]` with lower 4 bits set to zero: `w[t,i] = s[t,i][31] & 0xf0`.
        3. Let `i’ = i+1 mod m`.
        5. For each `u` from 0 to `M-1`:
            1. Calculate point `R[t,i’,u] = z[t,i]·B[t,u] - e[t,i]·P[t,i,u]`. Use `e0` instead of `e[t,0]` in each ring.
        6. Calculate `e[t,i’] = ScalarHash("e" || byte(cnt) || R[t,i’,0] || ... || R[t,i’,M-1] || msghash || uint64le(t) || uint64le(i’) || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
    3. Append `e[t,0]` to `E`: `E = E || e[t,0]`, where `e[t,0]` is encoded as a 32-byte little-endian integer.
6. Calculate `e’ = ScalarHash(E)`.
7. Return `true` if `e’` equals to `e0`. Otherwise, return `false`.



#### Recover Payload From Borromean Ring Signature

**Inputs:**

1. `msg`: the string to be signed.
2. `n`: number of rings.
3. `m`: number of signatures in each ring.
4. `M`: number of discrete logarithms to prove per signature (1 for normal signature, 2 for dlog equality proof).
5. `{B[i,u]}`: `n·M` base [points](#point) to validate the signature.
6. `{P[i,j,u]}`: `n·m·M` [points](#point) representing public keys.
7. `{p[i]}`: the list of `n` scalars representing private keys.
8. `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[i,j] == p[i]·G`.
9. `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.

**Output:** `{payload[i]}` list of `n·m` random 32-byte elements or `nil` if signature verification failed.

**Algorithm:**

1. Let the `msghash` be a hash of the input non-secret data: `msghash = SHA3-256("BRS" || byte(48+M) || n || m || {B[i]} || {P[i,j]} || msg)` where `n` and `m` are encoded as 64-bit little-endian integers.
2. Define `E` to be an empty binary string.
3. Set `cnt` byte to the value of top 4 bits of `e0`: `cnt = e0[31] >> 4`.
4. Let `counter` integer equal `cnt`.
5. Calculate a sequence of `n·m` 32-byte random overlay values:

        `{o[i]} = StreamHash("O" || uint64le(counter) || msghash || {p[i]} || {uint64le(j[i])}, 32·n·m)`, where:

6. Set top 4 bits of `e0` to zero.
7. For `t` from `0` to `n-1` (each ring):
    1. Let `e[t,0] = e0`.
    2. For `i` from `0` to `m-1` (each item):
        1. Calculate `z[t,i]` as `s[t,i]` with the most significant 4 bits set to zero.
        2. Calculate `w[t,i]` as a most significant byte of `s[t,i]` with lower 4 bits set to zero: `w[t,i] = s[t,i][31] & 0xf0`.
        3. If `i` is equal to `j[t]`:
            1. Calculate `k[t] = z[t,i] - p[t]·e[t,i] mod L`.
            2. Set top 4 bits of `k[t]` to the top 4 bits of `w[t,i]`: `k[t][31] |= w[t,i]`.
            3. Set `payload[m·t + i] = o[m·t + i] XOR k[t]`.
        4. If `i` is not equal to `j[t]`:
            1. Set `payload[m·t + i] = o[m·t + i] XOR s[t,i]`.
        5. Let `i’ = i+1 mod m`.
        6. For each `u` from 0 to `M-1`:
            1. Calculate point `R[t,i’,u] = z[t,i]·B[t,u] - e[t,i]·P[t,i,u]` and encode it as a 32-byte [public key](#point). Use `e0` instead of `e[t,0]` in each ring.
        7. Calculate `e[t,i’] = ScalarHash("e" || byte(cnt) || R[t,i’,0] || ... || R[t,i’,M-1] || msghash || uint64le(t) || uint64le(i’) || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
    3. Append `e[t,0]` to `E`: `E = E || e[t,0]`, where `e[t,0]` is encoded as a 32-byte little-endian integer.
8. Calculate `e’ = ScalarHash(E)`.
9. Return `payload` if `e’` equals to `e0`. Otherwise, return `nil`.






## Keys

### 2D key

*Two-dimensional key* (“2D key” or “2DK”) is a pair of 32-byte secret strings:

    {x, y}

2DK is encoded as a 64-byte concatenation of the corresponding secret strings (32 bytes each):

    2dk = x || y

2DKs allows “two-dimensional” key derivation by deriving from one key and leaving the other one unchanged, therefore allowing “vertical” derivation from [Record Encryption Keys](#record-encryption-key) to [Intermediate Encryption Keys](#intermediate-encryption-key), [Asset ID Encryption Keys](#asset-id-encryption-key) and [Value Encryption Keys](#value-encryption-key) and “horizontal” derivation from a root key pair associated with a user’s account to per-transaction and per-output key pairs.


### Record Encryption Key

Record encryption key (REK or `rek`) is a [2D key](#2d-key):

    rek = {x, y}

It is used to decrypt the payload data from the [value range proof](#value-range-proof), and derive [asset ID encryption key](#asset-id-encryption-key) and [value encryption key](#value-encryption-key).

The `y` is used to derive more specific keys as described below that all share the same second key `x`.
The `x` is used to derive the entire hierarchies of encryption keys, so that a [REK](#record-encryption-key), or [IEK](#intermediate-encryption-key) could be shared for the entire account instead of per-transaction.


### Intermediate Encryption Key

Intermediate encryption key (IEK or `iek`) is a [2D key](#2d-key) that allows decrypting the asset ID and the value in the output commitment. It is derived from the [record encryption key](#record-encryption-key) as follows:

    iek = {x: rek.x, y: Hash256("IEK", rek.y)}

### Asset ID Encryption Key

Asset ID encryption key (AEK or `aek`) is a [2D key](#2d-key) that allows decrypting the asset ID in the output commitment. It is derived from the [intermediate encryption key](#intermediate-encryption-key) as follows:

    aek = {x: iek.x, y: Hash256("AEK", iek.y)}

### Value Encryption Key

Value encryption key (VEK or `vek`) is a [2D key](#2d-key) that allows decrypting the amount in the output commitment. It is derived from the [intermediate encryption key](#intermediate-encryption-key) as follows:

    vek = {x: iek.x, y: Hash256("VEK", iek.y)}



### Asset ID Blinding Factor

A [scalar](#scalar) `c` used to produce a blinded asset ID commitment out of a [cleartext asset ID commitment](#asset-id-commitment):

    AC = (A + c·G, c·J)

The asset ID blinding factor is created by [Create Blinded Asset ID Commitment](#create-blinded-asset-id-commitment).


### Value Blinding Factor

An [scalar](#scalar) `f` used to produce a [value commitment](#value-commitment) out of the [asset ID commitment](#asset-id-commitment) `(H, C)`:

    VC = (value·H + f·G, value·C + f·J)

The value blinding factors are created by [Create Blinded Value Commitment](#create-blinded-value-commitment) algorithm.


### Transient Issuance Key

Transient issuance key is created for the [confidential issuance proof](#issuance-asset-range-proof) when there is no pre-defined issuance key.

**Inputs:**

1. `assetid`: asset ID for which the issuance key is being generated.
2. `aek`: [asset ID encryption key](#asset-id-encryption-key).

**Output:** `(y,Y)`: a pair of private and public keys.

**Algorithm:**

1. Calculate scalar `y = ScalarHash("IARP.y" || assetid || aek)`.
2. Calculate point `Y` by multiplying base point by `y`: `Y = y·G`.
3. Return key pair `(y,Y)`.





## Commitments

### Asset ID Point

_Asset ID point_ is a [point](#point) representing an asset ID.

It is defined as follows:

1. Let `counter = 0`.
2. Calculate `Hash256("AssetID" || assetID || uint64le(counter))` where `counter` is encoded as a 64-bit unsigned integer using little-endian convention.
3. Decode the resulting hash as a [point](#point) `P` on the elliptic curve.
4. If the point is invalid, increment `counter` and go back to step 2. This will happen on average for half of the asset IDs.
5. Calculate point `A = 8·P` (8 is a cofactor in edwards25519) which belongs to a subgroup [order](#elliptic-curve) `L`.
6. Return `A`.


### Asset ID Commitment

An asset ID commitment `AC` is an ElGamal commitment represented by a [point pair](#point-pair):

    AC = (H, C)

    H  = A + c·G
    C  = c·J

where:

* `A` is an [Asset ID Point](#asset-id-point), an orthogonal point representing an asset ID.
* `c` is a blinding [scalar](#scalar) for the asset ID.
* `G`, `J` are [generator points](#generators).

The asset ID commitment can either be nonblinded or blinded.

#### Create Nonblinded Asset ID Commitment

**Input:** `assetID`: the cleartext asset ID.

**Output:**  `(A,O)`: the nonblinded [asset ID commitment](#asset-id-commitment).

**Algorithm:**

1. Compute an [asset ID point](#asset-id-point):

        A = 8·Hash256(assetID || counter)

2. Return [point pair](#point-pair) `(A,O)` where `O` is a [zero point](#zero-point).


#### Create Blinded Asset ID Commitment

**Inputs:**

1. `assetID`: the cleartext asset ID.
2. `aek`: the [asset ID encryption key](#asset-id-encryption-key).

**Outputs:**

1. `(H,C)`: the blinded [asset ID commitment](#asset-id-commitment).
2. `c`: the [blinding factor](#asset-id-blinding-factor) such that `H == A + c·G, C = c·J`.

**Algorithm:**

1. Compute an [asset ID point](#asset-id-point):

        A = 8·Decode(Hash256(assetID...))

2. Compute [asset ID blinding factor](#asset-id-blinding-factor):

        c = ScalarHash("AC.c" || assetID || aek)

3. Compute an [asset ID commitment](#asset-id-commitment):

        AC = (H, C)
        H  = A + c·G
        C  = c·J

4. Return `(AC, c)`.


### Value Commitment

A value commitment `VC` is an ElGamal commitment represented by a [point pair](#point-pair):

    VC = (V, F)

    V  = v·H + f·G
    F  = v·C + f·J

where:

* `(H, C)` is an [asset ID commitment](#asset-id-commitment).
* `v` is an amount being committed.
* `f` is a blinding [scalar](#scalar) for the amount.
* `G`, `J` are [generator points](#generators).

The asset ID commitment can either be _nonblinded_ or _blinded_:

#### Create Nonblinded Value Commitment

**Inputs:**

1. `value`: the cleartext amount,
2. `(H,C)`: the [asset ID commitment](#asset-id-commitment).

**Output:** the value commitment `VC` represented by a [point pair](#point-pair).

**Algorithm:**

1. Calculate [point pair](#point-pair) `VC = value·(H, C)`.
2. Return `VC`.


#### Create Blinded Value Commitment

**Inputs:**

1. `vek`: the [value encryption key](#value-encryption-key) for the given output,
2. `value`: the amount to be blinded in the output,
3. `(H,C)`: the [asset ID commitment](#asset-id-commitment).

**Output:** `(VC, f)`: the pair of a [value commitment](#value-commitment) and its blinding factor.

**Algorithm:**

1. Calculate `f = ScalarHash("VC.f" || uint64le(value) || vek)`.
2. Calculate point `V = value·H + f·G`.
3. Calculate point `F = value·C + f·J`.
4. Create a [point pair](#point-pair): `VC = (V, F)`.
5. Return `(VC, f)`.



### Excess Factor

Excess factor is a [scalar](#scalar) representing a net difference between input and output blinding factors. 
It is computed by [balancing blinding factors](#balance-blinding-factors) and used to create an [Excess Commitment](#excess-commitment).

#### Balance Blinding Factors

**Inputs:**

1. The list of `n` input tuples `{(value[j], c[j], f[j])}`, where:
    * `value[j]`: the amount blinded in the j-th input,
    * `c[j]`: the [blinding factor](#asset-id-blinding-factor) in the j-th input (so that `H[j] = A[j] + c[j]·G`),
    * `f[j]`: the [value blinding factor](#value-blinding-factor) used in the j-th input (so that `V[j] = value[j]·H[j] + f[j]·G`).
2. The list of `m` output tuples `{(value’[i], c’[i], f’[i])}`, where:
    * `value’[i]`: the amount blinded in the i-th output,
    * `c’[i]`: the [blinding factor](#asset-id-blinding-factor) in the i-th output (so that `H’[i] = A’[i] + c’[i]·G`),
    * `f’[i]`: the [value blinding factor](#value-blinding-factor) used in the i-th output (so that `V’[i] = value’[i]·H’[i] + f’[i]·G`).

**Output:** `q`: the [excess blinding factor](#excess-factor) that must be added to the output blinding factors in order to balance inputs and outputs.

**Algorithm:**

1. Calculate the sum of input blinding factors: `Finput = ∑(value[j]·c[j]+f[j], j from 0 to n-1) mod L`.
2. Calculate the sum of output blinding factors: `Foutput = ∑(value’[i]·c’[i]+f’[i], i from 0 to m-1) mod L`.
3. Calculate the [excess blinding factor](#excess-factor) as difference between input and output sums: `q = Finput - Foutput mod L`.
4. Return `q`.


### Excess Commitment

An excess commitment `QC` is an ElGamal commitment to an [excess factor](#excess-factor) represented by a [point pair](#point-pair) together with a Schnorr signature proving the equality of the discrete logarithms in both points (`e,s`):

    QC = (q·G, q·J, e, s)

Excess pair `(q·G, q·J)` is used to [validate balance of value commitments](#validate-value-commitments-balance) while the associated signature proves that the points do not contain a factor affecting the amount of any asset.

#### Create Excess Commitment

**Inputs:**

1. `q`: the [excess blinding factor](#excess-factor)
2. `message`: a variable-length string.

**Output:**

1. `(QG,QJ)`: the [point pair](#point-pair) representing an ElGamal commitment to `q` using [generators](#generators) `G` and `J`.
2. `(e,s)`: the Schnorr signature proving that `(QG,QJ)` does not affect asset amounts.

**Algorithm:**

1. Calculate a [point pair](#point-pair):
        
        QG = q·G
        QJ = q·J

2. Calculate the nonce:

        r = ScalarHash("r" || QG || QJ || q || message)

3. Calculate points:
    
        R1 = r·G
        R2 = r·J

4. Calculate Schnorr challenge scalar:

        e = ScalarHash("EC" || QG || QJ || R1 || R2 || message)

5. Calculate Schnorr response scalar:

        s = k + q·e mod L

6. Return pair of scalars `(s,e)`.


#### Validate Excess Commitment

**Inputs:**

1. `(QG,QJ)`: the [point pair](#point-pair) representing an ElGamal commitment to secret blinding factor `q` using [generators](#generators) `G` and `J`.
2. `(e,s)`: the Schnorr signature proving that `(QG,QJ)` does not affect asset amounts.
3. `message`: a variable-length string.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. Calculate points:

        R1 = s·G - e·QG
        R2 = s·J - e·QJ

2. Calculate Schnorr challenge:

        e’ = ScalarHash("EC" || QG || QJ || R1 || R2 || message)

4. Return `true` if `e’ == e`, otherwise return `false`.


### Validate Value Commitments Balance

**Inputs:**

1. The list of `n` input value commitments `{VC[i]}`.
2. The list of `m` output value commitments `{VC’[i]}`.
3. The list of `k` [excess commitments](#excess-commitment) `{(QC[i], s[i], e[i])}`.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. [Validate](#validate-excess-commitment) each of `k` [excess commitments](#excess-commitment); if any is not valid, halt and return `false`.
2. Calculate the sum of input value commitments:
        
        Ti = ∑(VC[i], j from 0 to n-1)

3. Calculate the sum of output value commitments:

        To = ∑(VC’[i], i from 0 to m-1)

4. Calculate the sum of excess commitments:

        Tq = ∑[(QG[i], QJ[i]), i from 0 to k-1]

5. Return `true` if `Ti == To + Tq`, otherwise return `false`.






## Proofs


### Asset Range Proof

The asset range proof (ARP) demonstrates that a given [asset ID commitment](#asset-id-commitment) commits to one of the asset IDs specified in the transaction inputs. A [whole-transaction validation procedure](#validate-assets-flow) makes sure that all of the declared asset ID commitments in fact belong to the transaction inputs.

Asset range proof can be [non-confidential](#non-confidential-asset-range-proof) or [confidential](#confidential-asset-range-proof).

#### Non-Confidential Asset Range Proof

Field                        | Type      | Description
-----------------------------|-----------|------------------
Type                         | byte      | Contains value 0x00 to indicate the commitment is not blinded.
Asset ID Commitments         | [List](blockchain.md#list)\<[Asset ID Commitment](#asset-id-commitment)\> | List of asset ID commitments from the transaction inputs used in the range proof.
Asset Ring Signature         | [Ring Signature](#ring-signature) | A ring signature proving that the asset ID committed in the output belongs to the set of declared input commitments.
Asset ID                     | [AssetID](blockchain.md#asset-id) | 32-byte asset identifier.

Note: non-confidential asset range proof exposes the asset ID, but
still needs to prove that it belongs to the set of the admissible
asset IDs in order to provide soundness guarantees.


#### Confidential Asset Range Proof

Field                        | Type      | Description
-----------------------------|-----------|------------------
Type                         | byte      | Contains value 0x01 to indicate the commitment is blinded.
Asset ID Commitments         | [List](blockchain.md#list)\<[Asset ID Commitment](#asset-id-commitment)\> | List of asset ID commitments from the transaction inputs used in the range proof.
Asset Ring Signature         | [Ring Signature](#ring-signature) | A ring signature proving that the asset ID committed in the output belongs to the set of declared input commitments.


#### Create Asset Range Proof

**Inputs:**

1. `AC’`: the output [asset ID commitment](#asset-id-commitment) for which the range proof is being created.
2. `{AC[i]}`: `n` candidate [asset ID commitments](#asset-id-commitment).
3. `j`: the index of the designated commitment among the input asset ID commitments, so that `AC’ == AC[j] + (c’ - c)·(G,J)`.
4. `c’`: the [blinding factor](#asset-id-blinding-factor) for the commitment `AC’`.
5. `c`: the [blinding factor](#asset-id-blinding-factor) for the candidate commitment `AC[j]`.
6. `message`: a variable-length string.

**Output:** an [asset range proof](#asset-range-proof) consisting of a list of input asset ID commitments and a ring signature.

**Algorithm:**

1. Calculate the message hash to sign:

        msghash = Hash256("ARP" || AC’ || AC[0] || ... || AC[n-1] || message)

2. Calculate the set of public keys for the ring signature from the set of input asset ID commitments:

        P[i] = AC’.H - AC[i].H
        Q[i] = AC’.C - AC[i].C

3. Calculate the private key: `p = c’ - c mod L`.
4. [Create a ring signature](#create-ring-signature) using `msghash`, [generators](#generators) `(G,J)`, `{(P[i], Q[i])}`, `j`, and `p`.
5. Return the list of asset ID commitments `{AC[i]}` and the ring signature `e[0], s[0], ... s[n-1]`.

Note: unlike the [value range proof](#value-range-proof), this ring signature
is not used to store encrypted payload data because decrypting it would reveal
the asset ID of one of the inputs to the recipient.


#### Validate Asset Range Proof

**Inputs:**

1. `AC’`: the target [asset ID commitment](#asset-id-commitment).
2. One of the two [asset range proofs](#asset-range-proof):
    * A _confidential_ asset range proof consisting of:
        1. `{AC[i]}`: `n` input [asset ID commitments](#asset-id-commitment).
        2. `e[0], s[0], ... s[n-1]`: the ring signature.
    * A _non-confidential_ asset range proof consisting of:
        1. `{AC[i]}`: `n` input [asset ID commitments](#asset-id-commitment).
        2. `e[0], s[0], ... s[n-1]`: the ring signature.
        3. An [asset ID](blockchain.md#asset-id).
3. Provided separately:
    * `message`: a variable-length string.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. Calculate the message hash to sign:

        msghash = Hash256("ARP" || AC’ || AC[0] || ... || AC[n-1] || message)

2. Calculate the set of public keys for the ring signature from the set of input asset ID commitments:

        P[i] = AC’.H - AC[i].H
        Q[i] = AC’.C - AC[i].C

3. [Validate the ring signature](#validate-ring-signature) `e[0], s[0], ... s[n-1]` with `msg`, [generators](#generators) `(G,J)` and `{(P[i],Q[i])}`.
4. If verification was unsuccessful, return `false`.
5. If the asset range proof is non-confidential:
    1. Compute [asset ID point](#asset-id-point): `A’ = 8·Hash256(assetID || counter)`.
    2. Verify that [point pair](#point-pair) `(A’,O)` equals `AC’`.
6. Return `true`.





### Issuance Asset Range Proof WIP

The issuance asset range proof demonstrates that a given [confidential issuance](#confidential-issuance)
commits to one of the asset IDs specified in the transaction inputs.
Some inputs to the [validation procedure](#validate-issuance-asset-range-proof) are computed 
from other elements in the confidential issuance witness, as part of the [issuance validation procedure](blockchain.md#issuance-2-validation).

The size of the ring signature (`n+1` 32-byte elements) and the number of issuance keys (`n`)
are derived from `n` [asset issuance choices](blockchain.md#asset-issuance-choice) specified outside the range proof.

The proof also contains a _tracing point_ that that lets any issuer to prove or disprove whether the issuance is performed by their issuance key.

#### Non-Confidential Issuance Asset Range Proof WIP

Field                        | Type      | Description
-----------------------------|-----------|------------------
Type                         | byte      | Contains value 0x00 to indicate the commitment is not blinded.
Asset ID                     | [AssetID](blockchain.md#asset-id)   | 32-byte asset identifier.

#### Confidential Issuance Asset Range Proof WIP

Field                           | Type             | Description
--------------------------------|------------------|------------------
Type                            | byte             | Contains value 0x01 to indicate the commitment is blinded.
Issuance Keys                   | [List](blockchain.md#list)\<[Point](#point)\> | Keys to be used to calculate the public key for the corresponding index in the ring signature.
Tracing Point                   | [Point](#point)  | A point that lets any issuer to prove or disprove if this issuance is done by them.
Issuance Ring Signature         | [Ring Signature](#ring-signature)   | A ring signature proving that the issuer of an encrypted asset ID approved the issuance.


#### Create Issuance Asset Range Proof WIP

When creating a confidential issuance, the first step is to construct the rest of the input commitment and input witness, including an asset issuance choice for each asset that one wants to include in the anonymity set. The issuance key for each asset should be extracted from the [issuance programs](blockchain.md#program). (Issuance programs that support confidential issuance should have a branch that checks use of the correct issuance key using `ISSUANCEKEY` instruction.)

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `c`: the [blinding factor](#asset-id-blinding-factor) for commitment `AC` such that: `AC.H == A[j] + c·G`, `AC.C == c·J`.
3. `{a[i]}`: `n` 32-byte unencrypted [asset IDs](blockchain.md#asset-id).
4. `{Y[i]}`: `n` issuance keys encoded as [points](#point) corresponding to the asset IDs,
5. `message`: a variable-length string,
6. `nonce`: unique 32-byte [string](blockchain.md#string) that makes the tracing point unique,
7. `j`: the index of the asset being issued (such that `AC.H == A[j] + c·G`).
8. `y`: the private key for the issuance key corresponding to the asset being issued: `Y[j] = y·G`.

**Output:** an [issuance asset range proof](#issuance-asset-range-proof) consisting of:

* `{Y[i]}`: `n` issuance keys encoded as [points](#point) corresponding to the asset IDs,
* `T`: tracing [point](#point),
* `rs = {e[0], s[0], ... s[n-1]}`: the issuance ring signature.


**Algorithm:**

1. Calculate the base hash: `basehash = Hash256("IARP" || AC || uint64le(n) || a[0] || ... || a[n-1] || Y[0] || ... || Y[n-1] || nonce || message)` where `n` is encoded as a 64-bit unsigned little-endian integer.
2. Calculate marker point `M`:
    1. Let `counter = 0`.
    2. Calculate `Hash256("M" || basehash || uint64le(counter))` where `counter` is encoded as a 64-bit unsigned integer using little-endian convention.
    3. Decode the resulting hash as a [point](#point) `P` on the elliptic curve.
    4. If the point is invalid, increment `counter` and go back to step 2. This will happen on average for half of the asset IDs.
    5. Calculate point `M = 8·P` (8 is a cofactor in edwards25519) which belongs to a subgroup [order](#elliptic-curve) `L`.
3. Calculate the tracing point: `T = y·(J + M)`.
4. Calculate the blinded marker using the blinding factor used by commitment `AC`: `Bm = c·M`.
5. Calculate a 32-byte message hash to sign: `msghash = Hash256("msg" || basehash || M || T || Bm)`.
6. Create proof that the discrete log `Bm/M` is equal to the discrete log `AC.C/J`:
    1. Calculate the nonce `k = ScalarHash("k" || msghash || c)`.
    2. Calculate points `R1 = k·M` and `R2 = k·J`.
    3. Calculate scalar `e’ = ScalarHash("e" || msghash || R1 || R2)`.
    4. Calculate scalar `s’ = k + c·e mod L`.
    5. Let the marker signature `ms = (e’,s’)`.
7. Calculate [asset ID points](#asset-id-point) for each `{a[i]}`: `A[i] = 8·Decode(Hash256(a[i]...))`.
8. Calculate Fiat-Shamir challenge `h` for the issuance key:

        h = ScalarHash("h" || msghash)

9. Calculate point `Q = C + Bm + h·T`.
10. Calculate points `{P[i]}` for `n` pairs of asset ID points and corresponding issuance keys `A[i], Y[i]`:

        P[i] = AC.H — A[i] + h·Y[i]

11. Create ring proof of discrete log equality for the pair `P[j]/G` and `Q/(J+M)`:
    1. Calculate the signing key `x = c + h2·y`.
    2. [Create a ring signature](#create-ring-signature) `rs` using:
        * message `msghash`,
        * base points `G, J+M`,
        * public keys `{(P[i], Q)}`,
        * secret index `j`,
        * private key `x`.
12. Return [issuance asset range proof](#issuance-asset-range-proof) consisting of:
    * issuance keys `{Y[i]}`,
    * tracing point `T`,
    * blinded marker point `Bm`,
    * marker signature `ms`,
    * ring signature `rs`.



#### Validate Issuance Asset Range Proof WIP

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `IARP`: the to-be-verified [issuance asset range proof](#issuance-asset-range-proof) consisting of:
    1. If the `IARP` is non-confidential: only `assetid`.
    2. If the `IARP` is confidential:
        * `{Y[i]}`: `n` issuance keys encoded as [points](#point) corresponding to the asset IDs,
        * `T`: tracing [point](#point),
        * `Bm`: blinded marker [point](#point),
        * `ms = (e’,s’)`: the marker signature,
        * `rs = {e[0], s[0], ... s[n-1]}`: the issuance ring signature,
        * And provided separately from the range proof:
            * `{a[i]}`: `n` [asset IDs](blockchain.md#asset-id),
            * `message`: a variable-length string,
            * `nonce`: unique 32-byte [string](blockchain.md#string) that makes the tracing point unique.


**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. If the range proof is non-confidential:
    1. Compute [asset ID point](#asset-id-point): `A’ = 8·Decode(Hash256(assetID...))`.
    2. Verify that [point pair](#point-pair) `(A’,O)` equals `AC`.
2. If the range proof is confidential:
    1. Calculate the base hash: `basehash = Hash256("IARP" || AC || uint64le(n) || a[0] || ... || a[n-1] || Y[0] || ... || Y[n-1] || nonce || message)` where `n` is encoded as a 64-bit unsigned little-endian integer.
    2. Calculate marker point `M`:
        1. Let `counter = 0`.
        2. Calculate `Hash256("M" || basehash || uint64le(counter))` where `counter` is encoded as a 64-bit unsigned integer using little-endian convention.
        3. Decode the resulting hash as a [point](#point) `P` on the elliptic curve.
        4. If the point is invalid, increment `counter` and go back to step 2. This will happen on average for half of the asset IDs.
        5. Calculate point `M = 8·P` (8 is a cofactor in edwards25519) which belongs to a subgroup [order](#elliptic-curve) `L`.
    3. Calculate a 32-byte message hash to sign: `msghash = Hash256("msg" || basehash || M || T || Bm)`.
    4. Validate proof that the discrete log `Bm/M` is equal to the discrete log `AC.C/J`:
        1. Calculate point `R1 = s’·M - e’·Bm`.
        2. Calculate point `R2 = s’·J - e’·AC.C`.
        3. Calculate scalar `e” = ScalarHash("e" || msghash || R1 || R2)`.
        4. Verify that `e”` is equal to `e’`.
    5. Calculate [asset ID points](#asset-id-point) for each `{a[i]}`: `A[i] = 8·Decode(Hash256(a[i]...))`.
    6. Calculate Fiat-Shamir challenge `h` for the issuance key:

            h = ScalarHash("h" || msghash)

    7. Calculate point `Q = C + Bm + h·T`.
    8. Calculate points `{P[i]}` for `n` pairs of asset ID points and corresponding issuance keys `A[i], Y[i]`:

            P[i] = AC.H — A[i] + h·Y[i]

    9. Validate ring proof of discrete log equality for one of the pairs `P[i]/G` and `Q/(J+M)` by [validating the ring signature](#validate-ring-signature) `e[0], s[0], ... s[n-1]` with:
        * message `msghash`,
        * base points `G, J+M`,
        * public keys `{(P[i], Q)}`.

### Issuance Proof

Issuance proof allows an issuer to prove whether a given confidential issuance is performed with their key or not.

Field                           | Type             | Description
--------------------------------|------------------|------------------
Blinding factor commitment      | [Point](#point)  | A point `X = x·(J+M)` that commits to a blinding scalar `x` used in this proof.
Blinded suspect tracing point   | [Point](#point)  | A point `Z = x·T` that blinds a suspected tracing point (published in a given [IARP](#issuance-asset-range-proof)).
Blinded actual tracing point    | [Point](#point)  | A point `Z’ = x·T’` that blinds a tracing point actually produced by the current issuer for the use in this proof.
Signature 1                     | 64 bytes         | A pair of [scalars](#scalar) representing a single Schnorr signature.
Signature 2                     | 64 bytes         | A pair of [scalars](#scalar) representing a single Schnorr signature.


#### Create Issuance Proof WIP

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `IARP`: the to-be-verified [confidential issuance asset range proof](#confidential-issuance-asset-range-proof) consisting of:
    * `{Y[i]}`: `n` issuance keys encoded as [points](#point) corresponding to the asset IDs,
    * `T`: tracing [point](#point),
    * `Bm`: blinded marker [point](#point),
    * `ms = (e’,s’)`: the marker signature,
    * `rs = {e[0], s[0], ... s[n-1]}`: the issuance ring signature,
    * And provided separately from the range proof:
        * `{a[i]}`: `n` [asset IDs](blockchain.md#asset-id),
        * `message`: a variable-length string,
        * `nonce`: unique 32-byte [string](blockchain.md#string) that makes the tracing point unique.
3. Issuance key pair `y, Y` (where `Y = y·G`).

**Output:** an [issuance proof](#issuance-proof) consisting of:

* triplet of points `(X, Z, Z’)`,
* pair of scalars `(e1,s1)`,
* pair of scalars `(e2,s2)`.

**Algorithm:**

1. [Validate issuance asset range proof](#validate-issuance-asset-range-proof) to make sure tracing and marker points are correct.
2. Calculate the blinding scalar `x`:

        x = ScalarHash("x" || AC || T || y || nonce || message)

3. Blind the tracing point being tested: `Z = x·T`.
4. Calculate commitment to the blinding key: `X = x·(J+M)`.
5. Calculate and blind a tracing point corresponding to the issuance key pair `y,Y`: `Z’ = x·y·(J+M)`.
6. Calculate a message hash: `msghash = Hash32("IP" || AC || T || X || Z || Z’)`.
7. Create a proof that `Z` blinds tracing point `T` and `X` commits to that blinding factor (i.e. the discrete log `X/(J+M)` is equal to the discrete log `Z/T`):
    1. Calculate the nonce `k1 = ScalarHash("k1" || msghash || y || x)`.
    2. Calculate point `R1 = k1·(J+M)`.
    3. Calculate point `R2 = k1·T`.
    4. Calculate scalar `e1 = ScalarHash("e1" || msghash || R1 || R2)`.
    5. Calculate scalar `s1 = k1 + x·e1 mod L`.
8. Create a proof that `Z’` is a blinded tracing point corresponding to `Y[j]` (i.e. the discrete log `Z’/X` is equal to the discrete log `Y[j]/G`):
    1. Calculate the nonce `k2 = ScalarHash("k2" || msghash || y || x)`.
    2. Calculate point `R3 = k2·X`.
    3. Calculate point `R4 = k2·G`.
    4. Calculate scalar `e2 = ScalarHash("e2" || msghash || R3 || R4)`.
    5. Calculate scalar `s2 = k2 + y·e2 mod L`.
9. Return points `(X, Z, Z’)`, signature `(e1,s1)` and signature `(e2,s2)`.


#### Validate Issuance Proof WIP

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `IARP`: the to-be-verified [confidential issuance asset range proof](#confidential-issuance-asset-range-proof) consisting of:
    * `{Y[i]}`: `n` issuance keys encoded as [points](#point) corresponding to the asset IDs,
    * `T`: tracing [point](#point),
    * `Bm`: blinded marker [point](#point),
    * `ms = (e’,s’)`: the marker signature,
    * `rs = {e[0], s[0], ... s[n-1]}`: the issuance ring signature,
    * And provided separately from the range proof:
        * `{a[i]}`: `n` [asset IDs](blockchain.md#asset-id),
        * `message`: a variable-length string,
        * `nonce`: unique 32-byte [string](blockchain.md#string) that makes the tracing point unique.
3. Index `j` of the issuance key `Y[j]` being verified to be used (or not) in the given issuance range proof.
4. [Issuance proof](#issuance-proof) consisting of:
    * triplet of points `(X, Z, Z’)`,
    * pair of scalars `(e1,s1)`,
    * pair of scalars `(e2,s2)`.

**Output:**

* If the proof is valid: `“yes”` or `“no”` indicating whether the key `Y[j]` was used or not to issue asset ID in the commitment `AC`.
* If the proof is invalid: `nil`.

**Algorithm:**

1. Calculate a message hash: `msghash = Hash32("IP" || AC || T || X || Z || Z’)`.
2. Verify that `Z` blinds tracing point `T` and `X` commits to that blinding factor (i.e. the discrete log `X/(J+M)` is equal to the discrete log `Z/T`):
    1. Calculate point `R1 = s1·(J+M) - e1·X`.
    2. Calculate point `R2 = s1·T - e1·Z`.
    3. Calculate scalar `e’ = ScalarHash("e1" || msghash || R1 || R2)`.
    4. Verify that `e’` is equal to `e1`. If validation fails, halt and return `nil`.
3. Verify that `Z’` is a blinded tracing point corresponding to `Y[j]` (i.e. the discrete log `Z’/X` is equal to the discrete log `Y[j]/G`):
    1. Calculate point `R3 = s2·X - e2·Z’`.
    2. Calculate point `R4 = s2·G - e2·Y[j]`.
    3. Calculate scalar `e” = ScalarHash("e2" || msghash || R3 || R4)`.
    4. Verify that `e”` is equal to `e2`. If validation fails, halt and return `nil`.
4. If `Z` is equal to `Z’` return `“yes”`. Otherwise, return `“no”`.




### Value Range Proof WIP

Value range proof demonstrates that a [value commitment](#value-commitment) encodes a value between 0 and 2<sup>63</sup>–1. The 63-bit limit is chosen for consistency with the numeric limits defined for the asset version 1 outputs and VM version 1 [numbers](vm1.md#vm-number).

Value range proof can be [non-confidential](#non-confidential-value-range-proof) or [confidential](#confidential-value-range-proof).

#### Non-Confidential Value Range Proof WIP

A non-confidential range proof demonstrates the non-encrypted amount and allows efficient verification that a given [value commitment](#value-commitment) commits to that amount.

Field                        | Type      | Description
-----------------------------|-----------|------------------
Type                         | byte      | Contains value 0x00 to indicate the commitment is not blinded.
Amount                       | varint63  | Amount

#### Confidential Value Range Proof WIP

A confidential range proof proves that a given [value commitment](#value-commitment) commits to an amount in a valid range (between 0 and 2<sup>63</sup>–1) without revealing the exact value.

For the most compact encoding, confidential value range proof uses base-4 digits represented by 4-key ring signatures proving the value of each pair of bits. If the number of bits is odd, the last ring signature contains only 2 elements proving the value of the highest-order bit. All ring signatures share the same e-value (see below) forming a so-called "[borromean ring signature](#borromean-ring-signature)".

Value range proof allows a space-privacy tradeoff by making a smaller number of bits confidential while exposing a "minimum value" and a decimal exponent. The complete value is broken down in the following components:

    value = vmin + (10^exp)·(d[0]·(4^0) + ... + d[m-1]·(4^(m-1)))

Where d<sub>i</sub> is the i’th digit in a m-digit mantissa (that has either 2·m–1 or 2·m bits). Exponent `exp` and the minimum value `vmin` are public and by default set to zero by the user creating the transaction.

Field                     | Type      | Description
--------------------------|-----------|------------------
Type                      | byte      | Contains value 0x01 to indicate the commitment is blinded.
Number of bits            | byte      | Integer `n` indicating number of confidential mantissa bits between 1 and 63.
Exponent                  | byte      | Integer `exp` indicating the decimal exponent from 0 to 10.
Minimum value             | varint63  | Minimum value `vmin` from 0 to 2<sup>63</sup>–1.
Digit commitments         | [pubkey]  | List of `(n+1)/2 – 1` individual digit pedersen commitments where `n` is the number of mantissa bits.
Borromean Ring Signature  | [Borromean Ring Signature](#borromean-ring-signature) | List of all 32-byte elements comprising all ring signatures proving the value of each digit.

The total number of elements in the [Borromean Ring Signature](#borromean-ring-signature) is `1 + 4·n/2` where `n` is number of bits and `n/2` is a number of rings.

#### Create Value Range Proof WIP

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `VC`: the [value commitment](#value-commitment).
3. `N`: the number of bits to be blinded.
4. `value`: the 64-bit amount being encrypted and blinded.
5. `{pt[i]}`: plaintext payload string consisting of `2·N - 1` 32-byte elements.
6. `f`: the [value blinding factor](#value-blinding-factor).
7. `rek`: the [record encryption key](#record-encryption-key).
8. `message`: a variable-length string.

Note: this version of the signing algorithm does not use decimal exponent or minimum value and sets them both to zero.

**Output:** the [value range proof](#value-range-proof) consisting of:

* `N`: number of blinded bits (equals to `2·n`),
* `exp`: exponent (zero),
* `vmin`: minimum value (zero),
* `{D[t]}`: `n-1` digit commitments encoded as [points](#point) (excluding the last digit commitment),
* `{e,s[t,j]}`: `1 + 4·n` 32-byte elements representing a [borromean ring signature](#borromean-ring-signature),

In case of failure, returns `nil` instead of the range proof.

**Algorithm:**

1. Check that `N` belongs to the set `{8,16,32,48,64}`; if not, halt and return nil.
2. Check that `value` is less than `2^N`; if not, halt and return nil.
3. Define `vmin = 0`.
4. Define `exp = 0`.
5. Define `base = 4`.
6. Calculate the message to sign: `msghash = Hash256("VRP" || AC || VC || uint64le(N) || uint64le(exp) || uint64le(vmin) || message)` where `N`, `exp`, `vmin` are encoded as 64-bit little-endian integers.
7. Calculate payload encryption key unique to this payload and the value: `pek = Hash256("pek" || msghash || rek || f)`.
8. Let number of digits `n = N/2`.
9. [Encrypt the payload](#encrypt-payload) using `pek` as a key and `2·N-1` 32-byte plaintext elements to get `2·N` 32-byte ciphertext elements: `{ct[i]} = EncryptPayload({pt[i]}, pek)`.
10. For `t` from `0` to `n-1` (each digit):
    1. Calculate generator `G’[t]`:
        1. If `t` is less than `n-1`: set `G’[t] = G[t]`, where `G[t]` is a [tertiary generator](#generators) at index `t`.
        2. If `t` equals `n-1`: set `G’[t] = G - ∑G[i]` for all `i` from `0` to `n-2`.
    2. Calculate `digit[t] = value & (0x03 << 2·t)` where `<<` denotes a bitwise left shift.
    3. Calculate `D[t] = digit[t]·H + f·G’[t]`.
    4. Calculate `j[t] = digit[t] >> 2·t` where `>>` denotes a bitwise right shift.
11. For `t` from `0` to `n-1` (each digit):
    1. Calculate base points: `Bs[t] = (G’[t], J)`.
    2. For `i` from `0` to `base-1` (each digit’s value):
        1. Calculate point `P[t,i] = D[t] - i·(base^t)·H`.
        2. Calculate point `Q[t,i] = F[t] - i·(base^t)·C`.
12. [Create Borromean Ring Signature](#create-borromean-ring-signature) `brs` with the following inputs:
    * `msghash` as the message to sign.
    * `n`: number of rings.
    * `m = base`: number of signatures per ring.
    * `M = 2`
    * `{(G’[t], J)}`: `2·n` base points where `J` is reused for each `G’[t]`.
    * `{(P[i,j], Q[i,j])}`: `2·n·m` [points](#point).
    * `{f}`: the blinding factor `f` repeated `n` times.
    * `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[t,j[t]] == f·G’[t]`.
    * `{r[i]} = {ct[i]}`: random string consisting of `n·m` 32-byte ciphertext elements.
13. If failed to create borromean ring signature `brs`, return nil. The chance of this happening is below 1 in 2<sup>124</sup>. In case of failure, retry [creating blinded value commitment](#create-blinded-value-commitment) with incremented counter. This would yield a new blinding factor `f` that will produce different digit blinding keys in this algorithm.
14. Return the [value range proof](#value-range-proof):
    * `N`:  number of blinded bits (equals to `2·n`),
    * `exp`: exponent (zero),
    * `vmin`: minimum value (zero),
    * `{D[t]}`: `n-1` digit commitments encoded as [public keys](#point) (excluding the last digit commitment),
    * `{e,s[t,j]}`: `1 + n·4` 32-byte elements representing a [borromean ring signature](#borromean-ring-signature),



#### Validate Value Range Proof WIP

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `VC`: the [value commitment](#value-commitment).
3. `VRP`: the [value range proof](#value-range-proof) consisting of:
    * `N`: the number of bits in blinded mantissa (8-bit integer, `N = 2·n`).
    * `exp`: the decimal exponent (8-bit integer).
    * `vmin`: the minimum amount (64-bit integer).
    * `{D[t]}`: the list of `n-1` digit commitments encoded as [points](#point) (excluding the last digit commitment).
    * `{e0, s[i,j]...}`: the [borromean ring signature](#borromean-ring-signature) encoded as a sequence of `1 + 4·n` 32-byte integers.
4. `message`: a variable-length string.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. Perform limit checks one by one. If any one fails, halt and return `false`:
    1. Check that `exp` is less or equal to 10.
    2. Check that `vmin` is less than 2<sup>63</sup>.
    3. Check that `N` is divisible by 2.
    4. Check that `N` is equal or less than 64.
    5. Check that `N + exp·4` is less or equal to 64.
    6. Check that `(10^exp)·(2^N - 1)` is less than 2<sup>63</sup>.
    7. Check that `vmin + (10^exp)·(2^N - 1)` is less than 2<sup>63</sup>.
2. Let `n = N/2`.
3. Let `base = 4`.
4. Calculate the message to validate: `msghash = Hash256("VRP" || AC || VC || uint64le(N) || uint64le(exp) || uint64le(vmin) || message)` where `N`, `exp`, `vmin` are encoded as 64-bit little-endian integers.
5. Calculate last digit commitment `D[n-1] = (10^(-exp))·(VC.V - vmin·AC.H) - ∑(D[t])`, where `∑(D[t])` is a sum of all but the last digit commitment specified in the input to this algorithm.
6. For `t` from `0` to `n-1` (each digit):
    1. Calculate generator `G’[t]`:
        1. If `t` is less than `n-1`: set `G’[t] = G[t]`, where `G[t]` is a [tertiary generator](#generators) at index `t`.
        2. If `t` equals `n-1`: set `G’[t] = G - ∑G[i]` for all `i` from `0` to `n-2`.
    2. For `i` from `0` to `base-1` (each digit’s value):
        1. Calculate point `P[t,i] = D[t] - i·(base^t)·H`. For efficiency perform recursive point addition of `-(base^t)·H` instead of scalar multiplication.
        2. Calculate point `Q[t,i] = F[t] - i·(base^t)·C`. For efficiency perform iterative point addition of `-(base^t)·C` instead of scalar multiplication.
7. [Validate Borromean Ring Signature](#validate-borromean-ring-signature) with the following inputs:
    * `msghash`: the 32-byte string being verified.
    * `n`: number of rings.
    * `m=base`: number of signatures in each ring.
    * `M = 2`
    * `{(G’[t], J)}`: `2·n` base points.
    * `{(P[i,j], Q[i,j])}`: `2·n·m` public keys, [points](#point) on the elliptic curve.
    * `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.
8. Return `true` if verification succeeded, or `false` otherwise.



#### Recover Payload From Value Range Proof WIP

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `VC`: the [value commitment](#value-commitment).
3. `VRP`: the [value range proof](#value-range-proof) consisting of:
    * `N`: the number of bits in blinded mantissa (8-bit integer, `N = 2·n`).
    * `exp`: the decimal exponent (8-bit integer).
    * `vmin`: the minimum amount (64-bit integer).
    * `{D[t]}`: the list of `n-1` digit commitments encoded as [points](#point) (excluding the last digit commitment).
    * `{e0, s[i,j]...}`: the [borromean ring signature](#borromean-ring-signature) encoded as a sequence of `1 + 4·n` 32-byte integers.
4. `value`: the 64-bit amount being encrypted and blinded.
5. `f`: the [value blinding factor](#value-blinding-factor).
6. `rek`: the [record encryption key](#record-encryption-key).
7. `message`: a variable-length string.

**Output:** `{pt[i]}`: an array of 32-bytes of plaintext data if recovery succeeded, `nil` otherwise.

**Algorithm:**

1. Perform limit checks one by one. If any one fails, halt and return `false`:
    1. Check that `exp` is less or equal to 10.
    2. Check that `vmin` is less than 2<sup>63</sup>.
    3. Check that `N` is divisible by 2.
    4. Check that `N` is equal or less than 64.
    5. Check that `N + exp·4` is less or equal to 64.
    6. Check that `(10^exp)·(2^N - 1)` is less than 2<sup>63</sup>.
    7. Check that `vmin + (10^exp)·(2^N - 1)` is less than 2<sup>63</sup>.
2. Let `n = N/2`.
3. Let `base = 4`.
4. Calculate the message to validate: `msghash = Hash256("VRP" || AC || VC || uint64le(N) || uint64le(exp) || uint64le(vmin) || message)` where `N`, `exp`, `vmin` are encoded as 64-bit little-endian integers.
5. Calculate last digit commitment `D[n-1] = (10^(-exp))·(VC.V - vmin·AC.H) - ∑(D[t])`, where `∑(D[t])` is a sum of all but the last digit commitment specified in the input to this algorithm.
6. For `t` from `0` to `n-1` (each digit):
    1. Calculate generator `G’[t]`:
        1. If `t` is less than `n-1`: set `G’[t] = G[t]`, where `G[t]` is a [tertiary generator](#generators) at index `t`.
        2. If `t` equals `n-1`: set `G’[t] = G - ∑G[i]` for all `i` from `0` to `n-2`.
    2. Calculate `digit[t] = value & (0x03 << 2·t)` where `<<` denotes a bitwise left shift.
    3. Calculate `j[t] = digit[t] >> 2·t` where `>>` denotes a bitwise right shift.
    4. For `i` from `0` to `base-1` (each digit’s value):
        1. Calculate point `P[t,i] = D[t] - i·(base^t)·H`. For efficiency perform recursive point addition of `-(base^t)·H` instead of scalar multiplication.
        2. Calculate point `Q[t,i] = F[t] - i·(base^t)·C`. For efficiency perform iterative point addition of `-(base^t)·C` instead of scalar multiplication.
7. [Recover Payload From Borromean Ring Signature](#recover-payload-from-borromean-ring-signature): compute an array of `2·N` 32-byte chunks `{ct[i]}` using the following inputs (halt and return `nil` if decryption fails):
    * `msghash`: the 32-byte string to be signed.
    * `n=N/2`: number of rings.
    * `m=base`: number of signatures in each ring.
    * `M = 2`
    * `{(G’[t], J)}`: `2·n` base points.
    * `{(P[i,j], Q[i,j])}`: `2·n·m` public keys, [points](#point) on the elliptic curve.
    * `{f}`: the blinding factor `f` repeated `n` times.
    * `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[t,j[t]] == f·G’[t]`.
    * `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.
8. Derive payload encryption key unique to this payload and the value: `pek = Hash256("VRP.pek" || rek || f || VC)`.
9. [Decrypt payload](#decrypt-payload): compute an array of `2·N-1` 32-byte chunks: `{pt[i]} = DecryptPayload({ct[i]}, pek)`. If decryption fails, halt and return `nil`.
10. Return `{pt[i]}`, a plaintext array of `2·N-1` 32-byte elements.







### Value Proof

Value proof demonstrates that a given [value commitment](#value-commitment) encodes a specific amount and asset ID. It is used to privately prove the contents of an output without revealing blinding factors to a counter-party or an HSM.

#### Create Value Proof

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment) to `assetid`.
2. `VC`: the [value commitment](#value-commitment) to `value`.
3. `assetid`: the [asset ID](blockchain.md#asset-id) to be proven used in `AC`.
4. `value`: the amount to be proven.
5. `c`: the [asset ID blinding factor](#asset-id-blinding-factor) used in `AC`.
6. `f`: the [value blinding factor](#value-blinding-factor).
7. `message`: a variable-length string.

**Output:**

1. `(QG,QJ),e,s`: the [excess commitment](#excess-commitment) with its signature.

**Algorithm:**

1. Compute [scalar](#scalar) `q = value*c + f`.
2. [Create excess commitment](#create-excess-commitment) `(QG,QJ),e,s` using `q` and `message`.



#### Validate Value Proof

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment) to `assetid`.
2. `VC`: the [value commitment](#value-commitment) to `value`.
3. `assetid`: the [asset ID](blockchain.md#asset-id) to be proven used in `AC`.
4. `value`: the amount to be proven.
5. `(QG,QJ),e,s`: the [excess commitment](#excess-commitment) with its signature that proves that `assetid` and `value` are committed to `AC` and `VC`.
6. `message`: a variable-length string.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. [Validate excess commitment](#validate-excess-commitment) `(QG,QJ),e,s,message`.
2. Compute [asset ID point](#asset-id-point): `A’ = 8·Hash256(assetID || counter)`.
4. [Create nonblinded value commitment](#create-nonblinded-value-commitment): `V’ = value·A’`.
5. Verify that [point pair](#point-pair) `(QG + V’, QJ)` equals `VC`.



### Validate Issuance

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `VC`: the [value commitment](#value-commitment).
3. `IARP`: the [issuance asset ID range proof](#issuance-asset-range-proof) together with:
    * `{a[i]}`: [asset identifiers](blockchain.md#asset-id), one for each issuance key in the range proof.
    * `message`: a variable-length string.
    * `nonce`: a unique 32-byte string.
4. `VRP`: the [value range proof](#value-range-proof).

**Output:** `true` if verification succeeded, `false` otherwise.

**Algorithm:**

1. [Validate issuance asset range proof](#validate-issuance-asset-range-proof) using `(AC,IARP,{a[i]},message,nonce)`.
2. [Validate value range proof](#validate-value-range-proof) using `AC`, `VC` and `VRP`.
3. Return `true`.



### Validate Destination

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `VC`: the [value commitment](#value-commitment).
3. `ARP`: the [asset range proof](#asset-range-proof).
4. `VRP`: the [value range proof](#value-range-proof).

**Output:** `true` if verification succeeded, `false` otherwise.

**Algorithm:**

1. [Validate asset range proof](#validate-asset-range-proof) using `AC` and `ARP`.
2. [Validate value range proof](#validate-value-range-proof) using `AC`, `VC` and `VRP`.
3. Return `true`.



### Validate Assets Flow

**Inputs:**

1. List of sources (validated spends and issuances), each source consisting of:
    * `AC`: the [asset ID commitment](#asset-id-commitment).
    * `VC`: the [value commitment](#value-commitment).
2. List of destinations, each destination consisting of:
    * `AD`: the [asset ID commitment](#asset-id-commitment).
    * `VD`: the [value commitment](#value-commitment).
    * `ARP`: the [asset range proof](#asset-range-proof) or an empty string.
    * `VRP`: the [value range proof](#value-range-proof).
3. List of [excess commitments](#excess-commitment): `{(QC[i], s[i], e[i], message[i])}`.

**Output:** `true` if verification succeeded, `false` otherwise.

**Algorithm:**

1. For each destination:
    1. If `ARP` is empty has zero keys, verify that the destination `AC` equals one of the asset ID commitments among the sources.
    2. If `ARP` is confidential, verify that each asset ID commitment candidate belongs to the set of the source asset ID commitments.
    3. [Validate destination](#validate-destination).
2. [Validate value commitments balance](#validate-value-commitments-balance) using source, destination and excess commitments.
3. Return `true`.









## Encryption


### Encrypted Payload

#### Encrypt Payload

**Inputs:**

1. `n`: number of 32-byte elements in the plaintext to be encrypted.
2. `{pt[i]}`: list of `n` 32-byte elements of plaintext data.
3. `ek`: the encryption/authentication key unique to this payload.

**Output:** the `{ct[i]}`: list of `n` 32-byte ciphertext elements, where the last one is a 32-byte MAC.

**Algorithm:**

1. Calculate a keystream, a sequence of 32-byte random values: `{keystream[i]} = StreamHash("EP" || ek, 32·n)`.
2. Encrypt the plaintext payload: `{ct[i]} = {pt[i] XOR keystream[i]}`.
3. Calculate MAC: `mac = Hash256(ek || ct[0] || ... || ct[n-1])`.
4. Return a sequence of `n+1` 32-byte elements: `{ct[0], ..., ct[n-1], mac}`.

#### Decrypt Payload

**Inputs:**

1. `n`: number of 32-byte elements in the ciphertext to be decrypted.
2. `{ct[i]}`: list of `n+1` 32-byte ciphertext elements, where the last one is MAC (32 bytes).
3. `ek`: the encryption/authentication key.

**Output:** the `{pt[i]}` or `nil`, if authentication failed.

**Algorithm:**

1. Calculate MAC’: `mac’ = Hash256(ek || ct[0] || ... || ct[n-1])`.
2. Extract the transmitted MAC: `mac = ct[n]`.
3. Compare calculated  `mac’` with the received `mac`. If they are not equal, return `nil`.
4. Calculate a keystream, a sequence of 32-byte random values: `{keystream[i]} = StreamHash("EP" || ek, 32·n)`.
5. Decrypt the plaintext payload: `{pt[i]} = {ct[i] XOR keystream[i]}`.
5. Return `{pt[i]}`.


### Encrypted Value

Encrypted value is a 40-byte string representing a simple encryption of the numeric amount and its blinding factor as used in a [value commitment](#value-commitment). The encrypted value is authenticated by the corresponding _value commitment_.

#### Encrypt Value

**Inputs:**

1. `VC`: the [value commitment](#value-commitment).
2. `value`: the 64-bit amount being encrypted and blinded.
3. `f`: the [value blinding factor](#value-blinding-factor).
4. `vek`: the [value encryption key](#value-encryption-key).

**Output:** `(ev||ef)`, the [encrypted value](#encrypted-value) including its blinding factor.

**Algorithm:**

1. Expand the encryption key: `ek = StreamHash("EV" || vek || VC, 40)`.
2. Encrypt the value using the first 8 bytes: `ev = value XOR ek[0,8]`.
3. Encrypt the value blinding factor using the last 32 bytes: `ef = f XOR ek[8,32]` where `f` is encoded as 256-bit little-endian integer.
4. Return `(ev||ef)`.

#### Decrypt Value

**Inputs:**

1. `VC`: the full [value commitment](#value-commitment).
2. `AC`: the target [asset ID commitment](#asset-id-commitment) that obfuscates the asset ID.
3. `(ev||ef)`: the [encrypted value](#encrypted-value).
4. `vek`: the [value encryption key](#value-encryption-key).

Value and asset ID commitments must be [proven to be valid](#validate-assets-flow).

**Output:** `(value, f)`: decrypted and verified amount and the [value blinding factor](#value-blinding-factor); or `nil` if verification did not succeed.

**Algorithm:**

1. Expand the encryption key: `ek = StreamHash("EV" || vek || VC, 40)`.
2. Decrypt the value using the first 8 bytes: `value = ev XOR ek[0,8]`.
3. Decrypt the value blinding factor using the last 32 bytes: `f = ef XOR ek[8,32]` where `f` is encoded as 256-bit little-endian integer.
4. [Create blinded value commitment](#create-blinded-value-commitment) `VC’` using `AC`, `value` and the raw blinding factor `f` (instead of `vek`).
5. Verify that `VC’` equals `VC`. If not, halt and return `nil`.
6. Return `(value, f)`.


### Encrypted Asset ID

Encrypted value is a 64-byte string representing a simple encryption of the [asset ID](blockchain.md#asset-id) and its blinding factor as used in a [asset ID commitment](#asset-id-commitment). The encrypted asset ID is authenticated by the corresponding _asset ID commitment_.

#### Encrypt Asset ID

**Inputs:**

1. `assetID`: the [asset ID](blockchain.md#asset-id).
2. `AC`: the [asset ID commitment](#asset-id-commitment) hiding the `assetID`.
3. `c`: the [asset ID blinding factor](#asset-id-blinding-factor) for the commitment `AC` such that `AC == (8·Hash256(assetID...) + c·G, c·J)`.
4. `aek`: the [asset ID encryption key](#asset-id-encryption-key).

**Output:** `(ea||ec)`, the [encrypted asset ID](#encrypted-asset-id) including the encrypted blinding factor for `AC`.

**Algorithm:**

1. Expand the encryption key: `ek = StreamHash("EA" || aek || AC, 40)`.
2. Encrypt the asset ID using the first 32 bytes: `ea = assetID XOR ek[0,32]`.
3. Encrypt the blinding factor using the second 32 bytes: `ec = c XOR ek[32,32]` where `c` is encoded as a 256-bit little-endian integer.
4. Return `(ea||ec)`.


#### Decrypt Asset ID

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment) that obfuscates the asset ID.
2. `(ea||ec)`: the [encrypted asset ID](#encrypted-asset-id) including the encrypted blinding factor for `H`.
3. `aek`: the [asset ID encryption key](#asset-id-encryption-key).

Asset ID commitment must be [proven to be valid](#validate-assets-flow).

**Outputs:** `(assetID,c)`: decrypted and verified [asset ID](blockchain.md#asset-id) with its blinding factor, or `nil` if verification failed.

**Algorithm:**

1. Expand the decryption key: `ek = StreamHash("EA" || aek || AC, 40)`.
2. Decrypt the asset ID using the first 32 bytes: `assetID = ea XOR ek[0,32]`.
3. Decrypt the blinding factor using the second 32 bytes: `c = ec XOR ek[32,32]`.
4. [Create blinded asset ID commitment](#create-blinded-asset-id-commitment) `AC’` using `assetID` and the raw blinding factor `c` (instead of `aek`).
5. Verify that `AC’` equals `AC`. If not, halt and return `nil`.
6. Return `(assetID, c)`.






### Encrypt Issuance

**Inputs:**

1. `rek`: the [record encryption key](#record-encryption-key) unique to this issuance.
2. `assetID`: the output asset ID.
3. `value`: the output amount.
4. `N`: number of bits to encrypt (`value` must fit within `N` bits).
5. `{(assetIDs[i], Y[i])}`: `n` input asset IDs and corresponding issuance public keys.
6. `y`: issuance key for `assetID` such that `Y[j] = y·G` where `j` is the index of the issued asset: `assetIDs[j] == assetID`.
7. `message`: a variable-length string to be signed.
8. `nonce`: a unique 32-byte string.

**Outputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `VC`: the [value commitment](#value-commitment).
3. `IARP`: the [issuance asset ID range proof](#issuance-asset-range-proof).
4. `VRP`: the [value range proof](#value-range-proof).
5. `c`: the [asset ID blinding factor](#asset-id-blinding-factor) for the asset ID commitment `AC`.
6. `f`: the [value blinding factor](#value-blinding-factor) for the value commitment `VC`.

In case of failure, returns `nil` instead of the items listed above.

**Algorithm:**

1. [Derive asset ID encryption key](#asset-id-encryption-key) `aek` from `rek`.
2. [Derive value encryption key](#value-encryption-key) `vek` from `rek`.
3. Find `j` index of the `assetID` among `{assetIDs[i]}`. If not found, halt and return `nil`.
5. [Create blinded asset ID commitment](#create-blinded-asset-id-commitment): compute `(AC,c)` from `(assetid, aek)`.
6. [Create blinded value commitment](#create-blinded-value-commitment): compute `(VC,f)` from `(value, vek, AC)`.
7. [Create issuance asset range proof](#create-issuance-asset-range-proof): compute `IARP` from `(AC, c, {assetIDs[i]}, {Y[i]}, message, nonce, j, y)`.
8. [Create Value Range Proof](#create-value-range-proof): compute `VRP` from `(AC, VC, N, value, f, rek)` and all-zeroes payload.
9. Return `(AC, VC, IARP, VRP, c, f)`.




### Encrypt Output

This algorithm encrypts the amount and asset ID of a given output and creates [value range proof](#value-range-proof) with encrypted payload.
If the excess factor is provided, it is used to compute a matching blinding factor to cancel it out.

This algorithm _does not_ create [asset range proof](#asset-range-proof) (ARP), since it requires knowledge about the input descriptors and one of their blinding factors.

ARP can be added separately using [Create Asset Range Proof](#create-asset-range-proof).

**Inputs:**

1. `rek`: the [record encryption key](#record-encryption-key).
2. `assetID`: the output asset ID.
3. `value`: the output amount.
4. `N`: number of bits to encrypt (`value` must fit within `N` bits).
5. `plaintext`: binary string that has length of less than `32·(2·N-1)` bytes when encoded as [varstring31](blockchain.md#string).
6. Optional `q`: the [excess factor](#excess-factor) to have this output balance with the transaction. If omitted, blinding factor is generated at random.

**Outputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `VC`: the [value commitment](#value-commitment).
3. `VRP`: the [value range proof](#value-range-proof).
4. `c’`: the output [asset ID blinding factor](#asset-id-blinding-factor) for the asset ID commitment `H’`.
5. `f’`: the output [value blinding factor](#value-blinding-factor).

In case of failure, returns `nil` instead of the items listed above.

**Algorithm:**

1. Encode `plaintext` using [varstring31](blockchain.md#string) encoding and split the string in 32-byte chunks `{pt[i]}` (last chunk padded with zero bytes if needed).
2. If the number of chunks `{pt[i]}` exceeds `2·N-1`, halt and return `nil`.
3. If the number of chunks `{pt[i]}` is less than `2·N-1`, pad the array with all-zero 32-byte chunks.
4. If `value ≥ 2^N`, halt and return `nil`.
5. [Derive asset ID encryption key](#asset-id-encryption-key) `aek` from `rek`.
6. [Derive value encryption key](#value-encryption-key) `vek` from `rek`.
7. [Create blinded asset ID commitment](#create-blinded-asset-id-commitment): compute `(AC,c)` from `(assetID, aek)`.
9. [Create blinded value commitment](#create-blinded-value-commitment): compute `(V,f)` from `(vek, value, AC.H)`.
10. If `q` is provided:
    1. Compute `extra` scalar: `extra = q - f - value·c`.
    2. Add `extra` to the value blinding factor: `f = f + extra`.
    3. Adjust the value commitment too: `VC = VC + extra·(G,J)`.
    4. Note: as a result, the total blinding factor of the output will be equal to `q`.
11. [Create Value Range Proof](#create-value-range-proof): compute `VRP` from `(AC, VC, N, value, {pt[i]}, f, rek)`.
12. Return `(AC, VC, VRP, c, f)`.



### Decrypt Output

This algorithm decrypts fully encrypted amount and asset ID for a given output.

**Inputs:**

1. `rek`: the [record encryption key](#record-encryption-key).
2. `AC`: the [asset ID commitment](#asset-id-commitment).
3. `VC`: the [value commitment](#value-commitment).
4. `ARP`: the [asset range proof](#asset-range-proof).
5. `VRP`: the [value range proof](#value-range-proof).
6. `(ev||ef)`: the [encrypted value](#encrypted-value).
7. `(ea||ec)`: the [encrypted asset ID](#encrypted-asset-id).

**Outputs:**

1. `assetID`: the output asset ID.
2. `value`: the output amount.
3. `c`: the output [asset ID blinding factor](#asset-id-blinding-factor) for the asset ID commitment `H`.
4. `f`: the output [value blinding factor](#value-blinding-factor).
5. `plaintext`: the binary string that has length of less than `32·(2·N-1)` bytes when encoded as [varstring31](blockchain.md#string).

In case of failure, returns `nil` instead of the items listed above.

**Algorithm:**

1. [Derive asset encryption key](#asset-id-encryption-key) `aek` from `rek`.
2. [Derive value encryption key](#value-encryption-key) `vek` from `rek`.
3. Decrypt asset ID:
    1. If `ARP` is [non-confidential](#non-confidential-asset-range-proof): set `assetID` to the one stored in `ARP`, set `c` to zero.
    2. If `ARP` is [confidential](#confidential-asset-range-proof), [decrypt asset ID](#decrypt-asset-id): compute `(assetID,c)` from `((ea||ec),AC,aek)`. If verification failed, halt and return `nil`.
4. Decrypt value and recover payload:
    1. If `VRP` is [non-confidential](#non-confidential-value-range-proof):
        1. Set `value` to the one stored in `VRP`, set `f` to zero.
        2. Set `plaintext` to an empty string.
    2. If `VRP` is [confidential](#confidential-value-range-proof):
        1. [Decrypt value](#decrypt-value): compute `(value, f)` from `((ev||ef),AC,VC,vek)`. If verification failed, halt and return `nil`.
        2. [Recover payload from the range proof](#recover-payload-from-value-range-proof): compute a list of 32-byte chunks `{pt[i]}` from `(AC,VC,VRP,value,f,rek)`. If verification failed, halt and return `nil`.
        3. Flatten the array `{pt[i]}` in a binary string and decode it using [varstring31](blockchain.md#string) encoding. If decoding fails, halt and return `nil`.
5. Return `(assetID, value, c, f, plaintext)`.










## Security

### Asset Point denial of service risk

Since the amount of computations it takes to compute an [Asset ID point](#asset-id-point) is variable,
malicious prover may find an `assetID` by brute force that would require the verifier
to perform arbitrary amount of computations.

However, mounting an attack to force a non-negligible amount of work on a verifier is 
computationally infeasible. For `N` hash computations to be performed by a verifier,
a malicious prover would have to perform an order of `2^N` hash computations.

The alternative solution is to have creators of asset identifiers to choose an issuance
program (that defines the asset ID) so that their asset ID always hashes to a valid point.
Unfortunately, this approach rejects roughly half of existing asset IDs created on
the blockchains deployed before the extension to _Confidential Assets_. 
We consider that to preserve asset ID compatibility at the risk of an infeasible 
denial of service attack is an acceptable tradeoff.




TBD: security theorems and proofs



### Asset Commitment (AC)

    A = PointHash(assetid)

Non-blinded asset commitment:

    (A, O)

Blinded asset commitment:

    (A + f*G, f*J)

### Issuance Asset Range Proof (IARP) - WIP

    M = rand    - unique marker point

    x - blinding factor
    y - blinding factor

    H = A + x*G - blinded commitment
    B = x*J     - blinding commitment
    T = y*M     - tracing point
    Y = y*G     - issuance key

Verifier:

    Need to prove knowledge of `y` and `x`.

    1. Receive e,sx,sy,M,H,B,Bm,T,Y
    2. Compute R1 = sx*G - e*(H-A)
    3. Compute R2 = sx*J - e*B
    4. Compute R3 = sy*M - e*T
    5. Compute R4 = sy*G - e*Y
    6. Compute e' = Hash(R1||R2||R3||R4)
    7. Verify e' == e

Signer:

    1. Choose kx = random
    2. Choose ky = random
    3. Compute R1 = kx*G
    4. Compute R2 = kx*J
    5. Compute R3 = ky*M
    6. Compute R4 = ky*G
    7. Compute e = Hash(R1||R2||R3||R4)
    8. Compute sx = kx + e*x
    9. Compute sy = ky + e*y
    10. Return (e,sx,sy)

Proof of soundness:

    let R_i = r_i*G
    let J = j*G
    let M = m*G
    let T = t*G
    etc.

    Need to prove:

        1. b == (h - a)*j
        2. t == y*m
    
    1. Factor out `sx` from R1, R2, R3, R4 definitions:

        sx = r1 + e*(h - a)
        j*sx = r2 + e*b

    2. Factor out `sy` from R1, R2, R3, R4 definition:

        sy = r4 + e*y
        m*sy = r3 + e*t

    3. Equality must hold for any value of `e` (since it's determined after all vars except sy/sx).
       Therefore (placing definition of sx and sy from the first equation to the second equation):

        m*y == t
        j*(h-a) == b

        QED.

Proof of issuance:

    The proof above proves simultaneously the knowledge of `y` and commitment to a blinding factor for asset `A`.
    Provided Y is associated with A, this makes sure that the holder of `y` is blinding the `A`.

Proof of tracing:

    The binding property guarantees that T is `y*M`, meaning a multiplication of a public marker point
    is done by the issuance private key (same key as in `Y = y*G`).

Proof of safety of issuance:

    Proof of issuance cannot be replayed as all proofs of knowledge are tied to a given transaction.

Proof of safety of tracing:

    Tracing point cannot be replayed since proofs are tied to a given transaction.
    Two tracing points from different transactions cannot be linked since points M are unique and tied to a transaction.


Proof of blinding:

    1. To link tracing point T to an issuance key Y, one needs to break either ECDLP or DDH:

        ECDLP: extract all the dlogs and check if `t == m*y`
        DDH:   verify e(Y,M) == e(G,T) (using pairing e(xG,yG) -> e(xyG,G))

    2. To link tracing point to other tracing points `T' = y*M'`, one must break ECDLP to DDH
       since the points M are unique for each transaction:

        ECDLP: extract all the dlogs and check if `t == m*y && t' == m'*y`
        DDH:   verify e(Y,M) == e(G,T) && e(Y,M') == e(G,T')

    3. To link A to (B,H) one also needs to break either ECDLP or DDH.

        ECDLP: extract all dlogs, and check if j*(h-a) == b
        DDH:   verify e(H-A,J) == e(G,B)

Ring version:

    1. Iterate (A,Y) pairs.
    2. Compute a chain of e0 -> e1 -> ... e0'
    3. Verify e0' == e0

    At least one `{R_i}` tuple in the ring will have to be defined before its 
    factor `e` is determined, allowing application of the proof of soundness.

    Since the ring is perfectly symmetrical, the proof of blinding is reduced 
    to the set of the elements in the ring, without revealing to which element
    the commitment is bound to.



### Asset Range Proof (ARP)

Verifier:

    Needs to verify:

       H2 = H1 + x*G
       B2 = B1 + x*J

    1. Receive e, s, H1, H2, B1, B2
    2. Compute R1 = s*G - e*(H2 - H1)
    3. Compute R2 = s*J - e*(B2 - B1)
    4. Compute e' = Hash(R1||R2)
    5. Verify e' == e

Signer:

    1. Choose k = random
    2. Compute R1 = k*G
    3. Compute R2 = k*J
    4. Compute e = Hash(R1||R2)
    5. Compute s = k + e*x
    6. Return (e,s)

Proof of soundness:

    let R1 = r1*G
    let R2 = r2*G
    let J  = j*G
    let H1 = h1*G
    let H2 = h2*G
    let B1 = b1*G
    let B2 = b2*G

    Need to prove that (x can be any value):

       b2 == b1 + x*j
       h2 == h1 + x

       In other words: 
       b2 - b1 == (h2 - h2)*j

    1. Factor out `s` from R1 and R2 definitions:
     
        s = r1 + e*(h2 - h1)
        j*s = r2 + e*(b2 - b1)
    
    2. Equality must hold for any value of `e` (since it's determined after r1,r2,h1,h2,b1,b2).
       Therefore:

        j*(h2-h1) == b2 - b1

       which is what we are looking for

Proof of blinding:

    Given H1,B1,H2,B2 determine if H2 is a blinded H1 with the same factor as B2 in respect to B1.
    In this proof we assume absence of the signature that proves the binding.

    Let X = x*G = H2 - H1
    Let Y = y*J = B2 - B1

    Testing whether x == y requires either breaking ECDLP or DDH:

    ECDLP: 
    
        If j is known (such that J = j*G), then (j^-1)*Y = y*G and can be compared with X.

    DDH:
    
        If we can decide whether Y is DH of (X,J), then we can prove link between H1 and H2:

        DH(x*G, j*G) =?= x*j*G

        E.g. with pairings:

        e(X,J) =?= e(G,Y)

    Neither ECDLP nor DDH are tractable for Ed25519.

Ring version:

    1. Iterate (H1,B1) pairs.
    2. Compute a chain of e0 -> e1 -> ... e0'
    3. Verify e0' == e0

    At least one `{R_i}` tuple in the ring will have to be defined before its 
    factor `e` is determined, allowing application of the proof of soundness.

    Since the ring is perfectly symmetrical, the proof of blinding is reduced 
    to the set of the elements in the ring, without revealing to which element
    the commitment is bound to.


### Value Commitment (VC)


Non-blinded value commitment:

    (v*H, B)

Blinded value commitment:

    (v*H + f*G, v*B + f*J)


### Value Range Proof

Given:

    Asset commitment:
        H = a*G + c*G
        B = c*J

    Value commitment:
        V = v*H + f*G
        D = v*B + f*J

Need to verify:

    (V,D) commit to value `v` using asset commitment (H,B)

Verifier:

    Needs to verify (f could be anything, v chosen by verifier):

       V = v*H + f*G
       D = v*B + f*J

    1. Receive e, s, v, H, B, V, D
    2. Compute R1 = s*G - e*(V - v*H)
    3. Compute R2 = s*J - e*(D - v*B)
    4. Compute e' = Hash(R1||R2)
    5. Verify e' == e

Signer:

    1. Choose k = random
    2. Compute R1 = k*G
    3. Compute R2 = k*J
    4. Compute e = Hash(R1||R2)
    5. Compute s = k + e*f
    6. Return (e,s)

Proof of balance:

    Provided:
        Sum(V_i) == 0
        Sum(D_i) == 0
    
    Need:
        Sum(v_j)*A == 0 for each A

    1. Sum(D_i) == Sum[(v_i*c_i + f_i)*j*G] == 0
    2. Sum(V_i) == Sum[(v_i*a_i + v_i*c_i + f_i)*G] == 0
    3. Therefore: Sum[v_i*a_i] == 0
    4. What would be the probability of having v1*a1 == v2*a2 provided a1 != a2, and v1,v2 in 62-bit range?
    5. We assume a1 is given, v1,v2 could be tweaked to match a2. 
    6. The space of v1/v2 is 2^124 combinations.
    7. Both a1 and a2 are proven to be carried over since issuance where a1,a2 are 
       pseudo-randomly generated from Hash(assetid) with space 2^252.
    8. Therefore, chance that a2/a1 falls into one of possible 2^124 states for v1/v2 is:
    9. P = 2^124 / 2^252 = 2^128.
    10. Meaning, attacker is expected to perform 2^128 hashing operations to find a2 that allows morphing to/from a1.
    11. 







## Test vectors

TBD: Hash256, StreamHash, ScalarHash.

TBD: RS, BRS.

TBD: AC, VC, excess commitments.

TBD: ARP, IARP, VRP.


## Glossary

Term             | Description
-----------------|---------------------------------------------------------------------------------------
A                | [Asset point](#asset-id-point)
a                | Unknown discrete log of [Asset point](#asset-id-point) in respect to [generator G](#generators)
B                | TBD
b                | TBD
C                | Blinding factor half of the [asset ID commitment](#asset-id-commitment) (c·J)
c                | [Asset ID blinding factor](#asset-id-blinding-factor)
D                | digit commitment v·H+f·J
E                | TBD
e                | fiat-shamir challenge in Schnorr protocol
F                | TBD
f                | value blinding factor
G                | primary generator
H                | TBD
h                | secondary fiat-shamir challenge
I                | TBD
i                | index
J                | secondary generator
K                | TBD
L                | [Order](#elliptic-curve) of the generator on curve Ed25519.
l                | 
M                | TBD
m                | TBD
N                | TBD
n                | TBD
O                | Point at infinity
P                | TBD
Q                | TBD
R                | Commitment to a random nonce in the Schnorr protocol
r                | Random nonce for the Schnorr protocol
S                | TBD
T                | TBD
U                | TBD
V                | [Value commitment](#value-commitment)
v                | Amount scalar blinded using [value commitment](#value-commitment)
W                | TBD
X                | TBD
Y                | TBD
Z                | TBD


