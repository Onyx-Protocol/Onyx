# Confidential Assets

* [Introduction](#introduction)
* [Usage](#usage)
  * [Confidential issuance](#confidential-issuance)
  * [Simple transfer](#simple-transfer)
  * [Multi-party transaction](#multi-party-transaction)
* [Definitions](#definitions)
  * [Elliptic Curve Parameters](#elliptic-curve-parameters)
  * [Zero point](#zero-point)
  * [Generators](#generators)
  * [Scalar](#scalar)
  * [Point](#point)
  * [Point Pair](#point-pair)
  * [Point operations](#point-operations)
  * [Hash256](#hash256)
  * [ScalarHash](#scalarhash)
  * [Ring Signature](#ring-signature)
  * [Borromean Ring Signature](#borromean-ring-signature)
  * [Asset ID Commitment](#asset-id-commitment)
  * [Asset Range Proof](#asset-range-proof)
  * [Value Commitment](#value-commitment)
  * [Excess Factor](#excess-factor)
  * [Excess Commitment](#excess-commitment)
  * [Value Proof](#value-proof)
  * [Value Range Proof](#value-range-proof)
  * [Record Encryption Key](#record-encryption-key)
  * [Intermediate Encryption Key](#intermediate-encryption-key)
  * [Asset ID Encryption Key](#asset-id-encryption-key)
  * [Value Encryption Key](#value-encryption-key)
  * [Asset ID Blinding Factor](#asset-id-blinding-factor)
  * [Value Blinding Factor](#value-blinding-factor)
  * [Issuance Asset Range Proof](#issuance-asset-range-proof)
* [Core algorithms](#core-algorithms)
  * [Create Ring Signature](#create-ring-signature)
  * [Verify Ring Signature](#verify-ring-signature)
  * [Create Borromean Ring Signature](#create-borromean-ring-signature)
  * [Verify Borromean Ring Signature](#verify-borromean-ring-signature)
  * [Recover Payload From Borromean Ring Signature](#recover-payload-from-borromean-ring-signature)
  * [Encrypt Payload](#encrypt-payload)
  * [Decrypt Payload](#decrypt-payload)
  * [Create Nonblinded Asset ID Commitment](#create-nonblinded-asset-id-commitment)
  * [Create Blinded Asset ID Commitment](#create-blinded-asset-id-commitment)
  * [Create Asset Range Proof](#create-asset-range-proof)
  * [Verify Asset Range Proof](#verify-asset-range-proof)
  * [Create Nonblinded Value Commitment](#create-nonblinded-value-commitment)
  * [Create Blinded Value Commitment](#create-blinded-value-commitment)
  * [Balance Blinding Factors](#balance-blinding-factors)
  * [Encrypt Value](#encrypt-value)
  * [Decrypt Value](#encrypt-value)
  * [Create Value Proof](#create-value-proof)
  * [Verify Value Proof](#verify-value-proof)
  * [Create Value Range Proof](#create-value-range-proof)
  * [Verify Value Range Proof](#verify-value-range-proof)
  * [Recover Payload From Value Range Proof](#recover-payload-from-value-range-proof)
  * [Create Excess Commitment](#create-excess-commitment)
  * [Verify Excess Commitment](#verify-excess-commitment)
  * [Verify Value Commitments Balance](#verify-value-commitments-balance)
  * [Create Transient Issuance Key](#create-transient-issuance-key)
  * [Create Issuance Asset Range Proof](#create-issuance-asset-range-proof)
  * [Verify Issuance Asset Range Proof](#verify-issuance-asset-range-proof)
* [High-level procedures](#high-level-procedures)
  * [Verify Output](#verify-output)
  * [Verify Issuance](#verify-issuance)
  * [Verify Confidential Assets](#verify-confidential-assets)
  * [Encrypt Issuance](#encrypt-issuance)
  * [Encrypt Output](#encrypt-output)
  * [Decrypt Output](#decrypt-output)
* [Integration](#integration)
  * [VM1](#vm1)
  * [Blockchain data structures](#blockchain-data-structures)
  * [Blockchain validation](#blockchain-validation)
* [Test vectors](#test-vectors)  
  


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
5. For each asset ID, where issuance program does not check an issuance key, issuer [creates a transient issuance key](#create-transient-issuance-key).
6. Issuer provides arguments for each issuance program of each asset ID.
7. Issuer [encrypts issuance](#encrypt-issuance): generates asset ID and value commitments and provides necessary range proofs.
8. Issuer remembers values `(H,c,f)` to help complete the transaction. Once outputs are fully or partially specified, these values can be discarded.
9. Issuer proceeds with the rest of the transaction creation. See [simple transfer](#simple-transfer) and [multi-party transaction](#multi-party-transaction) for details.

### Simple transfer

1. Recipient generates the following parameters and sends them privately to the sender:
    * amount and asset ID to be sent,
    * control program,
    * [REK1](#record-encryption-key).
2. Sender composes a transaction with an unencrypted output with a given control program.
3. Sender adds necessary amount of inputs to satisfy the output:
    1. For each unspent output, sender pulls values `(assetid,value,H,c,f)` from its DB.
    2. If the unspent output is not confidential, then:
        * `H=8*Decode(SHA3(assetid))`, [unblinded asset id commitment](#unblinded-asset-id-commitment)
        * `c=0`
        * `f=0`
    3. Values `(aek,vek)` can be discarded once the output is spent on the blockchain. These are kept for PQ event in case the spender will have to prove hash-based commitments to `c` and `f` blinding factors.
4. If sender needs to add a change output:
    1. Sender [encrypts](#encrypt-output) the first output.
    2. Sender [balances blinding factors](#balance-blinding-factors) to create an excess factor `q`.
    3. Sender adds unencrypted change output.
    4. Sender generates a [REK2](#record-encryption-key) for the change output.
    5. Sender [encrypts](#encrypt-output) the change output with an additional excess factor `q`.
5. If sender does not need a change output:
    1. Sender [balances blinding factors](#balance-blinding-factors) of the inputs to create an excess factor `q`.
    2. Sender [encrypts](#encrypt-output) the first output with an additional excess factor `q`.
6. Sender stores values `(assetid,value,H,c,f,aek,vek)` in its DB with its change output for later spending.
7. Sender publishes the transaction.
8. Recipient receives the transaction and identifies its output via its control program.
9. Recipient uses its [REK1](#record-encryption-key) to [decrypt output](#decrypt-output).
10. Recipient stores resulting `(assetid,value,H,c,f,aek,vek)` in its DB with its change output for later spending.
11. Recipient separately stores decrypted plaintext payload with the rest of the reference data. It is not necessary for spending.


### Multi-party transaction

1. All parties communicate out-of-band payment details (how much is being paid for what), but not cryptographic material or control programs.
2. Each party:
    1. Generates cleartext outputs: (amount, asset ID, control program, [REK](#record-encryption-key)). These include both requested payment ("I want to receive €10") and the change ("I send back to myself $142").
    2. [Encrypts](#encrypt-output) each output.
    3. [Balances blinding factors](#balance-blinding-factors) to create an excess factor `q[i]`.
    4. For each output, stores values `(assetid,value,H,c,f,aek,vek)` associated with that output in its DB for later spending.
    5. Sends `q[i]` to the party that finalizes transaction.
3. Party that finalizes transaction:
    1. Receives `{q[i]}` values from all other parties (including itself).
    2. Sums all excess factors: `qsum = ∑q[i]`
    3. [Creates excess commitment](#create-excess-commitment) out of `qsum`: `Q = qsum·G` and signs it.
    4. Finalizes transaction by placing the excess commitment to the Common Fields.
    5. Publishes the transaction.
4. If the amounts and blinding factors are balanced, transaction is valid and included in the blockchain. Parties can not see each other’s outputs, but only their own.


## Definitions

### Elliptic Curve Parameters

**The elliptic curve** is edwards25519 as defined by [[RFC7748](https://tools.ietf.org/html/rfc7748)].

`L` is the **order of edwards25519** as defined by \[[RFC8032](https://tools.ietf.org/html/rfc8032)\] (i.e. 2<sup>252</sup>+27742317777372353535851937790883648493).


### Zero point

_Zero point_ `O` is a representation of the _point at infinity_, identity element in the [edwards25519](#elliptic-curve-parameters) subgroup. It is encoded as follows:

    O = 0x0100000000000000000000000000000000000000000000000000000000000000


### Generators

**Primary generator point** (`G`) is the elliptic curve specified as "B" in Section 5.1 of [[RFC8032](https://tools.ietf.org/html/rfc8032)].

Generator `G` has the following 32-byte encoding:

    G = 0x5866666666666666666666666666666666666666666666666666666666666666

**Secondary generator point** (`J`) is the elliptic curve defined as decoded hash of the primary generator `G`:

    J = 8·Decode(SHA3-256(Encode(G)))

Generator `J` has the following 32-byte encoding:

    J = 0x00c774b875ed4e395ebb0782b4d93db838d3c4c0840bc970570517555ca71b77


### Scalar

A _scalar_ is an integer in the range from `0` to `L-1` where `L` is the order of [edwards25519](#elliptic-curve-parameters) subgroup.
Scalars are encoded as little-endian 32-byte integers.


### Point

A point is a two-dimensional point on [edwards25519](#elliptic-curve-parameters).
Points are encoded according to [RFC8032](https://tools.ietf.org/html/rfc8032).


### Point Pair

A vector of two elliptic curve [points](#point). Point pair is encoded as 64-byte string composed of 32-byte encodings of each point.


### Point operations

Elliptic curve *points* support two operations:

1. Addition/subtraction of points (`A+B`, `A-B`)
2. Scalar multiplication (`a·B`).

These operations are defined as in \[[RFC8032](https://tools.ietf.org/html/rfc8032)\].

*Point pairs* support the same operations defined as:

1. Sum of two pairs is a pair of sums:

        (A,B) + (C,D) == (A+C, B+D)

2. Multiplication of a pair by a [scalar](#scalar) is a pair of scalar multiplications of each point:

        x·(A,B) == (x·A,x·B)


### Hash256

`Hash256` is a secure hash function that takes a variable-length binary string `x` as input and outputs a 256-bit string.

    Hash256(x) = SHA3-256("ChainCA-256" || x)


### StreamHash

`StreamHash` is a secure extendable-output hash function that takes a variable-length binary string `x` as input
and outputs a variable-length hash string depending on a number of bytes (`n`) requested.

    StreamHash(x, n) = SHAKE256("ChainCA-stream" || x, n)


### ScalarHash

`ScalarHash` is a secure hash function that takes a variable-length binary string `x` as input and outputs a [scalar](#scalar):

1. For the input string `x` compute a 512-bit hash `h`:

        h = SHA3-512("ChainCA-scalar" || x)

2. Interpret `h` as a little-endian integer and reduce modulo subgroup [order](#elliptic-curve-parameters) `L`:

        s = h mod L

3. Return the resulting scalar `s`.


### Ring Signature

Ring signature is a variable-length string representing a signature of knowledge of a one private key among an ordered set of public keys (specified separately). In other words, ring signature implements an OR function of the public keys: “I know the private key for A, or B or C”. Ring signatures are used in [asset range proofs](#asset-range-proof).

The ring signature is encoded as a string of `n+1` 32-byte elements where `n` is the number of public keys provided separately (typically stored or imputed from the data structure containing the ring signature):

    {e, s[0], s[1], ..., s[n-1]}

Each 32-byte element is an integer coded using little endian convention. I.e., a 32-byte string `x` `x[0],...,x[31]` represents the integer `x[0] + 2^8 · x[1] + ... + 2^248 · x[31]`.


### Borromean Ring Signature

Borromean ring signature ([[Maxwell2015](https://github.com/Blockstream/borromean_paper)]) is a data structure representing several [ring signatures](#ring-signature) compactly joined with an AND function of the ring signatures: “I know the private key for (A or B) and (C or D)”. Borromean ring signatures are used in [value range proofs](#value-range-proof) that prove the range of multiple digits at once.

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


### Asset ID Point

_Asset ID point_ is a [point](#point) representing an asset ID. It is defined as follows:

1. Let `counter = 0`.
2. Calculate `Hash256("AssetID" || assetID || counter)` where `counter` is encoded as a 64-bit unsigned integer using little-endian convention.
3. Decode the resulting hash as a [point](#point) `P` on the elliptic curve.
4. If the point is invalid, increment `counter` and go back to step 2. This will happen on average for half of the asset IDs.
5. Calculate point `A = 8·P` (8 is a cofactor in edwards25519) which belongs to a subgroup [order](#elliptic-curve-parameters) `L`.
6. Return `A`.


### Asset ID Commitment

An asset ID commitment `AC` is an ElGamal commitment represented by a [point pair](#point-pair):

    AC = (H, Ba)
    
    H  = A + b·G
    Ba = b·J
   
where:      

* `A` is an [Asset ID Point](#asset-id-point), an orthogonal point representing an asset ID.  
* `b` is a blinding [scalar](#scalar) for the asset ID.
* `G`, `J` are [generator points](#generators).

The asset ID commitment can either be nonblinded or blinded:

* [Create Nonblinded Asset ID Commitment](#create-nonblinded-asset-id-commitment)
* [Create Blinded Asset ID Commitment](#create-blinded-asset-id-commitment)


### Asset Range Proof

The asset range proof demonstrates that a given [asset ID commitment](#asset-id-commitment) commits to one of the asset IDs specified in the transaction inputs. A [whole-transaction validation procedure](#verify-confidential-assets) makes sure that all of the declared asset ID commitments in fact belong to the transaction inputs.

Asset range proof can be [nonconfidential](#nonconfidential-asset-range-proof) or [confidential](#confidential-asset-range-proof).

#### Non-Confidential Asset Range Proof

Field                        | Type      | Description
-----------------------------|-----------|------------------
Type                         | byte      | Contains value 0x00 to indicate the commitment is not blinded.
Asset ID                     | [AssetID](blockchain.md#asset-id)   | 32-byte asset identifier.

#### Confidential Asset Range Proof

Field                        | Type      | Description
-----------------------------|-----------|------------------
Type                         | byte      | Contains value 0x01 to indicate the commitment is blinded.
Asset ID Commitments         | [List](blockchain.md#list)\<[Asset ID Commitment](#asset-id-commitment)\> | List of asset ID commitments from the transaction inputs used in the range proof.
Asset Ring Signature         | [Ring Signature](#ring-signature) | A ring signature proving that the asset ID committed in the output belongs to the set of declared input commitments.


### Value Commitment

A value commitment `VC` is an ElGamal commitment represented by a [point pair](#point-pair):

    VC = (V, Bv)
    
    V  = v·H  + f·G
    Bv = v·Ba + f·J

where:

* `(H, Ba)` is an [asset ID commitment](#asset-id-commitment).
* `v` is an amount being committed.
* `f` is a blinding [scalar](#scalar) for the amount.
* `G`, `J` are [generator points](#generators).

The asset ID commitment can either be _nonblinded_ or _blinded_:

* [Create Nonblinded Value Commitment](#create-nonblinded-value-commitment)
* [Create Blinded Value Commitment](#create-blinded-value-commitment)


### Excess Factor

Excess factor is a [scalar](#scalar) representing a net difference between input and output blinding factors. It is computed using [Balance Blinding Factors](#balance-blinding-factors) and used to create an [Excess Commitment](#excess-commitment).


### Excess Commitment

An excess commitment `QC` is an ElGamal commitment to an [excess factor](#excess-factor) represented by a [point pair](#point-pair) together with a Schnorr signature proving the equality of the discrete logarithms in both points (`e,s`):

    QC = (q·G, q·J, e, s)

Excess pair `(q·G, q·J)` is used to [verify balance of value commitments](#verify-value-commitments-balance) while the associated signature proves that the points do not contain a factor affecting the amount of any asset.

See: 

* [Create Excess Commitment](#create-excess-commitment))
* [Verify Excess Commitment](#verify-excess-commitment))


### Value Proof

Value proof demonstrates that a given [value commitment](#value-commitment) encodes a specific amount and asset ID. It is used to privately prove the contents of an output without revealing blinding factors to a counter-party or an HSM.

See:

* [Create Value Proof](#create-value-proof)
* [Verify Value Proof](#verify-value-proof)


### Value Range Proof

Value range proof demonstrates that a [value commitment](#value-commitment) encodes a value between 0 and 2<sup>63</sup>–1. The 63-bit limit is chosen for consistency with the numeric limits defined for the asset version 1 outputs and VM version 1 [numbers](vm1.md#vm-number).

For the most compact encoding, value range proof uses base-4 digits represented by 4-key ring signatures proving the value of each pair of bits. If the number of bits is odd, the last ring signature contains only 2 elements proving the value of the highest-order bit. All ring signatures share the same e-value (see below) forming a so-called "[borromean ring signature](#borromean-ring-signature)".

Value range proof allows a space-privacy tradeoff by making a smaller number of bits confidential while exposing a "minimum value" and a decimal exponent. The complete value is broken down in the following components:

    value = vmin + (10^exp)·(d[0]·(4^0) + ... + d[m-1]·(4^(m-1)))

Where d<sub>i</sub> is the i’th digit in a m-digit mantissa (that has either 2·m–1 or 2·m bits). Exponent `exp` and the minimum value `vmin` are public and by default set to zero by the user creating the transaction.

Field                     | Type      | Description
--------------------------|-----------|------------------
Number of bits            | byte      | Integer `n` indicating number of confidential mantissa bits between 1 and 63.
Exponent                  | byte      | Integer `exp` indicating the decimal exponent from 0 to 10.
Minimum value             | varint63  | Minimum value `vmin` from 0 to 2<sup>63</sup>–1.
Digit commitments         | [pubkey]  | List of `(n+1)/2 – 1` individual digit pedersen commitments where `n` is the number of mantissa bits.
Borromean Ring Signature  | [Borromean Ring Signature](#borromean-ring-signature) | List of all 32-byte elements comprising all ring signatures proving the value of each digit.

The total number of elements in the [Borromean Ring Signature](#borromean-ring-signature) is `1 + 4·n/2` where `n` is number of bits and `n/2` is a number of rings.

### Record Encryption Key

Record encryption key (REK or `rek`) is a pair of [ChainKD](chainkd.md) extended public keys (xpubs).

    REK = {xpub1, xpub2}

It is used to decrypt the payload data from the [value range proof](#value-range-proof), and derive [asset ID encryption key](#asset-id-encryption-key) and [value encryption key](#value-encryption-key).

The first `xpub1` is used to derive more specific keys as described below that all share the same second key `xpub2`.
The second `xpub2` is used to derive the entire hierarchies of encryption keys, so that a [REK](#record-encryption-key), or [IEK](#intermediate-encryption-key) could be shared for the entire account instead of per-transaction.

### Intermediate Encryption Key

Intermediate encryption key (IEK or `iek`) allows decrypting the asset ID and the value in the output commitment. It is derived from the [record encryption key](#record-encryption-key) as follows:

    IEK = {ND(REK.xpub1, "IEK"), REK.xpub2}

where `ND` is non-hardened derivation as defined by [ChainKD](chainkd.md#derive-non-hardened-extended-public-key).

### Asset ID Encryption Key

Asset ID encryption key (AEK or `aek`) allows decrypting the asset ID in the output commitment. It is derived from the [intermediate encryption key](#intermediate-encryption-key) as follows:

    AEK = {ND(IEK.xpub1, "AEK"), IEK.xpub2}

where `ND` is non-hardened derivation as defined by [ChainKD](chainkd.md#derive-non-hardened-extended-public-key).

### Value Encryption Key

Value encryption key (VEK or `vek`) allows decrypting the amount in the output commitment. It is derived from the [intermediate encryption key](#intermediate-encryption-key) as follows:

    VEK = {ND(IEK.xpub1, "VEK"), IEK.xpub2}

where `ND` is non-hardened derivation as defined by [ChainKD](chainkd.md#derive-non-hardened-extended-public-key).


### Asset ID Blinding Factor

A [scalar](#scalar) `c` used to produce a blinded asset ID commitment out of a [cleartext asset ID commitment](#asset-id-commitment):

    AC = (A + c·G, c·J)

The asset ID blinding factor is created by [Create Blinded Asset ID Commitment](#create-blinded-asset-id-commitment).


### Value Blinding Factor

An [scalar](#scalar) `f` used to produce a [value commitment](#value-commitment) out of the [asset ID commitment](#asset-id-commitment) `(H, Ba)`:

    VC = (value·H + f·G, value·Ba + f·J)

The value blinding factors are created by [Create Blinded Value Commitment](#create-blinded-value-commitment) algorithm.


### Issuance Asset Range Proof

The issuance asset range proof demonstrates that a given [confidential issuance](asset-version-2-confidential-issuance-witness) commits to one of the asset IDs specified in the transaction inputs. It contains a ring signature. The other inputs to the [verification procedure](#verify-issuance-asset-range-proof) are computed from other elements in the confidential issuance witness, as part of the [validation procedure](#validate-transaction-input).

The size of the ring signature (`n+1` 32-byte elements) and the number of issuance keys (`n`) are derived from `n` [asset issuance choices](data.md#asset-issuance-choice) specified outside the range proof.

The proof also contains a _tracing point_ that that lets any issuer to prove or disprove whether the issuance is performed by their issuance key.

TBD: extend this to allow "watch keys" (will require change of `{Y}` to `{(Y,W)}` and maybe a pair of tracing points).

Field                           | Type             | Description
--------------------------------|------------------|------------------
Issuance Ring Signature         | [Ring Signature](#ring-signature)   | A ring signature proving that the issuer of an encrypted asset ID approved the issuance.
Tracing Point                   | [Point](#point)  | A point that lets any issuer to prove or disprove if this issuance is done by them.
Issuance Keys                   | [List](blockchain.md#list)\<[Point](#point)\> | Keys to be used to calculate the public key for the corresponding index in the ring signature.
Issuance Program                | [Program](blockchain.md#program)  | [Version of the VM](#vm-version) that executes the issuance signature program.
Issuance Signature Program      | varstring31      | Predicate committed to by the issuance asset range proof, which is evaluated to ensure that the transaction is authorized.
Program Arguments Count         | varint31         | Number of [program arguments](#program-arguments) that follow.
Program Arguments               | [varstring31]    | Data passed to the issuance signature program.





## Core algorithms

### Create Ring Signature

**Inputs:**

1. `msg`: the string to be signed.
2. `B`: base [point](#point) to verify the signature (not necessarily a [generator](#generator) point).
3. `{P[i]}`: `n` [points](#point) representing the public keys.
4. `j`: the index of the designated public key, so that `P[j] == p·B`.
5. `p`: the secret [scalar](#scalar) representing a private key for the public key `P[j]`.

**Output:** `{e0, s[0], ..., s[n-1]}`: the ring signature, `n+1` 32-byte elements.

**Algorithm:**

1. Let `counter = 0`.
2. Let the `msghash` be a hash of the input non-secret data: `msghash = Hash256(B || P[0] || ... || P[n-1] || msg)`.
3. Calculate a sequence of: `n-1` 32-byte random values, 64-byte `nonce` and 1-byte `mask`: `{r[i], nonce, mask} = StreamHash(counter || msghash || p || j, 32·(n-1) + 64 + 1)`, where:
    * `counter` is encoded as a 64-bit little-endian integer,
    * `p` is encoded as a 256-bit little-endian integer,
    * `j` is encoded as a 64-bit little-endian integer.
4. Calculate `k = nonce mod L`, where `nonce` is interpreted as a 64-byte little-endian integer and reduced modulo subgroup order `L`.
5. Calculate the initial e-value, let `i = j+1 mod n`:
    1. Calculate `R[i]` as the [point](#point) `k·B`.
    2. Define `w[j]` as `mask` with lower 4 bits set to zero: `w[j] = mask & 0xf0`.
    3. Calculate `e[i] = ScalarHash(R[i] || msghash || i || w[j])` where `i` is encoded as a 64-bit little-endian integer.
6. For `step` from `1` to `n-1` (these steps are skipped if `n` equals 1):
    1. Let `i = (j + step) mod n`.
    2. Calculate the forged s-value `s[i] = r[step-1]`, where `r[j]` is interpreted as a 64-byte little-endian integer and reduced modulo `L`.
    3. Define `z[i]` as `s[i]` with the most significant 4 bits set to zero.
    4. Define `w[i]` as a most significant byte of `s[i]` with lower 4 bits set to zero: `w[i] = s[i][31] & 0xf0`.
    5. Let `i’ = i+1 mod n`.
    6. Calculate point `R[i’] = z[i]·B - e[i]·P[i]`.
    7. Calculate `e[i’] = ScalarHash(R[i’] || msghash || i’ || w[i])` where `i’` is encoded as a 64-bit little-endian integer.
7. Calculate the non-forged `z[j] = k + p·e[j] mod L` and encode it as a 32-byte little-endian integer.
8. If `z[j]` is greater than 2<sup>252</sup>–1, then increment the `counter` and try again from the beginning. The chance of this happening is below 1 in 2<sup>124</sup>.
9. Define `s[j]` as `z[j]` with 4 high bits set to high 4 bits of the `mask`.
10. Return the ring signature `{e[0], s[0], ..., s[n-1]}`, total `n+1` 32-byte elements.


### Verify Ring Signature

**Inputs:**

1. `msg`: the string being signed.
2. `B`: base [point](#point) to verify the signature (not necessarily a [generator](#generator) point).
3. `{P[i]}`: `n` [points](#point) representing the public keys.
4. `e[0], s[0], ... s[n-1]`: ring signature consisting of `n+1` 32-byte elements.


**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. Let the `msghash` be a hash of the input non-secret data: `msghash = Hash256(B || P[0] || ... || P[n-1] || msg)`.
2. For each `i` from `0` to `n-1`:
    1. Define `z[i]` as `s[i]` with the most significant 4 bits set to zero (see note below).
    2. Define `w[i]` as a most significant byte of `s[i]` with lower 4 bits set to zero: `w[i] = s[i][31] & 0xf0`.
    3. Calculate point `R[i+1] = z[i]·B - e[i]·P[i]`.
    4. Calculate `e[i+1] = ScalarHash(R[i+1] || msghash || i+1 || w[i])` where `i+1` is encoded as a 64-bit little-endian integer.
3. Return true if `e[0]` equals `e[n]`, otherwise return false.

Note: when the s-values are decoded as little-endian integers we must set their 4 most significant bits to zero in order to restore the original scalar as produced while [creating the range proof](#create-asset-range-proof). During signing the non-forged s-value has its 4 most significant bits set to random bits to make it indistinguishable from the forged s-values.




### Create Borromean Ring Signature

**Inputs:**

1. `msg`: the string to be signed.
2. `n`: number of rings.
3. `m`: number of signatures in each ring.
4. `{B[i]}`: `n` base [points](#point) to verify the signature (not necessarily [generator](#generator) points).
5. `{P[i,j]}`: `n·m` [points](#point) representing public keys.
6. `{p[i]}`: the list of `n` [scalars](#scalar) representing private keys.
7. `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[i,j] == p[i]·B[i]`.
8. `{payload[i]}`: sequence of `n·m` random 32-byte elements.

**Output:** `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.

**Algorithm:**

1. Let the `msghash` be a hash of the input non-secret data: `msghash = Hash256(n || m || {B[i]} || {P[i,j]} || msg)` where `n` and `m` are encoded as 64-bit little-endian integers.
2. Let `counter = 0`.
3. Let `cnt` byte contain lower 4 bits of `counter`: `cnt = counter & 0x0f`.
4. Calculate a sequence of `n·m` 32-byte random overlay values: `{o[i]} = StreamHash(counter || msghash || {p[i]} || {j[i]}, 32·n·m)`, where:
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
        2. Calculate `R[t,j’]` as the point `k[t]·B[t]`.
        3. Calculate `e[t,j’] = ScalarHash(cnt, R[t, j’] || msghash || t || j’ || w[t,j])` where `t` and `j’` are encoded as 64-bit little-endian integers.
    7. If `j ≠ m-1`, then for `i` from `j+1` to `m-1`:
        1. Calculate the forged s-value: `s[t,i] = r[m·t + i]`.
        2. Define `z[t,i]` as `s[t,i]` with 4 most significant bits set to zero.
        3. Define `w[t,i]` as a most significant byte of `s[t,i]` with lower 4 bits set to zero: `w[t,i] = s[t,i][31] & 0xf0`.
        4. Let `i’ = i+1 mod m`.
        5. Calculate point `R[t,i’] = z[t,i]·B[t] - e[t,i]·P[t,i]`.
        6. Calculate `e[t,i’] = ScalarHash(cnt, R[t,i’] || msghash || t || i’ || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
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
        5. Calculate point `R[t,i’] = z[t,i]·B[t] - e[t,i]·P[t,i]`. If `i` is zero, use `e0` in place of `e[t,0]`.
        6. Calculate `e[t,i’] = ScalarHash(cnt, R[t,i’] || msghash || t || i’ || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
    4. Calculate the non-forged `z[t,j] = k[t] + p[t]·e[t,j] mod L` and encode it as a 32-byte little-endian integer.
    5. If `z[t,j]` is greater than 2<sup>252</sup>–1, then increment the `counter` and try again from step 3. The chance of this happening is below 1 in 2<sup>124</sup>.
    6. Define `s[t,j]` as `z[t,j]` with 4 high bits set to `mask[t]` bits.
9. Set top 4 bits of `e0` to the lower 4 bits of `counter`.
10. Return the [borromean ring signature](#borromean-ring-signature):
    * `{e,s[t,j]}`: `n·m+1` 32-byte elements.



### Verify Borromean Ring Signature

**Inputs:**

1. `msg`: the string to be signed.
2. `n`: number of rings.
3. `m`: number of signatures in each ring.
4. `{B[i]}`: `n` base [points](#point) to verify the signature (not necessarily [generator](#generator) points).
5. `{P[i,j]}`: `n·m` public keys, [points](data.md#public-key) on the elliptic curve.
6. `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. Let the `msghash` be a hash of the input non-secret data: `msghash = SHA3-256(n || m || {B[i]} || {P[i,j]} || msg)` where `n` and `m` are encoded as 64-bit little-endian integers.
2. Define `E` to be an empty binary string.
3. Set `cnt` byte to the value of top 4 bits of `e0`: `cnt = e0[31] >> 4`.
4. Set top 4 bits of `e0` to zero.
5. For `t` from `0` to `n-1` (each ring):
    1. Let `e[t,0] = e0`.
    2. For `i` from `0` to `m-1` (each item):
        1. Calculate `z[t,i]` as `s[t,i]` with the most significant 4 bits set to zero.
        2. Calculate `w[t,i]` as a most significant byte of `s[t,i]` with lower 4 bits set to zero: `w[t,i] = s[t,i][31] & 0xf0`.
        3. Let `i’ = i+1 mod m`.
        4. Calculate point `R[t,i’] = z[t,i]·B[t] - e[t,i]·P[t,i]`. Use `e0` instead of `e[t,0]` in each ring.
        5. Calculate `e[t,i’] = ScalarHash(cnt || R[t,i’] || msghash || t || i’ || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
    3. Append `e[t,0]` to `E`: `E = E || e[t,0]`, where `e[t,0]` is encoded as a 32-byte little-endian integer.
6. Calculate `e’ = ScalarHash(E)`.
7. Return `true` if `e’` equals to `e0`. Otherwise, return `false`.



### Recover Payload From Borromean Ring Signature

**Inputs:**

1. `msg`: the string to be signed.
2. `n`: number of rings.
3. `m`: number of signatures in each ring.
4. `{B[i]}`: `n` base [points](#point) to verify the signature (not necessarily [generator](#generator) points).
5. `{P[i,j]}`: `n·m` public keys, [points](data.md#public-key) on the elliptic curve.
6. `{p[i]}`: the list of `n` scalars representing private keys.
7. `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[i,j] == p[i]·G`.
8. `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.

**Output:** `{payload[i]}` list of `n·m` random 32-byte elements or `nil` if signature verification failed.

**Algorithm:**

1. Let the `msghash` be a hash of the input non-secret data: `msghash = SHA3-256(n || m || {B[i]} || {P[i,j]} || msg)` where `n` and `m` are encoded as 64-bit little-endian integers.
2. Define `E` to be an empty binary string.
3. Set `cnt` byte to the value of top 4 bits of `e0`: `cnt = e0[31] >> 4`.
4. Let `counter` integer equal `cnt`.
5. Calculate a sequence of `n·m` 32-byte random overlay values: `{o[i]} = StreamHash(counter || msghash || {p[i]} || {j[i]}, 32·n·m)`, where:
    * `counter` is encoded as a 64-bit little-endian integer,
    * private keys `{p[i]}` are encoded as concatenation of 256-bit little-endian integers,
    * secret indexes `{j[i]}` are encoded as concatenation of 64-bit little-endian integers.
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
        6. Calculate point `R[t,i’] = z[t,i]·B[t] - e[t,i]·P[t,i]` and encode it as a 32-byte [public key](data.md#public-key). Use `e0` instead of `e[t,0]` in each ring.
        7. Calculate `e[t,i’] = ScalarHash(cnt || R[t,i’] || msghash || t || i’ || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
    3. Append `e[t,0]` to `E`: `E = E || e[t,0]`, where `e[t,0]` is encoded as a 32-byte little-endian integer.
8. Calculate `e’ = ScalarHash(E)`.
9. Return `payload` if `e’` equals to `e0`. Otherwise, return `nil`.




### Encrypt Payload

**Inputs:**

1. `n`: number of 32-byte elements in the plaintext to be encrypted.
2. `{pt[i]}`: list of `n` 32-byte elements of plaintext data.
3. `ek`: the encryption/authentication key unique to this payload.

**Output:** the `{ct[i]}`: list of `n` 32-byte ciphertext elements, where the last one is a 32-byte MAC.

**Algorithm:**

1. Calculate a keystream, a sequence of 32-byte random values: `{keystream[i]} = StreamHash(ek, 32·n)`.
2. Encrypt the plaintext payload: `{ct[i]} = {pt[i] XOR keystream[i]}`.
3. Calculate MAC: `mac = Hash256(ek || ct[0] || ... || ct[n-1])`.
4. Return a sequence of `n+1` 32-byte elements: `{ct[0], ..., ct[n-1], mac}`.


### Decrypt Payload

**Inputs:**

1. `n`: number of 32-byte elements in the ciphertext to be decrypted.
2. `{ct[i]}`: list of `n+1` 32-byte ciphertext elements, where the last one is MAC (32 bytes).
3. `ek`: the encryption/authentication key.

**Output:** the `{pt[i]}` or `nil`, if authentication failed.

**Algorithm:**

1. Calculate MAC’: `mac’ = Hash256(ek || ct[0] || ... || ct[n-1])`.
2. Extract the transmitted MAC: `mac = ct[n]`.
3. Compare calculated  `mac’` with the received `mac`. If they are not equal, return `nil`.
4. Calculate a keystream, a sequence of 32-byte random values: `{keystream[i]} = StreamHash(ek, 32·n)`.
5. Decrypt the plaintext payload: `{pt[i]} = {ct[i] XOR keystream[i]}`.
5. Return `{pt[i]}`.


### Create Nonblinded Asset ID Commitment

**Input:** `assetID`: the cleartext asset ID.

**Output:**  `(H,O)`: the nonblinded [asset ID commitment](#asset-id-commitment).

**Algorithm:**

1. Compute an [asset ID point](#asset-id-point):
        
        A = 8·Hash256(assetID || counter)

2. Return [point pair](#point-pair) `(A,O)` where `O` is a [zero point](#zero-point).


### Create Blinded Asset ID Commitment

**Inputs:**

1. `assetID`: the cleartext asset ID.
2. `aek`: the [asset ID encryption key](#asset-id-encryption-key).

**Outputs:**  

1. `(H,Ba)`: the blinded [asset ID commitment](#asset-id-commitment).
2. `c`: the [blinding factor](#asset-id-blinding-factor) such that `H == A + c·G, Ba = c·J`.

**Algorithm:**

1. Compute an [asset ID point](#asset-id-point): 
        
        A = 8·Hash256(assetID || counter)

2. Compute [asset ID blinding factor](#asset-id-blinding-factor):

        s = ScalarHash(assetID || aek)

3. Compute an [asset ID commitment](#asset-id-commitment): 
    
        AC = (H, Ba)
        H  = A + c·G
        Ba = c·J

4. Return `(AC, c)`.



### Create Asset Range Proof

**Inputs:**

1. `H’`: the output [asset ID commitment](#asset-id-commitment) for which the range proof is being created.
2. `(ea,ec)`: the [encrypted asset ID](#encrypted-asset-id) including the encrypted blinding factor for `H’`.
3. `{H[i]}`: `n` candidate [asset ID commitments](#asset-id-commitment).
4. `j`: the index of the designated commitment among the input asset ID commitments, so that `H’ == H[j] + (c’ - c)·G`.
5. `c’`: the [blinding factor](#asset-id-blinding-factor) for the commitment `H’`.
6. `c`: the [blinding factor](#asset-id-blinding-factor) for the candidate commitment `H[j]`.

**Output:** an asset range proof consisting of a list of input asset ID commitments and a ring signature.

**Algorithm:**

1. Calculate the message to sign: `msg = SHA3-256(0x55 || H’ || H[0] || ... || H[n-1] || ea || ec)`.
2. Calculate the set of public keys for the ring signature from the set of input asset ID commitments: `P[i] = H’ - H[i]`.
3. Calculate the private key: `p = c’ - c mod L`.
4. [Create a ring signature](#create-ring-signature) using `msg`, `{P[i]}`, `j`, and `p`.
5. Return the list of asset ID commitments `{H[i]}` and the ring signature `e[0], s[0], ... s[n-1]`.

Note: unlike the [value range proof](#value-range-proof), this ring signature is not used to store encrypted payload data because decrypting it would reveal the asset ID of one of the inputs to the recipient.


### Verify Asset Range Proof

**Inputs:**

1. `H’`: the target [asset ID commitment](#asset-id-commitment).
2. `(ea,ec)`: the [encrypted asset ID](#encrypted-asset-id) including the encrypted blinding factor for `H’`.
3. The to-be-verified [asset range proof](#asset-range-proof) consisting of:
    1. `{H[i]}`: `n` input [asset ID commitments](#asset-id-commitment).
    2. `e[0], s[0], ... s[n-1]`: the ring signature.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. Calculate `msg = SHA3-256(0x55 || H’ || H[0] || ... || H[n-1] || ea || ec)`.
2. Calculate the set of public keys for the ring signature from the set of input asset ID commitments: `P[i] = H’ - H[i]`.
3. [Verify the ring signature](#verify-ring-signature) `e[0], s[0], ... s[n-1]` with `msg` and `{P[i]}`.
4. Return true if verification was successful, and false otherwise.




### Create Nonblinded Value Commitment

**Inputs:**

1. `value`: the cleartext amount,
2. `(H,Ba)`: the [asset ID commitment](#asset-id-commitment).

**Output:** the value commitment `VC` represented by a [point pair](#point-pair).

**Algorithm:**

1. Calculate [point pair](#point-pair) `VC = value·(H, Ba)`.
2. Return `VC`.

### Create Blinded Value Commitment

**Inputs:**

1. `vek`: the [value encryption key](#value-encryption-key) for the given output,
2. `value`: the amount to be blinded in the output,
3. `(H,Ba)`: the [asset ID commitment](#asset-id-commitment).

**Output:** `(VC, f)`: the pair of a [value commitment](#value-commitment) and its blinding factor.

**Algorithm:**

1. Calculate `f = ScalarHash(0xbf || vek)`.
2. Calculate point `V = value·H + f·G`.
3. Calculate point `Bv = value·Ba + f·J`.
4. Create a [point pair](#point-pair): `VC = (V, Bv)`.
5. Return `(VC, f)`.


### Balance Blinding Factors

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



### Create Value Proof

TBD.

1. Take `value, assetid, c, f`.
2. Compute `q = value*c + f`
3. Compute excess commitment `(Q,e,s)` from `q` (pubkey with signature).
4. Return `value, assetid, Q,e,s`.


### Verify Value Proof

TBD.

1. Take `value,assetid,Q,e,s`.
2. Verify excess commitment `Q,e,s`.
3. Compute nonblinded asset commitment: `A = 8*Decode(SHA3(assetid))`.
4. Compute nonblinded value commitment: `V’ = value*A`.
5. Add point `Q` to `V’`: `V’ = V’+Q`.
6. Verify that `V’` equals `V` in the given output.



### Create Value Range Proof

**Inputs:**

1. `H’`: the [asset ID commitment](#asset-id-commitment).
2. `V`: the [value commitment](#value-commitment).
3. `(ev,ef)`: the [encrypted value](#encrypted-value) including its blinding factor.
4. `N`: the number of bits to be blinded.
5. `value`: the 64-bit amount being encrypted and blinded.
6. `{pt[i]}`: plaintext payload string consisting of `2·N - 1` 32-byte elements.
7. `f`: the [value blinding factor](#value-blinding-factor).
8. `rek`: the [record encryption key](#record-encryption-key).

Note: this version of the signing algorithm does not use decimal exponent or minimum value and sets them both to zero.

**Output:** the [value range proof](#value-range-proof) consisting of:

* `N`: number of blinded bits (equals to `2·n`),
* `exp`: exponent (zero),
* `vmin`: minimum value (zero),
* `{D[t]}`: `n-1` digit commitments encoded as [public keys](data.md#public-key) (excluding last digit commitment),
* `{e,s[t,j]}`: `1 + 4·n` 32-byte elements representing a [borromean ring signature](#borromean-ring-signature),

In case of failure, returns `nil` instead of the range proof.

**Algorithm:**

1. Check that `N` belongs to the set `{8,16,32,48,64}`; if not, halt and return nil.
2. Check that `value` is less than `2^N`; if not, halt and return nil.
3. Define `vmin = 0`.
4. Define `exp = 0`.
5. Define `base = 4`.
6. Calculate payload encryption key unique to this payload and the value: `pek = SHA3-256(0xec || rek || f || V)`.
7. Calculate the message to sign: `msg = SHA3-256(H’ || V || N || exp || vmin || ev || ef)` where `N`, `exp`, `vmin` are encoded as 64-bit little-endian integers.
8. Let number of digits `n = N/2`.
9. [Encrypt the payload](#encrypt-payload) using `pek` as a key and `2·N-1` 32-byte plaintext elements to get `2·N` 32-byte ciphertext elements: `{ct[i]} = EncryptPayload({pt[i]}, pek)`.
10. Calculate 64-byte digit blinding factors for all but last digit: `{b[t]} = StreamHash(0xbf || msg || f, 64·(n-1))`.
11. Interpret each 64-byte `b[t]` (`t` from 0 to `n-2`) is interpreted as a little-endian integer and reduce modulo `L` to a 32-byte scalar.
12. Calculate the last digit blinding factor: `b[n-1] = f - ∑b[t] mod L`, where `t` is from 0 to `n-2`.
13. For `t` from `0` to `n-1` (each digit):
    1. Calculate `digit[t] = value & (0x03 << 2·t)` where `<<` denotes a bitwise left shift.
    2. Calculate `D[t] = digit[t]·H + b[t]·G`.
    3. Calculate `j[t] = digit[t] >> 2·t` where `>>` denotes a bitwise right shift.
    4. For `i` from `0` to `base-1` (each digit’s value):
        1. Calculate point `P[t,i] = D[t] - i·(base^t)·H’`.
14. [Create Borromean Ring Signature](#create-borromean-ring-signature) `brs` with the following inputs:
    1. `msg` as the message to sign.
    2. `n`: number of rings.
    3. `m = base`: number of signatures per ring.
    4. `{P[i,j]}`: `n·m` public keys, [points](data.md#public-key) on the elliptic curve.
    5. `{b[i]}`: the list of `n` blinding factors as private keys.
    6. `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[i,j] == b[i]·G`.
    7. `{r[i]} = {ct[i]}`: random string consisting of `n·m` 32-byte ciphertext elements.
15. If failed to create borromean ring signature `brs`, return nil. The chance of this happening is below 1 in 2<sup>124</sup>. In case of failure, retry [creating blinded value commitments](#create-blinded-value-commitments) with incremented counter. This would yield a new blinding factor `f` that will produce different digit blinding keys in this algorithm.
16. Return the [value range proof](#value-range-proof):
    * `N`:  number of blinded bits (equals to `2·n`),
    * `exp`: exponent (zero),
    * `vmin`: minimum value (zero),
    * `{D[t]}`: `n-1` digit commitments encoded as [public keys](data.md#public-key) (excluding the last digit commitment),
    * `{e,s[t,j]}`: `1 + n·4` 32-byte elements representing a [borromean ring signature](#borromean-ring-signature),



### Verify Value Range Proof

**Inputs:**

1. `H’`: the [verified](#verify-asset-range-proof) [asset ID commitment](#asset-id-commitment).
2. `V`: the [value commitment](#value-commitment).
3. `(ev,ef)`: the [encrypted value](#encrypted-value) including its blinding factor.
4. `VRP`: the [value range proof](#value-range-proof) consisting of:
    * `N`: the number of bits in blinded mantissa (8-bit integer, `N = 2·n`).
    * `exp`: the decimal exponent (8-bit integer).
    * `vmin`: the minimum amount (64-bit integer).
    * `{D[t]}`: the list of `n-1` digit pedersen commitments encoded as [public keys](data.md#public-key).
    * `{e0, s[i,j]...}`: the [borromean ring signature](#borromean-ring-signature) encoded as a sequence of `1 + 4·n` 32-byte integers.

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
3. Calculate the message to verify: `msg = SHA3-256(H’ || V || N || exp || vmin || ev || ef)` where `N`, `exp`, `vmin` are encoded as 64-bit little-endian integers.
4. Calculate last digit commitment `D[n-1] = (10^(-exp))·(V - vmin·H) - ∑(D[t])`, where `∑(D[t])` is a sum of all but the last digit commitment specified in the input to this algorithm.
5. For `t` from `0` to `n-1` (each ring):
    1. Define `base = 4`.
    2. For `i` from `0` to `base-1` (each digit’s value):
        1. Calculate point `P[t,i] = D[t] - i·(base^t)·H`.
6. [Verify Borromean Ring Signature](#verify-borromean-ring-signature) with the following inputs:
    1. `msg`: the 32-byte string being verified.
    2. `n`: number of rings.
    3. `m=base`: number of signatures in each ring.
    4. `{P[i,j]}`: `n·m` public keys, [points](data.md#public-key) on the elliptic curve.
    5. `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.
7. Return `true` if verification succeeded, or `false` otherwise.



### Recover Payload From Value Range Proof

**Inputs:**

1. `H`: the [verified](#verify-asset-range-proof) [asset ID commitment](#asset-id-commitment).
2. `V`: the [value commitment](#value-commitment).
3. `(ev,ef)`: the [encrypted value](#encrypted-value) including its blinding factor.
4. Value range proof consisting of:
    * `N`: the number of bits in blinded mantissa (8-bit integer, `N = 2·n`).
    * `exp`: the decimal exponent (8-bit integer).
    * `vmin`: the minimum amount (64-bit integer).
    * `{D[t]}`: the list of `n-1` digit pedersen commitments encoded as [public keys](data.md#public-key).
    * `{e0, s[i,j]...}`: the [borromean ring signature](#borromean-ring-signature) encoded as a sequence of `1 + 4·n` 32-byte integers.
5. `value`: the 64-bit amount being encrypted and blinded.
6. `f`: the [value blinding factor](#value-blinding-factor).
7. `rek`: the [record encryption key](#record-encryption-key).


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
3. Calculate the message to verify: `msg = SHA3-256(H || V || N || exp || vmin || ev || ef)` where `N`, `exp`, `vmin` are encoded as 64-bit little-endian integers.
4. Calculate last digit commitment `D[n-1] = (10^(-exp))·(V - vmin·H) - ∑(D[t])`, where `∑(D[t])` is a sum of all but the last digit commitment specified in the input to this algorithm.
5. Calculate 64-byte digit blinding factors for all but last digit: `{b[t]} = StreamHash(0xbf || msg || f, 64·(n-1))`.
6. Interpret each 64-byte `b[t]` (`t` from 0 to `n-2`) is interpreted as a little-endian integer and reduce modulo `L` to a 32-byte scalar.
7. Calculate the last digit blinding factor: `b[n-1] = f - ∑b[t] mod L`, where `t` is from 0 to `n-2`.
8. For `t` from `0` to `n-1` (each digit):
    1. Calculate `digit[t] = value & (0x03 << 2·t)` where `<<` denotes a bitwise left shift.
    2. Calculate `j[t] = digit[t] >> 2·t` where `>>` denotes a bitwise right shift.
    3. Define `base = 4`.
    4. For `i` from `0` to `base-1` (each digit’s value):
        1. Calculate point `P[t,i] = D[t] - i·(base^t)·H`.
9. [Recover Payload From Borromean Ring Signature](#recover-payload-from-borromean-ring-signature): compute an array of `2·N` 32-byte chunks `{ct[i]}` using the following inputs (halt and return `nil` if decryption fails):
    1. `msg`: the 32-byte string to be signed.
    2. `n=N/2`: number of rings.
    3. `m=base`: number of signatures in each ring.
    4. `{P[i,j]}`: `n·m` public keys, [points](data.md#public-key) on the elliptic curve.
    5. `{b[i]}`: the list of `n` blinding factors as private keys.
    6. `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[i,j] == b[i]·G`.
    7. `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.
10. Derive payload encryption key unique to this payload and the value: `pek = SHA3-256(0xec || rek || f || V)`.
11. [Decrypt payload](#decrypt-payload): compute an array of `2·N-1` 32-byte chunks: `{pt[i]} = DecryptPayload({ct[i]}, pek)`. If decryption fails, halt and return `nil`.
12. Return `{pt[i]}`, a plaintext array of `2·N-1` 32-byte elements.





### Create Excess Commitment

**Inputs:**

1. `q`: the [excess blinding factor](#excess-factor)

**Output:**

1. `(QG,QJ)`: the [point pair](#point-pair) representing an ElGamal commitment to `q` using [generators](#generators) `G` and `J`.
2. `(e,s)`: the Schnorr signature proving that `(QG,QJ)` does not affect asset amounts.

**Algorithm:**

1. Calculate a [point pair](#point-pair) `QG = q·G, QJ = q·J`.
2. Calculate Fiat-Shamir factor `h = ScalarHash("h" || G || J || QG || QJ)`.
3. Calculate the base point `B = h·G + J`.
4. Calculate the nonce `k = ScalarHash(h || q)`.
5. Calculate point `R = k·B`.
6. Calculate scalar `e = ScalarHash("e" || QG || QJ || R)`.
7. Calculate scalar `s = k + q·e mod L`.
8. Return `(s,e)`.


### Verify Excess Commitment

**Inputs:**

1. `(QG,QJ)`: the [point pair](#point-pair) representing an ElGamal commitment to secret blinding factor `q` using [generators](#generators) `G` and `J`.
2. `(e,s)`: the Schnorr signature proving that `(QG,QJ)` does not affect asset amounts.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. Calculate Fiat-Shamir factor `h = ScalarHash("h" || G || J || QG || QJ)`.
2. Calculate the base point `B = h·G + J`.
3. Calculate combined public key point `Q = h·QG + QJ`.
4. Calculate point `R = s·B - e·Q`.
5. Calculate scalar `e’ = ScalarHash("e" || QG || QJ || R)`.
6. Return `true` if `e’ == e`, otherwise return `false`.




### Verify Value Commitments Balance

**Inputs:**

1. The list of `n` input value commitments `{VC[i]}`.
2. The list of `m` output value commitments `{VC’[i]}`.
3. The list of `k` [excess commitments](#excess-commitment) `{(QC[i], s[i], e[i])}`.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. [Verify](#verify-excess-commitment) each of `k` [excess commitments](#excess-commitment); if any is not valid, halt and return `false`.
2. Calculate the sum of input value commitments: `Ti = ∑(VC[i], j from 0 to n-1)`.
3. Calculate the sum of output value commitments: `To = ∑(VC’[i], i from 0 to m-1)`.
4. Calculate the sum of excess commitments: `Tq = ∑(QC[i], i from 0 to k-1)`.
5. Return `true` if `Ti == To + Tq`, otherwise return `false`.



### Create Transient Issuance Key

**Inputs:**

1. `assetid`: asset ID for which the issuance key is being generated.
2. `aek`: [asset ID encryption key](#asset-id-encryption-key).

**Output:** `(y,Y)`: a pair of private and public keys.

**Algorithm:**

1. Calculate scalar `y = ScalarHash(0xa1 || assetid || aek)`.
2. Calculate point `Y` by multiplying base point by `y`: `Y = y·G`.
3. Return key pair `(y,Y)`.




### Create Issuance Asset Range Proof

When creating a confidential issuance, the first step is to construct the rest of the input commitment and input witness, including an asset issuance choice for each asset that one wants to include in the anonymity set. The issuance key for each asset should be extracted from the issuance programs. (Issuance programs that support confidential issuance should have a branch that uses `CHECKISSUANCE` to check for a confidential issuance key.)

**Inputs:**

1. `H`: the [asset ID commitment](#asset-id-commitment).
2. `c`: the [blinding factor](#asset-id-blinding-factor) for commitment `H` such that: `H = A + c·G`.
3. `{a[i]}`: `n` 32-byte unencrypted [asset IDs](data.md#asset-id).
4. `{Y[i]}`: `n` issuance keys (each a 32-byte [public key](data.md#public-key).
5. `vmver`: VM version for the issuance signature program.
6. `program`: issuance signature program.
7. `j`: the index of the asset being issued.
8. `y`: the private key for the issuance key corresponding to the asset being issued: `Y[j] = y·G`.

**Output:** an [issuance asset range proof](#issuance-asset-range-proof) consisting of:

* `e[0], s[0], ... s[n-1]`: the issuance ring signature,
* `{Y[i]}`: `n` issuance keys,
* `vmver`: VM version for the issuance signature program,
* `program`: issuance signature program,
* `[]`: empty list of program arguments (to be filled in by the issuer).

**Algorithm:**

1. Calculate nonblinded asset commitments for the values in `a`: `A[i] = 8·Decode(SHA3(a[i]))`.
2. Calculate a 96-byte commitment string: `commit = StreamHash(0x66 || H || A[0] || ... || A[n-1] || Y[0] || ... || Y[n-1] || vmver || program, 8·96)`, where `vmver` is encoded as a 64-bit unsigned little-endian integer.
3. Calculate message to sign as first 32 bytes of the commitment string: `msg = commit[0:32]`.
4. Calculate the coefficient `h` from the remaining 64 bytes of the commitment string: `h = commit[32:96]`. Interpret `h` as a 64-byte little-endian integer and reduce modulo subgroup order `L`.
5. Calculate `n` public keys `{P[i]}`: `P[i] = H - A[i] + h·Y[i]`.
6. Calculate private key `p = c + h·y`.
7. [Create a ring signature](#verify-ring-signature) with:
    * message `msg`,
    * `n` public keys `{P[i]}`,
    * index `j`,
    * private key `p`.
8. Return an issuance range proof consisting of `(e0,{s[i]}, {Y[i]}, vmver, program, [])`.




### Verify Issuance Asset Range Proof

**Inputs:**

1. `IARP`: the to-be-verified [issuance asset range proof](#issuance-asset-range-proof) consisting of:
    * `e[0], s[0], ... s[n-1]`: the issuance ring signature,
    * `{Y[i]}`: `n` issuance keys,
    * `vmver`: VM version for the issuance signature program,
    * `program`: issuance signature program,
2. `H`: the [asset ID commitment](#asset-id-commitment).
3. `{a[i]}`: `n` 32-byte unencrypted [asset IDs](data.md#asset-id).

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. Calculate nonblinded asset commitments for the values in `a`: `A[i] = 8·Decode(SHA3(a[i]))`.
2. Calculate a 96-byte commitment string: `commit = StreamHash(0x66 || H || A[0] || ... || A[n-1] || Y[0] || ... || Y[n-1] || vmver || program, 96)`, where `vmver` is encoded as a 64-bit unsigned little-endian integer.
3. Calculate message to sign as first 32 bytes of the commitment string: `msg = commit[0:32]`.
4. Calculate the coefficient `h` from the remaining 64 bytes of the commitment string: `h = commit[32:96]`. Interpret `h` as a 64-byte little-endian integer and reduce modulo subgroup order `L`.
5. Calculate the `n` public keys `{P[i]}`: `P[i] = H - A[i] + h·Y[i]`.
6. [Verify the ring signature](#verify-ring-signature) `e[0], s[0], ... s[n-1]` with message `msg` and public keys `{P[i]}`.





## High-level procedures

### Verify Output

**Inputs:**

1. `AD`: the [asset ID descriptor](#asset-id-descriptor).
2. `VD`: the [value commitment](#value-descriptor).
3. `ARP`: the [asset range proof](#asset-range-proof) or an empty string.
4. `VRP`: the [value range proof](#value-range-proof) or an empty string.

**Output:** `true` if verification succeeded, `false` otherwise.

**Algorithm:**

1. If `ARP` is not empty and `AD` is blinded:
    1. [Verify asset range proof](#verify-asset-range-proof) using `(AD.H,AD.(ea,ec),ARP)`. If verification failed, halt and return `false`.
2. If `VRP` is not empty and `VD` is blinded:
    1. [Verify value range proof](#verify-value-range-proof) using `(AD.H,VD.V,VD.(ev,ef),VRP)`. If verification failed, halt and return `false`.
3. Return `true`.


### Verify Issuance

**Inputs:**

1. `AD`: the [asset ID descriptor](#asset-id-descriptor).
2. `VD`: the [value descriptor](#value-descriptor).
3. `{a[i]}`: `n` 32-byte unencrypted [asset IDs](data.md#asset-id).
4. `IARP`: the [issuance asset ID range proof](#issuance-asset-range-proof).
5. `VRP`: the [value range proof](#value-range-proof).

**Output:** `true` if verification succeeded, `false` otherwise.

**Algorithm:**

1. If `IARP` is not empty and `AD` is blinded:
    1. [Verify issuance asset range proof](#verify-issuance-asset-range-proof) using `(IARP,AD.H,{a[i]})`. If verification failed, halt and return `false`.
2. If `VRP` is not empty and `VD` is blinded:
    1. [Verify value range proof](#verify-value-range-proof) using `(AD.H, VD.V, evef=(0x00...,0x00...),VRP)`. If verification failed, halt and return `false`.
3. Return `true`.




### Verify Confidential Assets

**Inputs:**

1. List of issuances, each input consisting of:
    * `AD`: the [asset ID descriptor](#asset-id-descriptor).
    * `VD`: the [value descriptor](#value-descriptor).
    * `{a[i]}`: `n` 32-byte unencrypted [asset IDs](data.md#asset-id).
    * `IARP`: the [issuance asset ID range proof](#issuance-asset-range-proof).
    * `VRP`: the [value range proof](#value-range-proof).
2. List of inputs, each input consisting of:
    * `AD`: the [asset ID descriptor](#asset-id-descriptor).
    * `VD`: the [value descriptor](#value-descriptor).
3. List of outputs, each output consisting of:
    * `AD`: the [asset ID descriptor](#asset-id-descriptor).
    * `VD`: the [value descriptor](#value-descriptor).
    * `ARP`: the [asset range proof](#asset-range-proof) or empty string.
    * `VRP`: the [value range proof](#value-range-proof) or empty string.
4. The list of [excess commitments](#excess-commitment): `{(Q[i], s[i], e[i])}`.

**Output:** `true` if verification succeeded, `false` otherwise.

**Algorithm:**

1. [Verify each issuance](#verify-issuance). If verification failed, halt and return `false`.
2. For each output:
    1. If `AD` is blinded and `ARP` is an empty string, or an ARP with zero keys, verify that `AD.H` equals one of the asset ID commitments in the inputs or issuances. If not, halt and return `false`.
    2. If `AD` is blinded and `ARP` is not empty, verify that each asset ID commitment in the `ARP` belongs to the set of the asset ID commitments on the inputs and issuances. If not, halt and return `false`.
    3. If there are more than one output and the output’s value descriptor is blinded:
        1. Verify that the value range proof is not empty. Otherwise, halt and return `false`.
    4. [Verify output](#verify-output). If verification failed, halt and return `false`.
3. [Verify value commitments balance](#verify-value-commitments-balance) using a union of issuance and input value commitments as input commitments. If verification failed, halt and return `false`.
4. Return `true`.




### Encrypt Issuance

**Inputs:**

1. `rek`: the [record encryption key](#record-encryption-key) unique to this issuance.
2. `assetID`: the output asset ID.
3. `value`: the output amount.
4. `N`: number of bits to encrypt (`value` must fit within `N` bits).
5. `{(assetIDs[i], Y[i])}`: `n` input asset IDs and corresponding issuance public keys.
6. `y`: issuance key for `assetID` such that `Y[j] = y·G` where `j` is the index of the issued asset: `assetIDs[j] == assetID`.
7. `(vmver’,program’)`: the signature program and its VM version to be signed by the issuance proof.

**Outputs:**

1. `AD`: the [asset ID descriptor](#asset-id-descriptor).
2. `VD`: the [value descriptor](#value-descriptor).
3. `IARP`: the [issuance asset ID range proof](#issuance-asset-range-proof).
4. `VRP`: the [value range proof](#value-range-proof).
5. `c`: the [asset ID blinding factor](#asset-id-blinding-factor) for the asset ID commitment `AD.H`.
6. `f`: the [value blinding factor](#value-blinding-factor).

In case of failure, returns `nil` instead of the items listed above.

**Algorithm:**

1. [Derive asset encryption key](#asset-id-encryption-key) `aek` from `rek`.
2. [Derive value encryption key](#value-encryption-key) `vek` from `rek`.
3. [Create nonblinded asset ID commitment](#create-nonblinded-asset-id-commitment) for all values in `{assetIDs[i]}`: `A[i] = 8·Decode(SHA3(assetIDs[i]))`.
4. Find `j` index of the `assetID` among `{assetIDs[i]}`. If not found, halt and return `nil`.
5. [Create blinded asset ID commitment](#create-blinded-asset-id-commitment): compute `(H,c)` from `(A, 0, aek)`.
6. [Create blinded value commitment](#create-blinded-value-commitment): compute `(V,f)` from `(vek, value, H, c)`.
7. [Create issuance asset range proof](#create-issuance-asset-range-proof): compute `IARP` from `(H, c, {A[i]}, {Y[i]}, vmver’, program’, j, y)`.
8. [Create Value Range Proof](#create-value-range-proof): compute `VRP` from `(H, V, (0x00...,0x00...), N, value, {0x00...}, f, rek)`.
9. Create [blinded asset ID descriptor](#blinded-asset-id-descriptor) `AD` containing `H` and all-zero [encrypted asset ID](#encrypted-asset-id).
10. Create [blinded value descriptor](#blinded-value-descriptor) `VD` containing `V` and all-zero [encrypted value](#encrypted-value).
11. Return `(AD, VD, IARP, VRP, c, f)`.




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
5. `plaintext`: binary string that has length of less than `32·(2·N-1)` bytes when encoded as [varstring31](data.md#varstring31).
6. Optional `q`: the [excess factor](#excess-factor) to have this output balance with the transaction. If omitted, blinding factor is generated at random.

**Outputs:**

1. `AD`: the [asset ID descriptor](#asset-id-descriptor).
2. `VD`: the [value descriptor](#value-descriptor).
3. `VRP`: the [value range proof](#value-range-proof).
4. `c’`: the output [asset ID blinding factor](#asset-id-blinding-factor) for the asset ID commitment `H’`.
5. `f’`: the output [value blinding factor](#value-blinding-factor).

In case of failure, returns `nil` instead of the items listed above.

**Algorithm:**

1. Encode `plaintext` using [varstring31](data.md#varstring31) encoding and split the string in 32-byte chunks `{pt[i]}` (last chunk padded with zero bytes if needed).
2. If the number of chunks `{pt[i]}` exceeds `2·N-1`, halt and return `nil`.
3. If the number of chunks `{pt[i]}` is less than `2·N-1`, pad the array with all-zero 32-byte chunks.
4. If `value ≥ 2^N`, halt and return `nil`.
5. [Derive asset encryption key](#asset-id-encryption-key) `aek` from `rek`.
6. [Derive value encryption key](#value-encryption-key) `vek` from `rek`.
7. [Create blinded asset ID commitment](#create-blinded-asset-id-commitment): compute `(H’,c’)` from `(assetID, aek)`.
8. [Encrypt asset ID](#encrypt-asset-id): compute `(ea,ec)` from `(assetID, H’, c’, aek)`.
9. [Create blinded value commitment](#create-blinded-value-commitment): compute `(V’,f’)` from `(vek, value, H’)`.
10. If `q` is provided:
    1. Compute `extra` scalar: `extra = q - f’ - value·c’`.
    2. Add `extra` to the value blinding factor: `f’ = f’ + extra`.
    3. Adjust the value commitment too: `V = V + extra·G`.
    4. Note: as a result, the total blinding factor of the output will be equal to `q`.
11. [Encrypt Value](#encrypt-value): compute `(ev,ef)` from `(V’, value, f’, vek)`.
12. [Create Value Range Proof](#create-value-range-proof): compute `VRP` from `(H’, V’, (ev,ef), N, value, {pt[i]}, f’, rek)`.
13. Create [encrypted asset ID descriptor](#encrypted-asset-id-descriptor) `AD` containing `H’` and `(ea,ec)`.
14. Create [encrypted value descriptor](#encrypted-value-descriptor) `VD` containing `V’` and `(ev,ef)`.
15. Return `(AD, VD, VRP, c’, f’)`.



### Decrypt Output

This algorithm decrypts fully encrypted amount and asset ID for a given output.

**Inputs:**

1. `rek`: the [record encryption key](#record-encryption-key).
2. `AD`: the [asset ID descriptor](#asset-id-descriptor).
3. `VD`: the [value descriptor](#value-descriptor).
4. `VRP`: the [value range proof](#value-range-proof) or an empty string.

**Outputs:**

1. `assetID`: the output asset ID.
2. `value`: the output amount.
3. `c`: the output [asset ID blinding factor](#asset-id-blinding-factor) for the asset ID commitment `H`.
4. `f`: the output [value blinding factor](#value-blinding-factor).
5. `plaintext`: the binary string that has length of less than `32·(2·N-1)` bytes when encoded as [varstring31](data.md#varstring31).

In case of failure, returns `nil` instead of the items listed above.

**Algorithm:**

1. [Derive asset encryption key](#asset-id-encryption-key) `aek` from `rek`.
2. [Derive value encryption key](#value-encryption-key) `vek` from `rek`.
3. Decrypt asset ID:
    1. If `AD` is [nonblinded](#nonblinded-asset-id-descriptor): set `assetID` to the one stored in `AD`, set `c` to zero.
    2. If `AD` is [blinded and not encrypted](#blinded-asset-id-descriptor), halt and return nil.
    3. If `AD` is [encrypted](#encrypted-asset-id-descriptor), [Decrypt Asset ID](#decrypt-asset-id): compute `(assetID,c)` from `(H,(ea,ec),aek)`. If verification failed, halt and return `nil`.
4. Decrypt value:
    1. If `VD` is [nonblinded](#nonblinded-value-descriptor): set `value` to the one stored in `VD`, set `f` to zero.
    2. If `VD` is [blinded and not encrypted](#blinded-value-descriptor), halt and return nil.
    3. If `VD` is [encrypted](#encrypted-value-descriptor), [Decrypt Value](#decrypt-value): compute `(value, f)` from `(H,V,(ev,ef),vek)`. If verification failed, halt and return `nil`.
5. If value range proof `VRP` is not empty:
    1. [Recover payload from Value Range Proof](#recover-payload-from-value-range-proof): compute a list of 32-byte chunks `{pt[i]}` from `(H,V,(ev,ef),VRP,value,f,rek)`. If verification failed, halt and return `nil`.
    2. Flatten the array `{pt[i]}` in a binary string and decode it using [varstring31](data.md#varstring31) encoding. If decoding fails, halt and return `nil`.
6. If value range proof `VRP` is empty, set `plaintext` to an empty string.
7. Return `(assetID, value, c, f, plaintext)`.





## Integration

This section provides an overview for protocol changes in the VM1 and blockchain validation logic necessary to support Confidential Assets.

### VM1

#### 1. New behavior for existing opcodes

* [CHECKOUTPUT](vm1.md#checkoutput): accepts commitments (encoded as 32-byte public keys) for both asset ID and amount arguments. Fails execution if commitment is provided when the corresponding field is nonblinded. Fails execution if raw value is provided, but the field is blinded.
* [ASSET](vm1.md#asset): fails execution if asset ID is blinded.
* [AMOUNT](vm1.md#amount): fails execution if amount is blinded.

#### 2. New introspection opcodes

* [ASSETCOMMITMENT](vm1.md#assetcommitment): returns asset ID commitment, if asset ID is blinded. Fails execution otherwise.
* [VALUECOMMITMENT](vm1.md#valuecommitment): returns value commitment, if amount is blinded. Fails execution otherwise.
* [ISSUANCEKEY](vm1.md#issuancekey): returns public key declared in the [issuance choice context](vm1.md#issuance-choice-context).


### Blockchain data structures

#### 1. Issuance input

1. TBD
2. TBD

#### 2. Output

1. TBD
2. TBD

#### 3. Common fields

1. List of excess commitments (point and a signature).

### Blockchain validation

1. Verify block version = 2.
2. Verify tx version = 2.
3. Verify tx balances.
4. Verify range proofs.
5. Verify issuance range proof.


## Test vectors

TBD: Hash256, StreamHash, ScalarHash.

TBD: AC, VC, excess commits.

TBD: RS, BRS.

TBD: ARP, VRP.

