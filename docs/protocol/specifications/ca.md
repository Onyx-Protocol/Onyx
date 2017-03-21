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
  * [Issuance Asset Range Proof](#issuance-asset-range-proof)
  * [Value Commitment](#value-commitment)
  * [Excess Factor](#excess-factor)
  * [Excess Commitment](#excess-commitment)
  * [Value Proof](#value-proof)
  * [Value Range Proof](#value-range-proof)
  * [Extended Key Pair](#extended-key-pair)
  * [Record Encryption Key](#record-encryption-key)
  * [Intermediate Encryption Key](#intermediate-encryption-key)
  * [Asset ID Encryption Key](#asset-id-encryption-key)
  * [Value Encryption Key](#value-encryption-key)
  * [Asset ID Blinding Factor](#asset-id-blinding-factor)
  * [Value Blinding Factor](#value-blinding-factor)
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
  * [Create Issuance Proof](#create-issuance-proof)
  * [Verify Issuance Proof](#verify-issuance-proof)
* [High-level procedures](#high-level-procedures)
  * [Verify Output](#verify-output)
  * [Verify Issuance](#verify-issuance)
  * [Verify Confidential Assets](#verify-confidential-assets)
  * [Encrypt Issuance](#encrypt-issuance)
  * [Encrypt Output](#encrypt-output)
  * [Decrypt Output](#decrypt-output)
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
        * `AC=(A,O)`, [nonblinded asset ID commitment](#nonblinded-asset-id-commitment)
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


## Definitions

### Elliptic Curve Parameters

**The elliptic curve** is edwards25519 as defined by [[RFC7748](https://tools.ietf.org/html/rfc7748)].

`L` is the **order of edwards25519** as defined by \[[RFC8032](https://tools.ietf.org/html/rfc8032)\] (i.e. 2<sup>252</sup>+27742317777372353535851937790883648493).


### Zero point

_Zero point_ `O` is a representation of the _point at infinity_, identity element in the [edwards25519](#elliptic-curve-parameters) subgroup. It is encoded as follows:

    O = 0x0100000000000000000000000000000000000000000000000000000000000000


### Generators

**Primary generator point** (`G`) is the elliptic curve point specified as "B" in Section 5.1 of [[RFC8032](https://tools.ietf.org/html/rfc8032)].

Generator `G` has the following 32-byte encoding:

    G = 0x5866666666666666666666666666666666666666666666666666666666666666

**Secondary generator point** (`J`) is the elliptic curve point defined as decoded hash of the primary generator `G`:

    J = 8·Decode(SHA3-256(Encode(G)))

Generator `J` has the following 32-byte encoding:

    J = 0x00c774b875ed4e395ebb0782b4d93db838d3c4c0840bc970570517555ca71b77

**Tertiary generator points** (`G[i]`) are 31 elliptic curve points defined as decoded hash of the primary generator `G`:

    G[i] = 8·Decode(SHA3-256(byte(i) || Encode(G) || int64le(cnt)))

Counter `cnt` is chosen to be the smallest positive integer starting with 0 that yields a valid edwards25519 point for a given index `i`.

    G[0]  = 0xe68528ab16b201331fc980c33eef08f7d114554715d370a2c614182ef296dab3   cnt = 0
    G[1]  = 0x32011e4f5c29bbc20d5c96500e87e2303a004687895b2d6d944ff687d0dbefad   cnt = 3
    G[2]  = 0x0d688b311df06d633ced925c1561bea9608f305781c1ab32c55944628181cd1e   cnt = 0
    G[3]  = 0xe17522742ed8bd11aa5d1f2e341400eb1c6f85b47c46817ea0e90b5d5510b420   cnt = 1
    G[4]  = 0x67454d0f02d3962508b89d4209996943825dbf261e7e6e07a842d45b33b2baad   cnt = 0
    G[5]  = 0xc7f0c5eebcb5f37194b7ab96af66e79e0aa37a6cdbde5fbd6af13637b6f05cab   cnt = 2
    G[6]  = 0xc572d7c6f3ef692efbd13928dad208c4572ffe371d88f70a763af3a11cac8709   cnt = 2
    G[7]  = 0xe450cc93f07e0c18a79c1f0572a6971da37bfa81c6003835acf68a8afc1ca33b   cnt = 1
    G[8]  = 0x409ae3e34c0ff3929bceaf7b934923809b461038a1d31c7a0928c8c7ab707604   cnt = 5
    G[9]  = 0xc43d0400219b6745b95ff81176dfbbd5d33b9cc869e171411fff96656273b96c   cnt = 3
    G[10] = 0xd1eeee54b75cc277bf8a6454accce6086ab24750b0d58a11fb7cad35eba42ff6   cnt = 4
    G[11] = 0x2446b2efa69fb26a4268037909c466c9b5083bfecf3c2ab3a420114a6f91f0eb   cnt = 1
    G[12] = 0xd0c4ee744ac129d0282a1554ca7a339e3d9db740826d365eefe984c0e5023969   cnt = 0
    G[13] = 0xe1d621717a876830e0c7c1bf8e7e674cf5cbe3aa1e71885d7d347854277aa6ca   cnt = 0
    G[14] = 0x6e95425b9481a70aa553f1e7d30b8182ef659f94ec4e446bc61058615460cbcc   cnt = 2
    G[15] = 0x4200e80a3976d66f392c7998aa889e6a9efdc65abb6d39575ee8fd8b295008ad   cnt = 3
    G[16] = 0x3e3e626d2c051c82de750847ced96e1f6af5f4a706703512914c0e334c3cf76e   cnt = 6
    G[17] = 0xb98d0b73da8ae83754bc61c550c2c6ad76f78ba66e601c3876aea38e086552ae   cnt = 1
    G[18] = 0x90128059cb3b5baa3b1230e2ef211257253d477490e182bcb60c89bae43752fe   cnt = 0
    G[19] = 0xb04be209278413859ad51cf6d4a7f15bc2dea9f71c34f71945469705c3885b27   cnt = 0
    G[20] = 0xfda85012a00938e6f12f4da3cb1642cd1963295d3b089dcb0ee81e73e1b14050   cnt = 1
    G[21] = 0x73f1392e664fa1687983fcb1c7397b89876f6da8357ee8b07cb44534bc160644   cnt = 0
    G[22] = 0x0f347deffff466dec1af40197d39e97933112af29d6f305734dc7a4c6e2aceaf   cnt = 0
    G[23] = 0xc9c779f2644195546a17991a455a6d16a446305f80605e8466f5cd0861a6cb48   cnt = 0
    G[24] = 0x56614c7cbd1f4b27100d84bd76b4e472237e09ad0970745da252ef0b197291b1   cnt = 4
    G[25] = 0x4b266eaac77da3229fd884b4fc8163d8fae10a914334805a80b93da1ea8cb7ab   cnt = 0
    G[26] = 0xe1b33961996a81b591fd54b72b67fe23c3bfac82223713865a39e9802c8a393e   cnt = 0
    G[27] = 0xf1a19594ea8a6caa753c03d3e63a545ad8dc5ee331647bfeb7a9ac5b21cc04d8   cnt = 1
    G[28] = 0x60f79007f42376ed140fe7efd43218106613546d8cb3bd06a5cef2e73b02fad7   cnt = 0
    G[29] = 0xe9cb7b6fd3bb865dac6cff479bc2e3ce98ab95e4a6a57d81ae6d6cb032375f4a   cnt = 2
    G[30] = 0x7ee2183153687344e093278bc692c4915761ada87a51a778b605e88078d9902a   cnt = 1


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

See:

* [Create Ring Signature](#create-ring-signature)
* [Verify Ring Signature](#verify-ring-signature)


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

See:

* [Create Borromean Ring Signature](#create-borromean-ring-signature)
* [Verify Borromean Ring Signature](#verify-borromean-ring-signature)


### Asset ID Point

_Asset ID point_ is a [point](#point) representing an asset ID. It is defined as follows:

1. Let `counter = 0`.
2. Calculate `Hash256("AssetID" || assetID || uint64le(counter))` where `counter` is encoded as a 64-bit unsigned integer using little-endian convention.
3. Decode the resulting hash as a [point](#point) `P` on the elliptic curve.
4. If the point is invalid, increment `counter` and go back to step 2. This will happen on average for half of the asset IDs.
5. Calculate point `A = 8·P` (8 is a cofactor in edwards25519) which belongs to a subgroup [order](#elliptic-curve-parameters) `L`.
6. Return `A`.


### Asset ID Commitment

An asset ID commitment `AC` is an ElGamal commitment represented by a [point pair](#point-pair):

    AC = (H, Ba)
    
    H  = A + c·G
    Ba = c·J
   
where:      

* `A` is an [Asset ID Point](#asset-id-point), an orthogonal point representing an asset ID.  
* `c` is a blinding [scalar](#scalar) for the asset ID.
* `G`, `J` are [generator points](#generators).

The asset ID commitment can either be nonblinded or blinded:

* [Create Nonblinded Asset ID Commitment](#create-nonblinded-asset-id-commitment)
* [Create Blinded Asset ID Commitment](#create-blinded-asset-id-commitment)


### Asset Range Proof

The asset range proof demonstrates that a given [asset ID commitment](#asset-id-commitment) commits to one of the asset IDs specified in the transaction inputs. A [whole-transaction validation procedure](#verify-confidential-assets) makes sure that all of the declared asset ID commitments in fact belong to the transaction inputs.

Asset range proof can be [non-confidential](#non-confidential-asset-range-proof) or [confidential](#confidential-asset-range-proof).

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

See:

* [Create Asset Range Proof](#create-asset-range-proof)
* [Verify Asset Range Proof](#verify-asset-range-proof)



### Issuance Asset Range Proof

The issuance asset range proof demonstrates that a given [confidential issuance](#confidential-issuance) commits to one of the asset IDs specified in the transaction inputs. It contains a ring signature. The other inputs to the [verification procedure](#verify-issuance-asset-range-proof) are computed from other elements in the confidential issuance witness, as part of the [validation procedure](#validate-transaction-input).

The size of the ring signature (`n+1` 32-byte elements) and the number of issuance keys (`n`) are derived from `n` [asset issuance choices](blockchain.md#asset-issuance-choice) specified outside the range proof.

The proof also contains a _tracing point_ that that lets any issuer to prove or disprove whether the issuance is performed by their issuance key.

#### Non-Confidential Issuance Asset Range Proof

Field                        | Type      | Description
-----------------------------|-----------|------------------
Type                         | byte      | Contains value 0x00 to indicate the commitment is not blinded.
Asset ID                     | [AssetID](blockchain.md#asset-id)   | 32-byte asset identifier.

#### Confidential Issuance Asset Range Proof

Field                           | Type             | Description
--------------------------------|------------------|------------------
Type                            | byte             | Contains value 0x01 to indicate the commitment is blinded.
Issuance Keys                   | [List](blockchain.md#list)\<[Point](#point)\> | Keys to be used to calculate the public key for the corresponding index in the ring signature.
Tracing Point                   | [Point](#point)  | A point that lets any issuer to prove or disprove if this issuance is done by them.
Blinded Marker Point            | [Point](#point)  | A blinding factor commitment using a marker point (used together with the tracing point).
Marker Signature                | 64 bytes         | A pair of [scalars](#scalar) representing a single Schnorr signature for the marker and tracing points.
Issuance Ring Signature         | [Ring Signature](#ring-signature)   | A ring signature proving that the issuer of an encrypted asset ID approved the issuance.

TBD: modify the confidential proof to allow "watch keys" (will require change of `{Y}` to `{(Y,W)}` and a pair of tracing points).

See:

* [Create Issuance Asset Range Proof](#create-issuance-asset-range-proof)
* [Verify Issuance Asset Range Proof](#verify-issuance-asset-range-proof)




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

Value range proof can be [non-confidential](#non-confidential-value-range-proof) or [confidential](#confidential-value-range-proof).

#### Non-Confidential Value Range Proof

A non-confidential range proof demonstrates the non-encrypted amount and allows efficient verification that a given [value commitment](#value-commitment) commits to that amount.

Field                        | Type      | Description
-----------------------------|-----------|------------------
Type                         | byte      | Contains value 0x00 to indicate the commitment is not blinded.
Amount                       | varint63  | Amount

#### Confidential Value Range Proof

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

See:

* [Create Value Range Proof](#create-value-range-proof)
* [Verify Value Range Proof](#verify-value-range-proof)


### Extended Key Pair

*Extended key pair* (EKP) is a pair of [ChainKD](chainkd.md) extended public keys (xpubs):

    {xpub1, xpub2}

EKP is encoded as a 128-byte concatenation of the corresponding extended public keys (64 bytes each):

    EKP = xpub1 || xpub2

EKPs allows “two-dimensional” key derivation by deriving from one key and leaving the other one unchanged, therefore allowing “vertical” derivation from [Record Encryption Keys](#record-encryption-keys) to [Intermediate Encryption Keys](#intermediate-encryption-keys), [Asset ID Encryption Keys](#asset-id-encryption-keys) and [Value Encryption Keys](#value-encryption-keys) and “horizontal” derivation from a root key pair associated with a user’s account to per-transaction and per-output key pairs.


### Record Encryption Key

Record encryption key (REK or `rek`) is an [extended key pair](#extended-key-pair):

    REK = {xpub1, xpub2}

It is used to decrypt the payload data from the [value range proof](#value-range-proof), and derive [asset ID encryption key](#asset-id-encryption-key) and [value encryption key](#value-encryption-key).

The first `xpub1` is used to derive more specific keys as described below that all share the same second key `xpub2`.
The second `xpub2` is used to derive the entire hierarchies of encryption keys, so that a [REK](#record-encryption-key), or [IEK](#intermediate-encryption-key) could be shared for the entire account instead of per-transaction.


### Intermediate Encryption Key

Intermediate encryption key (IEK or `iek`) is an [extended key pair](#extended-key-pair) that allows decrypting the asset ID and the value in the output commitment. It is derived from the [record encryption key](#record-encryption-key) as follows:

    IEK = {ND(REK.xpub1, "IEK"), REK.xpub2}

where `ND` is non-hardened derivation as defined by [ChainKD](chainkd.md#derive-non-hardened-extended-public-key).


### Asset ID Encryption Key

Asset ID encryption key (AEK or `aek`) is an [extended key pair](#extended-key-pair) that allows decrypting the asset ID in the output commitment. It is derived from the [intermediate encryption key](#intermediate-encryption-key) as follows:

    AEK = {ND(IEK.xpub1, "AEK"), IEK.xpub2}

where `ND` is non-hardened derivation as defined by [ChainKD](chainkd.md#derive-non-hardened-extended-public-key).


### Value Encryption Key

Value encryption key (VEK or `vek`) is an [extended key pair](#extended-key-pair) that allows decrypting the amount in the output commitment. It is derived from the [intermediate encryption key](#intermediate-encryption-key) as follows:

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


















## Core algorithms

### Create Ring Signature

**Inputs:**

1. `msg`: the string to be signed.
2. `B`: base [point](#point) to verify the signature (not necessarily a [generator](#generators) point).
3. `{P[i]}`: `n` [points](#point) representing the public keys.
4. `j`: the index of the designated public key, so that `P[j] == p·B`.
5. `p`: the secret [scalar](#scalar) representing a private key for the public key `P[j]`.

**Output:** `{e0, s[0], ..., s[n-1]}`: the ring signature, `n+1` 32-byte elements.

**Algorithm:**

1. Let `counter = 0`.
2. Let the `msghash` be a hash of the input non-secret data: `msghash = Hash256(B || P[0] || ... || P[n-1] || msg)`.
3. Calculate a sequence of: `n-1` 32-byte random values, 64-byte `nonce` and 1-byte `mask`: `{r[i], nonce, mask} = StreamHash(uint64le(counter) || msghash || p || uint64le(j), 32·(n-1) + 64 + 1)`, where:
    * `counter` is encoded as a 64-bit little-endian integer,
    * `p` is encoded as a 256-bit little-endian integer,
    * `j` is encoded as a 64-bit little-endian integer.
4. Calculate `k = nonce mod L`, where `nonce` is interpreted as a 64-byte little-endian integer and reduced modulo subgroup order `L`.
5. Calculate the initial e-value, let `i = j+1 mod n`:
    1. Calculate `R[i]` as the [point](#point) `k·B`.
    2. Define `w[j]` as `mask` with lower 4 bits set to zero: `w[j] = mask & 0xf0`.
    3. Calculate `e[i] = ScalarHash(R[i] || msghash || uint64le(i) || w[j])` where `i` is encoded as a 64-bit little-endian integer.
6. For `step` from `1` to `n-1` (these steps are skipped if `n` equals 1):
    1. Let `i = (j + step) mod n`.
    2. Calculate the forged s-value `s[i] = r[step-1]`.
    3. Define `z[i]` as `s[i]` with the most significant 4 bits set to zero.
    4. Define `w[i]` as a most significant byte of `s[i]` with lower 4 bits set to zero: `w[i] = s[i][31] & 0xf0`.
    5. Let `i’ = i+1 mod n`.
    6. Calculate point `R[i’] = z[i]·B - e[i]·P[i]`.
    7. Calculate `e[i’] = ScalarHash(R[i’] || msghash || uint64le(i’) || w[i])` where `i’` is encoded as a 64-bit little-endian integer.
7. Calculate the non-forged `z[j] = k + p·e[j] mod L` and encode it as a 32-byte little-endian integer.
8. If `z[j]` is greater than 2<sup>252</sup>–1, then increment the `counter` and try again from the beginning. The chance of this happening is below 1 in 2<sup>124</sup>.
9. Define `s[j]` as `z[j]` with 4 high bits set to high 4 bits of the `mask`.
10. Return the ring signature `{e[0], s[0], ..., s[n-1]}`, total `n+1` 32-byte elements.


### Verify Ring Signature

**Inputs:**

1. `msg`: the string being signed.
2. `B`: base [point](#point) to verify the signature (not necessarily a [generator](#generators) point).
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
4. `{B[i]}`: `n` base [points](#point) to verify the signature (not necessarily [generator](#generators) points).
5. `{P[i,j]}`: `n·m` [points](#point) representing public keys.
6. `{p[i]}`: the list of `n` [scalars](#scalar) representing private keys.
7. `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[i,j] == p[i]·B[i]`.
8. `{payload[i]}`: sequence of `n·m` random 32-byte elements.

**Output:** `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.

**Algorithm:**

1. Let the `msghash` be a hash of the input non-secret data: `msghash = Hash256(uint64le(n) || uint64le(m) || {B[i]} || {P[i,j]} || msg)` where `n` and `m` are encoded as 64-bit little-endian integers.
2. Let `counter = 0`.
3. Let `cnt` byte contain lower 4 bits of `counter`: `cnt = counter & 0x0f`.
4. Calculate a sequence of `n·m` 32-byte random overlay values: `{o[i]} = StreamHash(uint64le(counter) || msghash || {p[i]} || {uint64le(j[i])}, 32·n·m)`, where:
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
        3. Calculate `e[t,j’] = ScalarHash(byte(cnt), R[t, j’] || msghash || uint64le(t) || uint64le(j’) || w[t,j])` where `t` and `j’` are encoded as 64-bit little-endian integers.
    7. If `j ≠ m-1`, then for `i` from `j+1` to `m-1`:
        1. Calculate the forged s-value: `s[t,i] = r[m·t + i]`.
        2. Define `z[t,i]` as `s[t,i]` with 4 most significant bits set to zero.
        3. Define `w[t,i]` as a most significant byte of `s[t,i]` with lower 4 bits set to zero: `w[t,i] = s[t,i][31] & 0xf0`.
        4. Let `i’ = i+1 mod m`.
        5. Calculate point `R[t,i’] = z[t,i]·B[t] - e[t,i]·P[t,i]`.
        6. Calculate `e[t,i’] = ScalarHash(byte(cnt), R[t,i’] || msghash || uint64le(t) || uint64le(i’) || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
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
        6. Calculate `e[t,i’] = ScalarHash(byte(cnt), R[t,i’] || msghash || uint64le(t) || uint64le(i’) || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
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
4. `{B[i]}`: `n` base [points](#point) to verify the signature (not necessarily [generator](#generators) points).
5. `{P[i,j]}`: `n·m` public keys, [points](#point) on the elliptic curve.
6. `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. Let the `msghash` be a hash of the input non-secret data: `msghash = SHA3-256(uint64le(n) || uint64le(m) || {B[i]} || {P[i,j]} || msg)` where `n` and `m` are encoded as 64-bit little-endian integers.
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
        5. Calculate `e[t,i’] = ScalarHash(byte(cnt) || R[t,i’] || msghash || uint64le(t) || uint64le(i’) || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
    3. Append `e[t,0]` to `E`: `E = E || e[t,0]`, where `e[t,0]` is encoded as a 32-byte little-endian integer.
6. Calculate `e’ = ScalarHash(E)`.
7. Return `true` if `e’` equals to `e0`. Otherwise, return `false`.



### Recover Payload From Borromean Ring Signature

**Inputs:**

1. `msg`: the string to be signed.
2. `n`: number of rings.
3. `m`: number of signatures in each ring.
4. `{B[i]}`: `n` base [points](#point) to verify the signature (not necessarily [generator](#generators) points).
5. `{P[i,j]}`: `n·m` public keys, [points](#point) on the elliptic curve.
6. `{p[i]}`: the list of `n` scalars representing private keys.
7. `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[i,j] == p[i]·G`.
8. `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.

**Output:** `{payload[i]}` list of `n·m` random 32-byte elements or `nil` if signature verification failed.

**Algorithm:**

1. Let the `msghash` be a hash of the input non-secret data: `msghash = SHA3-256(n || m || {B[i]} || {P[i,j]} || msg)` where `n` and `m` are encoded as 64-bit little-endian integers.
2. Define `E` to be an empty binary string.
3. Set `cnt` byte to the value of top 4 bits of `e0`: `cnt = e0[31] >> 4`.
4. Let `counter` integer equal `cnt`.
5. Calculate a sequence of `n·m` 32-byte random overlay values: `{o[i]} = StreamHash(uint64le(counter) || msghash || {p[i]} || {uint64le(j[i])}, 32·n·m)`, where:
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
        6. Calculate point `R[t,i’] = z[t,i]·B[t] - e[t,i]·P[t,i]` and encode it as a 32-byte [public key](#point). Use `e0` instead of `e[t,0]` in each ring.
        7. Calculate `e[t,i’] = ScalarHash(byte(cnt) || R[t,i’] || msghash || uint64le(t) || uint64le(i’) || w[t,i])` where `t` and `i’` are encoded as 64-bit little-endian integers.
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

2. Calculate the Fiat-Shamir factor (note that it commits to all input non-secret data via `msghash` as necessary):

        h = ScalarHash("h" || msghash)

3. Calculate the base point by applying `h` to [both generators](#generators):

        B = h·G + J

4. Calculate the set of public keys for the ring signature from the set of input asset ID commitments:
    
        P[i] = h·(AC’.H - AC[i].H) + AC’.Ba - AC[i].Ba

5. Calculate the private key: `p = c’ - c mod L`.
6. [Create a ring signature](#create-ring-signature) using `msghash`, `B`, `{P[i]}`, `j`, and `p`.
7. Return the list of asset ID commitments `{AC[i]}` and the ring signature `e[0], s[0], ... s[n-1]`.

Note: unlike the [value range proof](#value-range-proof), this ring signature is not used to store encrypted payload data because decrypting it would reveal the asset ID of one of the inputs to the recipient.


### Verify Asset Range Proof

**Inputs:**

1. `AC’`: the target [asset ID commitment](#asset-id-commitment).
2. One of the two [asset range proofs](#asset-range-proof):
    1. A non-confidential asset range proof consisting of:
        1. `assetID`: an [asset ID](blockchain.md#asset-id).
    2. A confidential asset range proof consisting of:
        1. `{AC[i]}`: `n` input [asset ID commitments](#asset-id-commitment).
        2. `e[0], s[0], ... s[n-1]`: the ring signature.
        3. Provided separately: 
            * `message`: a variable-length string.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. If the asset range proof is non-confidential:
    1. Compute [asset ID point](#asset-id-point): `A’ = 8·Hash256(assetID || counter)`.
    2. Verify that [point pair](#point-pair) `(A’,O)` equals `AC’`.
2. If the asset range proof is confidential:
    1. Calculate the message hash to sign:

            msghash = Hash256("ARP" || AC’ || AC[0] || ... || AC[n-1] || message)
    
    2. Calculate the Fiat-Shamir factor (note that it commits to all input non-secret data via `msghash` as necessary):

            h = ScalarHash("h" || msghash)

    3. Calculate the base point by applying `h` to [both generators](#generators):

            B = h·G + J

    4. Calculate the set of public keys for the ring signature from the set of input asset ID commitments:

            P[i] = h·(AC’.H - AC[i].H) + AC’.Ba - AC[i].Ba

    4. [Verify the ring signature](#verify-ring-signature) `e[0], s[0], ... s[n-1]` with `msg` and `{P[i]}`.
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



### Verify Value Proof

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment) to `assetid`.
2. `VC`: the [value commitment](#value-commitment) to `value`.
3. `assetid`: the [asset ID](blockchain.md#asset-id) to be proven used in `AC`.
4. `value`: the amount to be proven.
5. `(QG,QJ),e,s`: the [excess commitment](#excess-commitment) with its signature that proves that `assetid` and `value` are committed to `AC` and `VC`.
6. `message`: a variable-length string.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. [Verify excess commitment](#verify-excess-commitment) `(QG,QJ),e,s,message`.
2. Compute [asset ID point](#asset-id-point): `A’ = 8·Hash256(assetID || counter)`.
4. [Create nonblinded value commitment](#create-nonblinded-value-commitment): `V’ = value·A’`.
5. Verify that [point pair](#point-pair) `(QG + V’, QJ)` equals `VC`.



### Create Value Range Proof

**Inputs:**

1. `AC’`: the [asset ID commitment](#asset-id-commitment).
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
6. Calculate the message to sign: `msghash = Hash256("VRP" || AC’ || VC || uint64le(N) || uint64le(exp) || uint64le(vmin) || message)` where `N`, `exp`, `vmin` are encoded as 64-bit little-endian integers.
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
11. Calculate the Fiat-Shamir factor:

        h = ScalarHash("h" || msghash || D[0] || ... || D[n-2])
        
12. Precompute reusable points across all digit commitments:

        X1 = h·Bv
        X2 = AC’.H + h·Ba

13. For `t` from `0` to `n-1` (each digit):
    1. Calculate base point: `B[t] = G’[t] + h·J`.
    2. For `i` from `0` to `base-1` (each digit’s value):
        1. Calculate point `P[t,i] = D[t] + X1 - i·(base^t)·X2`.
14. [Create Borromean Ring Signature](#create-borromean-ring-signature) `brs` with the following inputs:
    * `msghash` as the message to sign.
    * `n`: number of rings.
    * `m = base`: number of signatures per ring.
    * `{B[i]}`: `n` base points.
    * `{P[i,j]}`: `n·m` [points](#point).
    * `{f}`: the blinding factor `f` repeated `n` times.
    * `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[t,j[t]] == f·G’[t]`.
    * `{r[i]} = {ct[i]}`: random string consisting of `n·m` 32-byte ciphertext elements.
15. If failed to create borromean ring signature `brs`, return nil. The chance of this happening is below 1 in 2<sup>124</sup>. In case of failure, retry [creating blinded value commitments](#create-blinded-value-commitments) with incremented counter. This would yield a new blinding factor `f` that will produce different digit blinding keys in this algorithm.
16. Return the [value range proof](#value-range-proof):
    * `N`:  number of blinded bits (equals to `2·n`),
    * `exp`: exponent (zero),
    * `vmin`: minimum value (zero),
    * `{D[t]}`: `n-1` digit commitments encoded as [public keys](#point) (excluding the last digit commitment),
    * `{e,s[t,j]}`: `1 + n·4` 32-byte elements representing a [borromean ring signature](#borromean-ring-signature),



### Verify Value Range Proof

**Inputs:**

1. `AC’`: the [asset ID commitment](#asset-id-commitment).
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
3. Calculate the message to verify: `msghash = Hash256("VRP" || AC’ || VC || uint64le(N) || uint64le(exp) || uint64le(vmin) || message)` where `N`, `exp`, `vmin` are encoded as 64-bit little-endian integers.
4. Calculate last digit commitment `D[n-1] = (10^(-exp))·(VC.V - vmin·AC’.H) - ∑(D[t])`, where `∑(D[t])` is a sum of all but the last digit commitment specified in the input to this algorithm.
5. Calculate the Fiat-Shamir factor:

        h = ScalarHash("h" || msghash || D[0] || ... || D[n-2])

6. Precompute reusable points across all digit commitments:

        X1 = h·Bv
        X2 = AC’.H + h·Ba

7. For `t` from `0` to `n-1` (each digit):
    1. Calculate generator `G’[t]`:
        1. If `t` is less than `n-1`: set `G’[t] = G[t]`, where `G[t]` is a [tertiary generator](#generators) at index `t`.
        2. If `t` equals `n-1`: set `G’[t] = G - ∑G[i]` for all `i` from `0` to `n-2`.
    2. Calculate base point: `B[t] = G’[t] + h·J`.
    3. Define `base = 4`.
    4. For `i` from `0` to `base-1` (each digit’s value):
        1. Calculate point `P[t,i] = D[t] + X1 - i·(base^t)·X2`. For efficiency perform recursive point addition of `-(base^t)·X2` instead of scalar multiplication.
8. [Verify Borromean Ring Signature](#verify-borromean-ring-signature) with the following inputs:
    * `msghash`: the 32-byte string being verified.
    * `n`: number of rings.
    * `m=base`: number of signatures in each ring.
    * `{B[i]}`: `n` base points.
    * `{P[i,j]}`: `n·m` public keys, [points](#point) on the elliptic curve.
    * `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.
9. Return `true` if verification succeeded, or `false` otherwise.



### Recover Payload From Value Range Proof

**Inputs:**

1. `AC’`: the [asset ID commitment](#asset-id-commitment).
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
3. Calculate the message to verify: `msghash = Hash256("VRP" || AC’ || VC || uint64le(N) || uint64le(exp) || uint64le(vmin) || message)` where `N`, `exp`, `vmin` are encoded as 64-bit little-endian integers.
4. Calculate last digit commitment `D[n-1] = (10^(-exp))·(VC.V - vmin·AC’.H) - ∑(D[t])`, where `∑(D[t])` is a sum of all but the last digit commitment specified in the input to this algorithm.
5. Calculate the Fiat-Shamir factor:

        h = ScalarHash("h" || msghash || D[0] || ... || D[n-2])

6. For `t` from `0` to `n-1` (each digit):
    1. Calculate generator `G’[t]`:
        1. If `t` is less than `n-1`: set `G’[t] = G[t]`, where `G[t]` is a [tertiary generator](#generators) at index `t`.
        2. If `t` equals `n-1`: set `G’[t] = G - ∑G[i]` for all `i` from `0` to `n-2`.
    2. Calculate `digit[t] = value & (0x03 << 2·t)` where `<<` denotes a bitwise left shift.
    3. Calculate `j[t] = digit[t] >> 2·t` where `>>` denotes a bitwise right shift.
    4. Calculate base point: `B[t] = G’[t] + h·J`.
    5. Define `base = 4`.
    6. For `i` from `0` to `base-1` (each digit’s value):
        1. Calculate point `P[t,i] = D[t] + X1 - i·(base^t)·X2`. For efficiency perform recursive point addition of `-(base^t)·X2` instead of scalar multiplication.

7. [Recover Payload From Borromean Ring Signature](#recover-payload-from-borromean-ring-signature): compute an array of `2·N` 32-byte chunks `{ct[i]}` using the following inputs (halt and return `nil` if decryption fails):
    * `msghash`: the 32-byte string to be signed.
    * `n=N/2`: number of rings.
    * `m=base`: number of signatures in each ring.
    * `{B[i]}`: `n` base points.
    * `{P[i,j]}`: `n·m` public keys, [points](#point) on the elliptic curve.
    * `{f}`: the blinding factor `f` repeated `n` times.
    * `{j[i]}`: the list of `n` indexes of the designated public keys within each ring, so that `P[t,j[t]] == f·G’[t]`.
    * `{e0, s[0,0], ..., s[i,j], ..., s[n-1,m-1]}`: the [borromean ring signature](#borromean-ring-signature), `n·m+1` 32-byte elements.
8. Derive payload encryption key unique to this payload and the value: `pek = Hash256("VRP.pek" || rek || f || VC)`.
9. [Decrypt payload](#decrypt-payload): compute an array of `2·N-1` 32-byte chunks: `{pt[i]} = DecryptPayload({ct[i]}, pek)`. If decryption fails, halt and return `nil`.
10. Return `{pt[i]}`, a plaintext array of `2·N-1` 32-byte elements.





### Create Excess Commitment

**Inputs:**

1. `q`: the [excess blinding factor](#excess-factor)
2. `message`: a variable-length string.

**Output:**

1. `(QG,QJ)`: the [point pair](#point-pair) representing an ElGamal commitment to `q` using [generators](#generators) `G` and `J`.
2. `(e,s)`: the Schnorr signature proving that `(QG,QJ)` does not affect asset amounts.

**Algorithm:**

1. Calculate a [point pair](#point-pair) `QG = q·G, QJ = q·J`.
2. Calculate Fiat-Shamir factor `h = ScalarHash("EC" || G || J || QG || QJ || message)`.
3. Calculate the base point `B = h·G + J`.
4. Calculate the nonce `k = ScalarHash(h || q)`.
5. Calculate point `R = k·B`.
6. Calculate scalar `e = ScalarHash("e" || h || R)`.
7. Calculate scalar `s = k + q·e mod L`.
8. Return `(s,e)`.


### Verify Excess Commitment

**Inputs:**

1. `(QG,QJ)`: the [point pair](#point-pair) representing an ElGamal commitment to secret blinding factor `q` using [generators](#generators) `G` and `J`.
2. `(e,s)`: the Schnorr signature proving that `(QG,QJ)` does not affect asset amounts.
3. `message`: a variable-length string.

**Output:** `true` if the verification succeeded, `false` otherwise.

**Algorithm:**

1. Calculate Fiat-Shamir factor `h = ScalarHash("EC" || G || J || QG || QJ || message)`.
2. Calculate the base point `B = h·G + J`.
3. Calculate combined public key point `Q = h·QG + QJ`.
4. Calculate point `R = s·B - e·Q`.
5. Calculate scalar `e’ = ScalarHash("e" || h || R)`.
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
4. Calculate the sum of excess commitments: `Tq = ∑[(QG[i], QJ[i]), i from 0 to k-1]`.
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

When creating a confidential issuance, the first step is to construct the rest of the input commitment and input witness, including an asset issuance choice for each asset that one wants to include in the anonymity set. The issuance key for each asset should be extracted from the [issuance programs](blockchain.md#program). (Issuance programs that support confidential issuance should have a branch that checks use of the correct issuance key using `ISSUANCEKEY` instruction.)

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `c`: the [blinding factor](#asset-id-blinding-factor) for commitment `AC` such that: `AC.H == A[j] + c·G`, `AC.Ba == c·J`.
3. `{a[i]}`: `n` 32-byte unencrypted [asset IDs](blockchain.md#asset-id).
4. `{Y[i]}`: `n` issuance keys encoded as [points](#point) corresponding to the asset IDs,
5. `message`: a variable-length string,
6. `nonce`: unique 32-byte [string](blockchain.md#string) that makes the tracing point unique,
7. `j`: the index of the asset being issued (such that `AC.H == A[j] + c·G`).
8. `y`: the private key for the issuance key corresponding to the asset being issued: `Y[j] = y·G`.

**Output:** an [issuance asset range proof](#issuance-asset-range-proof) consisting of:

* `{Y[i]}`: `n` issuance keys encoded as [points](#point) corresponding to the asset IDs,
* `T`: tracing [point](#point),
* `Bm`: blinded marker [point](#point),
* `ms = (e’,s’)`: the marker signature,
* `rs = {e[0], s[0], ... s[n-1]}`: the issuance ring signature.


**Algorithm:**

1. Calculate the base hash: `basehash = Hash256("IARP" || AC || uint64le(n) || a[0] || ... || a[n-1] || Y[0] || ... || Y[n-1] || nonce || message)` where `n` is encoded as a 64-bit unsigned little-endian integer.
2. Calculate marker point `M`:
    1. Let `counter = 0`.
    2. Calculate `Hash256("M" || basehash || uint64le(counter))` where `counter` is encoded as a 64-bit unsigned integer using little-endian convention.
    3. Decode the resulting hash as a [point](#point) `P` on the elliptic curve.
    4. If the point is invalid, increment `counter` and go back to step 2. This will happen on average for half of the asset IDs.
    5. Calculate point `M = 8·P` (8 is a cofactor in edwards25519) which belongs to a subgroup [order](#elliptic-curve-parameters) `L`.
3. Calculate the tracing point: `T = y·(J + M)`.
4. Calculate the blinded marker using the blinding factor used by commitment `AC`: `Bm = c·M`.
5. Calculate a 32-byte message hash and three 64-byte Fiat-Shamir challenges for all the signatures (total 224 bytes):

        (msghash, h1, h2, h3) = StreamHash("h" || basehash || M || T || Bm, 32 + 3·64)

6. Interpret `h1`, `h2`, `h3` as 64-byte little-endian integers and reduce each of them modulo subgroup order `L`.
7. Create proof that the discrete log `Bm/M` is equal to the discrete log `AC.Ba/J`:
    1. Compute base point `B = h1·M + J`.
    2. Calculate the nonce `k = ScalarHash("k" || msghash || c)`.
    3. Calculate point `R = k·B`.
    4. Calculate scalar `e’ = ScalarHash("e" || msghash || R)`.
    5. Calculate scalar `s’ = k + c·e mod L`.
    6. Let the marker signature `ms = (e’,s’)`.
8. Calculate [asset ID points](#asset-id-point) for each `{a[i]}`: `A[i] = 8·Decode(Hash256(a[i]...))`.
9. Calculate point `Q = Ba + Bm + h2·T`.
10. Calculate points `{P[i]}` for `n` pairs of asset ID points and corresponding issuance keys `A[i], Y[i]`:

        P[i] = AC.H — A[i] + h2·Y[i]

11. Create ring proof of discrete log equality for the pair `P[j]/G` and `Q/(J+M)`:
    1. Calculate base point `B = G + h3·(J+M)`.
    2. For each `P[i]` compute `P’[i] = P[i] + h3·Q`.
    3. Calculate the signing key `x = c + h2·y`.
    4. [Create a ring signature](#create-ring-signature) `rs` using:
        * message `msghash`,
        * base point `B`, 
        * public keys `{P’[i]}`, 
        * secret index `j`, 
        * private key `x`.
12. Return [issuance asset range proof](#issuance-asset-range-proof) consisting of:
    * issuance keys `{Y[i]}`, 
    * tracing point `T`,
    * blinded marker point `Bm`,
    * marker signature `ms`, 
    * ring signature `rs`.



### Verify Issuance Asset Range Proof

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
        5. Calculate point `M = 8·P` (8 is a cofactor in edwards25519) which belongs to a subgroup [order](#elliptic-curve-parameters) `L`.
    3. Calculate a 32-byte message hash and three 64-byte Fiat-Shamir challenges for all the signatures (total 224 bytes):

            (msghash, h1, h2, h3) = StreamHash("h" || basehash || M || T || Bm, 32 + 3·64)

    4. Interpret `h1`, `h2`, `h3` as 64-byte little-endian integers and reduce each of them modulo subgroup order `L`.
    5. Verify proof that the discrete log `Bm/M` is equal to the discrete log `AC.Ba/J`:
        1. Compute base point `B = h1·M + J`.
        2. Compute public key `P = h1·Bm + AC.Ba`.
        3. Calculate point `R = s’·B - e’·P`.
        4. Calculate scalar `e” = ScalarHash("e" || msghash || R)`.
        5. Verify that `e”` is equal to `e’`.
    6. Calculate [asset ID points](#asset-id-point) for each `{a[i]}`: `A[i] = 8·Decode(Hash256(a[i]...))`.
    7. Calculate point `Q = Ba + Bm + h2·T`.
    8. Calculate points `{P[i]}` for `n` pairs of asset ID points and corresponding issuance keys `A[i], Y[i]`:
    
            P[i] = AC.H — A[i] + h2·Y[i]

    9. Verify ring proof of discrete log equality for one of the pairs `P[i]/G` and `Q/(J+M)`:
        1. Calculate base point `B = G + h3·(J+M)`.
        2. Precompute point `Q’ = h3·Q`.
        3. For each `P[i]` compute `P’[i] = P[i] + Q’`.
        4. [Verify the ring signature](#verify-ring-signature) `e[0], s[0], ... s[n-1]` with message `msghash` and public keys `{P’[i]}`.



### Create Issuance Proof

Issuance proof allows an issuer to prove whether a given confidential issuance is performed with their key or not.

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

**Output:** an issuance proof consisting of:

* triplet of points `(X, Z, Z’)`,
* pair of scalars `(e1,s1)`,
* pair of scalars `(e2,s2)`.

**Algorithm:**

1. [Verify issuance asset range proof](#verify-issuance-asset-range-proof) to make sure tracing and marker points are correct.
2. Calculate the blinding key `x`:

        x = ScalarHash("x" || AC || T || y || nonce || message)

3. Blind the tracing point being tested: `Z = x·T`.
4. Calculate commitment to the blinding key: `X = x·(J+M)`.
5. Calculate and blind a tracing point corresponding to the issuance key pair `y,Y`: `Z’ = x·y·(J+M)`.
6. Calculate a 32-byte message hash and two 64-byte Fiat-Shamir challenges for all the signatures (total 160 bytes):

        (msghash, h1, h2) = StreamHash("IP" || AC || T || X || Z || Z’, 32 + 2·64)

7. Interpret `h1` and `h2` as 64-byte little-endian integers and reduce each of them modulo subgroup order `L`.
8. Create a proof that `Z` blinds tracing point `T` and `X` commits to that blinding factor (i.e. the discrete log `X/(J+M)` is equal to the discrete log `Z/T`):
    1. Calculate base point `B1 = h1·(J+M) + T`.
    2. Calculate the nonce `k1 = ScalarHash("k1" || msghash || y || x)`.
    3. Calculate point `R1 = k1·B1`.
    4. Calculate scalar `e1 = ScalarHash("e1" || msghash || R1)`.
    5. Calculate scalar `s1 = k1 + x·e1 mod L`.
9. Create a proof that `Z’` is a blinded tracing point corresponding to `Y[j]` (i.e. the discrete log `Z’/X` is equal to the discrete log `Y[j]/G`):
    1. Calculate base point `B2 = h2·X + G`.
    2. Calculate the nonce `k2 = ScalarHash("k2" || msghash || y || x)`.
    3. Calculate point `R2 = k2·B2`.
    4. Calculate scalar `e2 = ScalarHash("e2" || msghash || R2)`.
    5. Calculate scalar `s2 = k2 + y·e2 mod L`.
5. Return points `(X, Z, Z’)`, signature `(e1,s1)` and signature `(e2,s2)`.



### Verify Issuance Proof

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
4. Issuance proof consisting of:
    * triplet of points `(X, Z, Z’)`,
    * pair of scalars `(e1,s1)`,
    * pair of scalars `(e2,s2)`.

**Output:** 

* If the proof is valid: `“yes”` or `“no”` indicating whether the key `Y[j]` was used or not to issue asset ID in the commitment `AC`.
* If the proof is invalid: `nil`.

**Algorithm:**

1. Calculate a 32-byte message hash and two 64-byte Fiat-Shamir challenges for all the signatures (total 160 bytes):

        (msghash, h1, h2) = StreamHash("IP" || AC || T || X || Z || Z’, 32 + 2·64)

2. Interpret `h1` and `h2` as 64-byte little-endian integers and reduce each of them modulo subgroup order `L`.
3. Verify that `Z` blinds tracing point `T` and `X` commits to that blinding factor (i.e. the discrete log `X/(J+M)` is equal to the discrete log `Z/T`):
    1. Calculate base point `B1 = h1·(J+M) + T`.
    2. Calculate public key `P1 = h1·X + Z`.
    3. Calculate point `R1 = s1·B1 - e1·P1`.ruined
    4. Calculate scalar `e’ = ScalarHash("e1" || msghash || R1)`.
    5. Verify that `e’` is equal to `e1`. If validation fails, halt and return `nil`.
4. Verify that `Z’` is a blinded tracing point corresponding to `Y[j]` (i.e. the discrete log `Z’/X` is equal to the discrete log `Y[j]/G`):
    1. Calculate base point `B2 = h2·X + G`.
    2. Calculate public key `P2 = h2·Z’ + Y[j]`.
    3. Calculate point `R2 = s2·B2 - e2·P2`.
    4. Calculate scalar `e” = ScalarHash("e2" || msghash || R2)`.
    5. Verify that `e”` is equal to `e2`. If validation fails, halt and return `nil`.
5. If `Z` is equal to `Z’` return `“yes”`. Otherwise, return `“no”`.




















## High-level procedures

### Verify Output

**Inputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `VC`: the [value commitment](#value-commitment).
3. `ARP`: the [asset range proof](#asset-range-proof).
4. `VRP`: the [value range proof](#value-range-proof).

**Output:** `true` if verification succeeded, `false` otherwise.

**Algorithm:**

1. [Verify asset range proof](#verify-asset-range-proof) using `AC` and `ARP`.
2. [Verify value range proof](#verify-value-range-proof) using `AC`, `VC` and `VRP`.
3. Return `true`.


### Verify Issuance

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

1. [Verify issuance asset range proof](#verify-issuance-asset-range-proof) using `(AC,IARP,{a[i]},message,nonce)`.
2. [Verify value range proof](#verify-value-range-proof) using `AC`, `VC` and `VRP`.
3. Return `true`.



### Verify Confidential Assets

**Inputs:**

1. List of issuances, each input consisting of:
    * `AC`: the [asset ID commitment](#asset-id-commitment).
    * `VC`: the [value commitment](#value-commitment).
    * `IARP`: the [issuance asset ID range proof](#issuance-asset-range-proof) together with:
        * `{a[i]}`: [asset identifiers](blockchain.md#asset-id), one for each issuance key in the range proof.
        * `message`: a variable-length string.
        * `nonce`: a unique 32-byte string.
    * `VRP`: the [value range proof](#value-range-proof).
2. List of spends, each spend consisting of:
    * `AC`: the [asset ID commitment](#asset-id-commitment).
    * `VC`: the [value commitment](#value-commitment).
3. List of outputs, each output consisting of:
    * `AD`: the [asset ID commitment](#asset-id-commitment).
    * `VD`: the [value commitment](#value-commitment).
    * `ARP`: the [asset range proof](#asset-range-proof) or an empty string.
    * `VRP`: the [value range proof](#value-range-proof).
4. List of [excess commitments](#excess-commitment): `{(QC[i], s[i], e[i], message[i])}`.

**Output:** `true` if verification succeeded, `false` otherwise.

**Algorithm:**

1. [Verify each issuance](#verify-issuance).
2. For each output:
    1. If `ARP` is empty has zero keys, verify that the output `AC` equals one of the asset ID commitments in the inputs or issuances.
    2. If `ARP` is confidential, verify that each asset ID commitment candidate belongs to the set of the asset ID commitments on the inputs and issuances.
    3. [Verify output](#verify-output).
3. [Verify value commitments balance](#verify-value-commitments-balance) using a union of issuance and input value commitments as input commitments.
4. Return `true`.





### Encrypt Issuance WIP

**Inputs:**

1. `rek`: the [record encryption key](#record-encryption-key) unique to this issuance.
2. `assetID`: the output asset ID.
3. `value`: the output amount.
4. `N`: number of bits to encrypt (`value` must fit within `N` bits).
5. `{(assetIDs[i], Y[i])}`: `n` input asset IDs and corresponding issuance public keys.
6. `y`: issuance key for `assetID` such that `Y[j] = y·G` where `j` is the index of the issued asset: `assetIDs[j] == assetID`.
7. `message`: a variable-length string to be signed.

**Outputs:**

1. `AC`: the [asset ID commitment](#asset-id-commitment).
2. `VC`: the [value commitment](#value-commitment).
3. `IARP`: the [issuance asset ID range proof](#issuance-asset-range-proof).
4. `VRP`: the [value range proof](#value-range-proof).
5. `c`: the [asset ID blinding factor](#asset-id-blinding-factor) for the asset ID commitment `AC`.
6. `f`: the [value blinding factor](#value-blinding-factor) for the value commitment `VC`.

In case of failure, returns `nil` instead of the items listed above.

**Algorithm:**

1. [Derive asset encryption key](#asset-id-encryption-key) `aek` from `rek`.
2. [Derive value encryption key](#value-encryption-key) `vek` from `rek`.
3. [Create nonblinded asset ID commitment](#create-nonblinded-asset-id-commitment) for all values in `{assetIDs[i]}`: `A[i] = 8·Decode(Hash256(assetIDs[i]...))`.
4. Find `j` index of the `assetID` among `{assetIDs[i]}`. If not found, halt and return `nil`.
5. [Create blinded asset ID commitment](#create-blinded-asset-id-commitment): compute `(H,c)` from `(A, 0, aek)`.
6. [Create blinded value commitment](#create-blinded-value-commitment): compute `(V,f)` from `(vek, value, H, c)`.
7. [Create issuance asset range proof](#create-issuance-asset-range-proof): compute `IARP` from `(H, c, {A[i]}, {Y[i]}, vmver’, program’, j, y)`.
8. [Create Value Range Proof](#create-value-range-proof): compute `VRP` from `(H, V, (0x00...,0x00...), N, value, {0x00...}, f, rek)`.
9. Create [blinded asset ID descriptor](#blinded-asset-id-descriptor) `AD` containing `H` and all-zero [encrypted asset ID](#encrypted-asset-id).
10. Create [blinded value descriptor](#blinded-value-descriptor) `VD` containing `V` and all-zero [encrypted value](#encrypted-value).
11. Return `(AD, VD, IARP, VRP, c, f)`.




### Encrypt Output WIP

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

1. `AD`: the [asset ID descriptor](#asset-id-descriptor).
2. `VD`: the [value descriptor](#value-descriptor).
3. `VRP`: the [value range proof](#value-range-proof).
4. `c’`: the output [asset ID blinding factor](#asset-id-blinding-factor) for the asset ID commitment `H’`.
5. `f’`: the output [value blinding factor](#value-blinding-factor).

In case of failure, returns `nil` instead of the items listed above.

**Algorithm:**

1. Encode `plaintext` using [varstring31](blockchain.md#string) encoding and split the string in 32-byte chunks `{pt[i]}` (last chunk padded with zero bytes if needed).
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



### Decrypt Output WIP

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
5. `plaintext`: the binary string that has length of less than `32·(2·N-1)` bytes when encoded as [varstring31](blockchain.md#string).

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
    2. Flatten the array `{pt[i]}` in a binary string and decode it using [varstring31](blockchain.md#string) encoding. If decoding fails, halt and return `nil`.
6. If value range proof `VRP` is empty, set `plaintext` to an empty string.
7. Return `(assetID, value, c, f, plaintext)`.






## Test vectors

TBD: Hash256, StreamHash, ScalarHash.

TBD: RS, BRS.

TBD: AC, VC, excess commitments.

TBD: ARP, IARP, VRP.

