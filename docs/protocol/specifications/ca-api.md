# Confidential Assets API

* [Introduction](#introduction)
* [Data structures](#data-structures)
* [Actions](#actions)
* [Encryption](#encryption)
* [Compatibility](#compatibility)
* [Trackable addresses](#trackable-addresses)
* [Alternatives considered](#alternatives-considered)
* [Swagger specification](#swagger-specification)

## Introduction

New design for transaction builder that focuses on privacy.

### Goals

1. Native support for transaction entries.
2. Privacy-preserving transaction signing
3. Confidential assets support.
4. Support for constructing witnesses for custom control and issuance programs.
5. Built-in support for encrypting payload inside and outside the range proof.

### Addressed problems

1. Linkability of inputs and outputs in multi-party transfers.
2. Asset range proofs for outputs with asset ID that’s missing among inputs.
3. Automated verification for additional rounds of finalization.
4. Future extensibility to custom signing schemes.
5. Privacy of metadata (such as signing instructions, account IDs etc) for shared tx templates.

### Functional overview

1. Building transaction:
    1. Single-party transaction (with one-shot signing)
    2. Multi-party transaction (with second round of signing)
    3. Encrypting outputs by default
    4. Encrypting outputs with receiver details (externally provided key material)
2. Selective disclosure:
    1. Export "disclosure"
    2. Import "disclosure"
    3. Per-output disclosure:
        1. Asset ID - on/off
        2. Amount - on/off
        3. Payload - on/off
    4. Per-transaction disclosure:
        1. Select output(s)
        2. Toggle assetid/amount/payload for the transaction
    5. Per-account disclosure (for tracking accounts):
        1. Xpub - for tracking, always on.
        2. Asset ID - on/off
        3. Amount - on/off
        4. Payload - on/off
    6. Disclosure is encrypted directly to the other core's public key, so it's safe to carry around.
        1. Core's pubkey is blinded using ChainKD to prevent linkability to the same Core by counter-parties.


### Single-party transaction

Alice wants to send money to Bob.

Both Alice and Bob add keys to their Chain::HSMSigner associated with a Chain::Client object,
so Client can do sign requests:

    client.signer.add_key(key, hsm.signer_conn)

Bob generates a receiver `rcvr` that encapsulates control program and
encryption keys.

    bob_rcvr = chain.accounts.create_receiver(
        account_alias: 'bob'
    )

First party (Alice) uses `client.transactions` API to build a transaction to send funds to a given receiver from Bob:

    tx1 = client.transactions.build do |b|
      b.spend_from_account    account_alias: 'alice', asset_id: 'USD',  amount: 140
      b.control_with_receiver receiver: bob_rcvr,     asset_id: 'USD',  amount: 140, data: "Hello, world!"
    end

Suppose Alice has 1000 USD on the inputs. Transaction builder creates two outputs:

* Output 1: 140 USD to Bob's account
* Output 2: 860 USD to Alice's account

Output 1 is using one-time program described in Bob's receiver, and payload and amount are encrypted with the keys specified in the receiver.

Output 2 uses one-time program for Alice's change address and the amount is encrypted with the keys derived for the Alice's account.

When transaction is built, it must be signed and submitted.

    tx = client.transactions.sign(tx)
    client.transactions.submit(tx)

Core verifies the fully-signed transaction and publishes it if it's valid.


### Multi-party transaction

In a multi-party transaction receivers are not necessary (although, can be used as well).
Instead, each party can specify outputs they are interested in and immediately encrypt
with their own encryption keys without excessive disclosure with other parties.

First party (Alice) uses `client.transactions` API to build a partial transaction:

    tx1 = client.transactions.build do |b|
      b.spend_from_account   account_alias: 'alice', asset_id: 'USD',  amount: 140
      b.control_with_account account_alias: 'alice', asset_id: 'AAPL', amount: 1
    end

Alice sends `tx1` to Bob to add additional steps. Bob sets `base_transaction` to `tx1` to build on top of it.

    tx2 = client.transactions.build do |b|
      b.base_transaction     tx1
      b.spend_from_account   account_alias: 'bob', asset_id: 'USD',  amount: 140
      b.control_with_account account_alias: 'bob', asset_id: 'AAPL', amount: 1
    end

Bob can in turn leave `tx2` unbalanced and forward it to Carl, etc.

When no one else is adding to the transaction, it must be signed:

    tx3 = client.transactions.sign(tx2)

Each party has to sign the transaction, so it becomes fully signed.

When it is fully signed, it can be submitted.

    if tx3.signed?
        client.transactions.submit(tx3)
    end

Core verifies the fully-signed transaction and publishes it if it's valid.


### Creating disclosure

Alice wants to have read access to Bob's transactions.

1. Alice generates an `Import Key` in her Chain Core — one-time pubkey used for encrypting and authenticating a "disclosure" document.
2. Alice sends `Import Key` to Bob who generates a "Disclosure" in his Core that will be encrypted to that Import Key.
3. Bob configures a new disclosure:
    1. Disclosure scope:
        * Output(s)
        * Transaction(s)
        * Account(s)
    2. Disclosed fields:
        * Asset ID
        * Amount
        * Data
4. Bob's Core prepares cryptographic keys and proofs necessary for a given disclosure. For example:
    * for per-output disclosure it simply contains proofs revealing little identifying information,
    * for account it exports xpubs to track the account and provides root keys to decrypt necessary fields.
5. Bob's Core encrypts disclosure to the given `Import Key`.
6. Bob sends the encrypted disclosure to Alice.
7. Alice imports the encrypted disclosure to Alice's Core.
8. Core derives the decryption key from its root key, verifies the proofs and imports the keys.
9. If the disclosure is at account level, Core begins watching the blockchain for that account and indexing past (?) and future transactions.









## Data structures

### Receiver

Receiver is shared with a sending party with minimum information necessary to correctly encrypt
the output.

    {
        output_version: 2,                        // version of the output (default is 1)
        control_program: "fa90e031...",           // control program
        expires_at: "2017-10-02T10:00:00-05:00",  // expiration date
        dek: "de01836...",                        // data-encryption key
        aek: "ae819f7...",                        // asset ID-encryption key
        vek: "fe791c0..."                         // amount-encryption key
    }


### Transaction template

Transaction template contains transaction entries and additional data that helps multiple parties to cooperatively create the transaction.

    {
        version: 2,  // version of the transaction template format
        finalized: false, // set to true after first `client.transactions.sign()` call
        balanced:  false, // set when no placeholder inputs/outputs left
        signed:    false, // set when all signatures and proofs are provided
        entries: [
            {
                type:    "txheader",
                version:  2,  // core protocol version
                mintime:  X,
                maxtime:  Y,
                txid:     "0fa8127a9fe8d89b12..." // only added after the first `client.transactions.sign()` call
            },
            {
                type: "mux2",
                excess_factor: "0f0eda17b9f1e...", // 32-byte scalar
            },
            {
                type: "input2",
                spent_output_id: "...",
            },
            {
                type: "output2",
                asset_commitment:   "ac00fa9eab0...",
                value_commitment:   "5ca9f901248...",
                valure_range_proof: "9df90af8a0c...",
                program:            "...",
                data:               "da1a00fa9e628...",
                exthash:            "e40fa89202...",

                asset_range_proof_instructions: {
                    asset_id: "...",
                    factor:   "..."
                },
            },
            {
                type: "input-placeholder",
                asset_id: "AAPL",
                amount:   1,
            },
            {
                type: "output-placeholder",
                asset_id: "USD",
                amount:   140,
            },
        ],
        payloads: [
            {
                version: 1,  // version of the payload encryption
                id: <payload-id>,
                private: ENCRYPTED{
                    version: 2, // version of the transaction template
                    entries: [
                        {
                            type:    "txheader",
                            version:  2,  // core protocol version
                            mintime:  X,
                            maxtime:  Y,
                        },
                        {
                            type: "mux2",
                            excess_factor: "0f0eda17b9f1e...", // 32-byte scalar
                        },
                        {
                            type: "input2",
                            spent_output_id: "...",
                            program_witness: {
                                ...
                            }
                        },
                        {
                            type: "output2",
                            asset_commitment: "ac00fa9eab0...",
                            value_commitment: "5ca9f901248...",
                            program:          "...",
                            data:             "da1a00fa9e628...",
                            exthash:          "e40fa89202...",
                            asset_range_proof_instructions: {   // private instructions for this party
                                asset_id: "...",
                                factor:   "..."
                            },
                        },
                    ]
                }
            }
        ]
    }

### Program Witness Template

**Program Witness Template** represent context data, concrete witness data, AST paths and signature templates for satisfying a control/issuance program.

This does not include other possible witnesses.

    {
        context: {
            type: "tx",                   // type of context - "transaction", "block" etc
            entry_type: "input",          // type of entry in the transaction context
            vm_version: 2,                // version of the VM (affects allowed types of witness components)
            program: "fae89bcdfaf23...",  // program AST
            position: 0,                  // position of the destination of the current entry
            asset_id: "4f39abd7...",      // asset id of the current entry
            amount:   1,                  // number of asset units in the current entry
        },
        arguments: [                      // stack of arguments for the program
            {
                type: "clause",           // which clause to trigger in a program
                clause: "unlock",         // name of a clause - the
            },
            {
                type: "data",             // raw piece of data
                datatype: "string",       //
                hex: "8e92af820..."
            },
            {
                type: "multisig",
                "quorum": 1,
                "keys": [
                    {
                        "xpub": "...",
                        "derivation_path": [...]
                    }
                ],
                hash_type: "txsighash", // either 'hash' (exact value) or 'hash_type' (placeholder)
            },
            ...
        ]
    }

### Entry data

The raw data string associated with an output/retirement/issuance entry is composed of the following encrypted strings:

    <encrypted-asset-id><encrypted-amount><encrypted-refdata>

If the asset ID is not confidential, its ciphertext is omitted from the data attachment.
Likewise, if the amount is not confidential, its ciphertext is omitted from the data attachment.

#### Encode data

1. If the asset ID is confidential, [encrypt it](ca.md#encrypt-asset-id) and set `ea` to a 64-byte ciphertext. Otherwise, set `ea` to an empty string.
2. If the amount is confidential, [encrypt it](ca.md#encrypt-value) and set `ev` to a 40-byte ciphertext. Otherwise, set `ea` to an empty string.
3. If the data is confidential, [encrypt it](#reference-data-encryption) and set `ed` to the ciphertext. Otherwise, set `ed` to the plaintext data.
4. Concatenate `d = ea || ev || ed`.
5. Attach `d` to the transaction entry (output, retirement or issuance).

#### Decode data

1. If the asset ID is confidential, extract first 64 bytes of the data attachment and [decrypt them](ca.md#decrypt-asset-id).
2. If the amount is confidential, extract next 40 bytes of the data attachment and [decrypt them](ca.md#decrypt-value).
3. If the data is confidential, [decrypt](#reference-data-encryption) the remaining bytes of the data. Otherwise, set the plaintext to the remaining bytes without modification.


### Disclosure

#### Encrypted Disclosure

    {
        type: "encdisclosure1",              # version of the encrypted disclosure
        import_key: "fe9af9bc3923...",       # IK pubkey
        selector:   "589af9b1a730...",       # blinding selector
        ciphertext: "cc9e012f7a8f99ea9b...", # encrypted hex of the Cleartext Disclosure object
    }

#### Cleartext Disclosure

    {
        type: "disclosure1",                     # version of the cleartext disclosure
        items: [
            {
                scope: "output"/"retirement"/"issuance"/"account",  # type of the scope
                ...
            },
            {
                scope: "output",                 # disclosure for a single output/retirement/issuance
                entry_id: "...",                 # ID of the output (hex-encoded)
                transaction_id: "...",           # ID of the transaction (hex-encoded)
                data: {
                    plaintext: "...",            # Hex-encoded decrypted reference data
                    dek: "...",                  # Data Encryption Key for this entry
                },
                asset_id: {
                    asset_id: "...",             # Hex-encoded plaintext asset ID
                    aek: "...",                  # Asset Encryption Key for this entry
                },
                amount: {
                    amount: ...,                 # Hex-encoded plaintext asset ID
                    vek: "...",                  # Value Encryption Key for this entry
                }
            },
            {
                scope: "account",                # disclosure for an account
                account_id: "...",               # ID of the output (hex-encoded)
                account_xpubs: [...],            # List of xpubs forming an account
                account_quorum: 1,               # Number of keys required for signing in the account
                data: {
                    dek: "...",                  # Data Encryption Key for this account
                },
                asset_id: {
                    aek: "...",                  # Asset Encryption Key for this account
                },
                amount: {
                    vek: "...",                  # Value Encryption Key for this account
                }
            },
        ]
    }







## Actions

### Build

The API to build a transaction makes transfers confidential by default without explicit handling of encryption keys.

If the transfer is made to an account, it is encrypted with the keys associated with this account.

If the transfer is made to a receiver, the output or retirement is encrypted with keys specified by the receiver. If some keys are omitted,
then no encryption takes place.

To control confidentiality of specific entries, `confidential` key is used (defaults are `true` for all fields).

    chain.transactions.build do |b|
        b.base_transaction tx  # (optional)
        b.issue                ..., confidential: {data: true, asset_id: true, amount: true}
        b.control_with_account ..., confidential: {data: true, asset_id: true, amount: true}
        b.retire               ..., confidential: {data: true, asset_id: true, amount: true}
    end

It is an error to use `confidential` key with actions `spend_*` or `control_with_receiver`.
This is because spends inherit confidentiality from the previous outputs and receivers fully control confidentiality options.

Procedure:

1. Prepare a partial transaction with data from the `base_transaction`:
    1. Reserve unspent outputs.
    2. Add necessary inputs spending reserved unspent outputs.
    3. Add necessary issuances.
    4. Prepare `signing_instructions` for each input and issuance.
    5. Create and encrypts new outputs, adds VRPs (but not ARPs).
    6. Compute the `excess` blinding factor.
    7. Set mintime to current time: `mintime = currentTime`. TBD: this should be randomized slightly to avoid Core fingerprinting.
    8. Set maxtime to expiration time: `maxtime = expirationTime`. TBD: this should be randomized slightly to avoid Core fingerprinting.
    9. Prepare ARP instructions for each output: `asset_range_proof_instructions: {asset_id:..., factor:..., [input_id:..., input_factor:...]}`.
        * Input ID and input's blinding factor are included if such input is known.
    10. For each output in `base_transaction` where `asset_range_proof_instructions` are specified and partial tx has an input with the matching asset ID:
        1. Copy that output to the partial transaction.
        2. Add `input_id/input_factor` to `asset_range_proof_instructions` for that copy.
        3. Strip `asset_range_proof_instructions` from that output in `base_transaction`.
    11. Encrypt the partial transaction:
        1. Generate unique uniform `payload-id`.
        2. Derive [payload encryption key](#key-derivation) from Chain Core's master key: `PK`.
        3. Encrypt the partial transaction and wraps it with `{payload_id:, blob:}` structure.
2. Merge the data from partial transaction with the existing template (`base_transaction`):
    1. Merge tx.mintime with partial's mintime: `mintime = MAX(basetx.mintime, partialtx.mintime)`.
    2. Merge tx.maxtime with partial's maxtime: `maxtime = MIN(basetx.maxtime, partialtx.maxtime)`.
    3. Add outputs, stripping `asset_range_proof_instructions` from those outputs where `input_id/input_factor` is specified.
    4. Add `{payload_id:, blob:}` to the list of payloads in the `base_transaction`.
    5. Add `excess` from the partial transaction to the `excess` value in the MUX entry in the `base_transaction`.
    6. Replaces `placeholder:true` outputs with re-computed placeholder outputs based on additional inputs/outputs.
3. Shuffles entries:
    * hashes each entry with a unique per-tx key (derived from `EK`, derived from `payload_id`)
    * sorts the list lexicographically

### Sign

1. Send transaction template to Core to decrypt and verify signing instructions:
    1. If `excess` factor in the MUX entry is non-zero:
        1. If the transaction ID is not fixed yet and the current party has at least one output in it:
            * `excess` is added to the value commitment and value range proof is re-created.
        2. Otherwise, `excess` is transformed into an "excess commitment" with signature and added to the MUX entry in the transaction.
        3. `excess` value is zeroed from the transaction template.
    2. Transaction ID is computed from the given transaction template.
    3. For each encrypted payload:
        1. Derive [payload encryption key](#key-derivation) from Chain Core's master key: `PK`.
        2. If the key successfully decrypts and authenticates the payload:
            1. If `txheader` entry is present:
                1. If `entry.version` is set, verify that it is equal to `tx.txheader.version`.
                2. If `entry.mintime` is set, verify that `txheader.mintime` is greater or equal to the `entry.mintime`.
                3. If `entry.maxtime` is set, verify that `txheader.maxtime` is less or equal to the `entry.maxtime`.
            2. For each `input/issuance` entry:
                1. Verify that it's present in the transaction.
            2. For each `output/retirement` entry:
                1. Verify that there's an entry with such asset/value commitment, program, data and exthash.
                2. If `asset_range_proof_instructions` has `input_id/input_factor`, create a corresponding ARP and add to the corresponding output entry.
        3. If the payload is not authenticated by the key `EK`, ignore it.
    4. If verification succeeded:
        1. Remove the processed payload
        2. Send signing instructions for inputs/issuances to the client.
2. Client receives signing instructions and sends them to the HSM signer:
    1. HSM signer signs using the signing instructions.
    2. Client places the signature over TXSIGHASH in the entry inside the transaction.


### Create Disclosure Import Key

The Core that needs to import a disclosure must first generate a recipient key to ensure
that the document is encrypted in-transit (without assuming direct secure connection between two Cores).

   import_key = client.disclosures.create_import_key()
   import_key_serialized = import_key.to_json


### Export Disclosure

Bob receives `import_key` from Alice and uses it to build a disclosure object:

    encrypted_disclosure = client.disclosures.build(import_key: import_key) do |d|
      d.disclose_entry(entry_id: "fa0e8fb0ad...",            fields: {asset_id: true, amount: true, data: true})
      d.disclose_transaction(transaction_id: "57ad0fea9...", fields: {asset_id: true, amount: true, data: true})
      d.disclose_account(account_alias: "bob",               fields: {asset_id: true, amount: true, data: true})
    end
    disclosure_serialized = encrypted_disclosure.to_json

The resulting object is encrypted to an `import_key`, contains minimal metadata needed for decryption
and can be safely transmitted to the receiving Core for import.

* `disclose_entry` — adds proofs and decryption keys for a single output/retirement/issuance.
* `disclose_transaction` — adds proofs and decryption keys for each output in the transaction (omits outputs not decrypted by this Core).
* `disclose_account` — adds account xpubs, derivation path and root decryption keys for tracking all outputs for a given account.


### Import Disclosure

Alice receives `disclosure` object from Bob and attempts to import in the Core:

    cleartext_disclosure = client.disclosures.import(disclosure)

Alice can decrypt and inspect the disclosure without importing:

    cleartext_disclosure = client.disclosures.decrypt(disclosure)
    cleartext_disclosure.scope                       # => 'output'/'transaction_id'/'account'
    cleartext_disclosure.items[0].asset_id.asset_id  # => "fae9f0af..."

TBD: Should we support querying all stored disclosures too?













## Encryption

### Key derivation

Core manages two hierarchies of keys for encrypting transaction templates and asset amounts.
The first hierarchy is used to protect transaction building process and is not tied to any
account. The second hierarchy is deliberately tied to accounts and control programs in order
to simplify indexing and tracking of the payments.

Core is initialized with a **root secret key** `RK` which is stored in the Core's DB.

    RK = random

From that key, Core creates a **root import key** `RIK`, an xprv/xpub pair using RK as a seed:

    RIK = ChainKD(seed: "RIK" || RK)

For accounts and outputs Core creates **root confidentiality key** `RCK`:

    RCK = SHAKE128("RCK" || RK , 32)

To encrypt fields, `RCK` is expanded to a vector key containing 3 32-byte keys:

    RDEK, RAEK, RVEK = SHAKE128("RDEK" || RCK, 3·32)

    RDEK — Root Data Encryption Key
    RAEK — Root Asset ID Encryption Key
    RVEK — Root Value Encryption Key

For each account a deterministic account selector is made that's used to generate per-account keys:

    accsel = m,n,xpub1,xpub2,xpub3

    {ADEK,AAEK,AVEK} = SHAKE128("A" || {RDEK,RAEK,RVEK} || accsel, 32)

For each output, a key is derived using control program as a selector:

    {ODEK,OAEK,OVEK} = SHAKE128("O" || {ADEK,AAEK,AVEK} || control_program, 32)

For the retirement entry, a key is derived using the serialized `value_source` as selector and root keys:

    {TDEK,TAEK,TVEK} = SHAKE128("T" || {RDEK,RAEK,RVEK} || value_source, 32)

Asset ID-specific vector key:

    {SDEK,SAEK,SVEK} = SHAKE128("A" || {RDEK,RAEK,RVEK} || assetid, 32)

For each issuance, a key is derived using the anchor ID as selector and asset ID-specific keys:

    {YDEK,YAEK,YVEK} = SHAKE128("T" || {SDEK,SAEK,SVEK} || anchorID, 32)

For each access token there is a separate _access key_ `AK`:

    AK = SHAKE128(RK || access_token, 32)

For encryption of a transaction template payload, a unique key is derived from the _access key_ and _payload ID_:

    PK = SHAKE128(AK || payloadID, 32)



### Import key

Import key is created and used to encrypt/decrypt disclosure within Chain Core.

Applications are exposed to an opaque object that encapsulates a versioned import public key.

#### Generate Import Key

1. Load the `Root Import Key`, a ChainKD xpub.
2. Generate a unique selector used for blinding the root key:

        b = random 32 bytes

3. Derives one-time import pubkey (only the public key part):

        IKpub = ChainKD-NormalDerivation(RIKxpub, b).pubkey

4. Return a pair of the import key and a blinding selector `IK,b`:

        {
            type: "importkey1",
            key: IKpub,
            selector: b
        }

#### Encrypt Disclosure

1. Verify that import key’s `type` equals `importkey1`.
2. Serialize plaintext disclosure as `data`.
3. Generate a sender private key using Core's root key `RK` as a seed:

        r = ScalarHash("DH" || RK || IKpub || b || data)

4. Compute public sender key:

        R = r·G

5. Compute Diffie-Hellman secret to be used as encryption key:

        S = r·IKpub

6. Encode `S` as a public key and compute encryption key as:

        K = SHAKE128(S, 32)

7. Encrypt `data` using [Packet Encryption](#encrypt-packet) algorithm with key `K`:

        ciphertext = EncryptPacket(data, K)

8. Return encrypted disclosure:

        {
            type: "encdisclosure1", # version of the encrypted disclosure
            sender_key: R,
            selector:   b,
            ciphertext: <ciphertext>
        }

#### Decrypt Disclosure

1. Verify that encrypted disclosure's `type` equals `encdisclosure1`.
2. Derive import private key using the selector `b`:

        IKprv = ChainKD-NormalDerivation(RIKxprv, b).scalar

3. Compute Diffie-Hellman secret to be used as encryption key:

        S = IKprv·R

4. Encode `S` as a public key and compute encryption key as:

        K = SHAKE128(S, 32)

5. Decrypt `ciphertext` using [Packet Encryption](#decrypt-packet) algorithm with key `K`:

        data = DecryptPacket(ciphertext, K)

6. If decryption succeeded, return deserialized disclosure:

        deserialize(data)


### Reference data encryption

Reference data is encrypted/decrypted using [Packet Encryption](#packet-encryption) algorithm with one of the entry-specific keys:

    ODEK - Data Encryption Key for an output entry
    TDEK - Data Encryption Key for an retirement entry
    YDEK — Data Encryption Key for an issuance entry


### Packet Encryption

#### Encrypt Packet

1. Compute keystream of the same length as plaintext: `keystream = SHAKE128(EK, len(payload))`
2. Encrypt the payload with the keystream: `ct = payload XOR keystream`.
3. Compute MAC on the ciphertext `ct`: `mac = SHAKE128(ct || EK, 32)`.
4. Append MAC to the ciphertext: `ct’ = ct || mac`.

#### Decrypt Packet

1. Split ciphertext into raw ciphertext and MAC (last 32 bytes): `ct, mac`.
2. Compute MAC on the ciphertext `ct`: `mac’ = SHAKE128(ct || EK, 32)`.
3. Compare in constant time `mac’ == mac`. If not equal, return nil.
4. Compute keystream of the same length as ciphertext: `keystream = SHAKE128(EK, len(ciphertext))`
5. Decrypt the payload by XORing keystream with the ciphertext: `payload = ct XOR keystream`.
6. Return `payload`.










## Compatibility

Current SDK uses `signer.sign()` method to sign a partial transaction. We can keep this behavior and introduce an additional API:

    HSMSigner.sign() - existing behavior as is
    Client.sign()    - verifies payload and signs TXSIGHASH-based predicate via Client.signer.sign(txhash instead of checkpredicate)

When users upgrade to a new Chain Core, the tx template is changed, but the behavior of the application remains the same. Then, they can smoothly transition to a new API usage:

1. Configure Client.signer instead of a standalone HSMSigner instance.
2. Introduce a second round of tx template exchange to do `Client.sign` after other parties have participated.
3. If using confidential amounts, new signing mechanism is required.







## Trackable addresses

(This is similar to Stealth Addresses proposal, but compatible with usage by the recipient.)

To make accounts to be trackable without exchanging receivers it is possible 
to embed a random selector within the control program.

To make it compatible with sequential key derivation, the random selector is
deterministically produced from the index and an xpub.

Scheme overview:

1. Alice is an auditor, Bob is an account holder.
2. Bob generates a receiver with a sequence number N.
3. Bob deterministically derives a random nonce to be used as a ChainKD selector:

        nonce = SHA256(xpub || uint64le(N)[0,16]
        
4. Bob derives a one-time key using that nonce:

        pubkey = ChainKD-ND(xpub, nonce)

5. Bob creates a control program where pubkey is annotated with the nonce:

        <pubkey> <nonce> DROP CHECKSIG

    Note: this easily extends to multisig programs: each individual pubkey
    is annotated with `<nonce> DROP` opcode.

6. Bob sends receiver to a sender Sandy.
7. Sandy makes payment to that address.
8. Alice scans all outputs on blockchain, trying to check if any given public 
   key is derived from the xpub using an associated nonce.
9. Network cannot link two outputs to the same xpub because nonces are random and
   no one except Alice and Bob has xpub that contains "derivation key" entropy used
   to produce child keys.

Note: it is possible to save bandwidth by using 64-bit nonces instead of 128-bit ones at a slightly higher risk of collisions (still, negligible in practice). Nonce collisions link two outputs to the same account and may make accounting slightly more complicated by requiring linking through reference data (which could be a requirement anyway).






## Alternatives considered

### 1. Encryption of the exported disclosure

Possible options:

1. Encrypting to a Core-generated "Import Key" (as in this spec).
2. Not encrypting and relying on security of the transmission channels.

Arguments for encryption:

1. Encrypting disclosure ensures that data cannot be intercepted between two Cores without assuming that Cores connect directly to each other.
2. Encryption allows concrete auditability of data access and policies: import keys can be signed by long-term keys to provide a delegation chain. This is more important for manual handling of disclosures where a user of one Core exports and transmits it to other people who import it to the target Core.

Arguments against encryption:

1. Recipient must generate a receiving key first, before the exporter can create a disclosure.
2. We may figure a more generalized way for encryption of arbitrary data between Cores, and that would be a custom use of it.

### 2. Inline data VS out of band data

Possible options:

1. Automatically split data for inline (INL) and out of band (OOB) pieces.
2. Keep these explicitly separate.

We choose to keep these fields separate since the inline data has special features and considerations:

* Inline data does not require out-of-band transmission. Therefore, nodes can receive the data from raw blockchain data w/o setting up additional channels.
* Inline data has limited size: around 3-4 Kb.
* Inline data requires reveal of the numeric amount, so these two fields cannot be indepdently disclosed.

In the present specification we omit support for inline data entirely for simplicity of the interface and
intend to introduce it as an additional feature that allows applications to optimize bandwidth usage.



## Swagger specification

    ---
    swagger: '2.0'

    [...]

    definitions:

      Receiver:
        type: object
        required:
          - control_program
          - expires_at
        properties:
          output_version:
            type: integer
            description: The version of the accepted output. Default is 1,
              which means a non-confidential (pre-CA) output is expected.
          control_program:
            type: string
            description: The raw hex of the control program.
          expires_at:
            type: string
            description: An RFC3339 timestamp indicating when the receiver expires.
          dek:
            type: string
            description: The raw hex of the data encryption key
          aek:
            type: string
            description: The raw hex of the asset ID encryption key
          vek:
            type: string
            description: The raw hex of the value encryption key

      TransactionTemplate:
        type: object
        required:
          - version
          - finalized
          - balanced
          - signed
          - entries
          - payloads
        properties:
          version:
            type: integer
            description: Version of the transaction template format. Current version is 2.
          finalized:
            type: boolean
            description: Whether the transacton ID is fixed; that is,
              at least one signature covers the entire transaction.
          balanced:
            type: boolean
            description: Whether no placeholder values left and transaction
              is ready to be signed.
          signed:
            type: boolean
            description: Whether the transaction is fully signed and can be published.
          entries:
            type: array
            items:
              $ref: '#/definitions/TransactionTemplateEntry'
          payloads:
            type: array
            items:
              $ref: '#/definitions/TransactionTemplatePayload'

      TransactionTemplateEntry:
        type: object
        description: There are several types of actions for building transactions. Since Swagger 2.0
          does not allow for polymorphic types, the individual properties are not listed here.
          Please refer to the definitions of:
          TxHeaderTemplate,
          Mux1Template,
          Issuance1Template,
          Input1Template,
          Output1Template,
          Retirement1Template,
          Mux2Template,
          Issuance2Template,
          Input2Template,
          Output2Template,
          Retirement2Template,
          InputPlaceholder,
          OutputPlaceholder.

      

      TransactionTemplatePayload:
        type: object
        required:
          - version
          - id
          - ciphertext
        properties:
          version:
            type: integer
            description: Version of the payload encryption format. Current version is 1.
          id:
            type: string
            description: Unique hex identifier of the payload.
          ciphertext:
            type: string
            description: Hex-encoded encrypted payload. See TransactionTemplatePayloadContent

      TransactionTemplatePayloadContent:
        description: Payload encapsulates subset of transaction with private signing instructions.
          When transaction is being fully signed, owner of a payload verifies the resulting transaction
          against this subset and uses private signing instruction to generate necessary witness data.
        type: object
        required:
          - version
          - entries
        properties:
          version:
            type: integer
            description: Version of the transaction template format. Current version is 2.
          entries:
            type: array
            items:
              $ref: '#/definitions/TransactionTemplateEntry'






