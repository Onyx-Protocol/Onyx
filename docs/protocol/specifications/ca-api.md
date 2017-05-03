# Confidential Assets API

* [Introduction](#introduction)
* [Goals](#goals)
* [Addressed problems](#addressed-problems)
* [Overview](#overview)
* [Data structures](#data-structures)
* [Actions](#actions)
* [Encryption](#encryption)
* [Compatibility](#compatibility)
* [Alternatives considered](#alternatives-considered)

## Introduction

New design for transaction builder that focuses on privacy.

## Goals

1. Native support for transaction entries.
2. Privacy-preserving transaction signing
3. Confidential assets support.
4. Support for constructing witnesses for custom control and issuance programs.
5. Built-in support for encrypting payload inside and outside the range proof.

## Addressed problems

1. Linkability of inputs and outputs in multi-party transfers.
2. Asset range proofs for outputs with asset ID that’s missing among inputs.
3. Automated verification for additional rounds of finalization.
4. Future extensibility to custom signing schemes.
5. Privacy of metadata (such as signing instructions, account IDs etc) for shared tx templates.


## Overview

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
        control_program: "fa90e031...",   // control program
        rek: "fae0fab..."                 // record-encryption key, from which blinding factors are derived
    }


### Transaction template

Transaction template contains transaction entries and additional data that helps multiple parties to cooperatively create the transaction.

    {
        version: 2,  // version of the transaction template format
        finalized: false, // set to true after first `client.transactions.sign()` call
        balanced:  false, // set when no placeholder inputs/outputs left
        signed:    false, // set when all signatures and proofs are provided
        transaction: [
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
                type: "output2",
                asset_id: "USD",
                amount:   140,
                placeholder: true,
            },
        ],
        payloads: [
            {
                version: 1,  // version of the payload encryption
                id: <payload-id>,
                private: ENCRYPTED{
                    version: 2, // version of the transaction template
                    transaction: [
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


### Disclosure

#### Encrypted Disclosure

    {
        version: 1,                          # version of the encrypted disclosure
        import_key: "fe9af9bc3923...",       # IK pubkey
        selector:   "589af9b1a730...",       # blinding selector
        payload:    "cc9e012f7a8f99ea9b...", # encrypted disclosure blob
    }

#### Unencrypted Disclosure

    {
        version: 1,                          # version of the plaintext disclosure
        items: [
            {
                scope: "output"/"tx"/"account",  # 
                fields: 
            },
        ]
        
    }


## Actions

### Build

1. Prepares a partial transaction with data from the `base_transaction`:
    1. Reserves unspent outputs.
    2. Adds necessary inputs spending reserved unspent outputs.
    3. Adds necessary issuances.
    4. Prepares `signing_instructions` for each input and issuance.
    5. Creates and encrypts new outputs, adds VRPs (but not ARPs).
    6. Computes the `excess` blinding factor.
    7. Sets mintime to current time: `mintime = currentTime`.
    8. Sets maxtime to expiration time: `maxtime = expirationTime`.
    9. Prepares ARP instructions for each output: `asset_range_proof_instructions: {asset_id:..., factor:..., [input_id:..., input_factor:...]}`.
        * Input ID and input's blinding factor are included if such input is known.
    10. For each output in `base_transaction` where `asset_range_proof_instructions` are specified and partial tx has an input with the matching asset ID:
        1. Copy that output to the partial transaction.
        2. Add `input_id/input_factor` to `asset_range_proof_instructions` for that copy.
        3. Strip `asset_range_proof_instructions` from that output in `base_transaction`.
    11. Encrypts the partial transaction:
        1. Generates unique uniform `payload-id`.
        2. Derives encryption key from Chain Core's master key: `EK = SHA3(master || SHA3(payload-id))`.
        3. Encrypts the partial transaction and wraps it with `{payload_id:, blob:}` structure.
2. Merges the data from partial transaction with the existing template (`base_transaction`):
    1. Merges tx.mintime with partial's mintime: `mintime = MAX(basetx.mintime, partialtx.mintime)`.
    2. Merges tx.maxtime with partial's maxtime: `maxtime = MIN(basetx.maxtime, partialtx.maxtime)`.
    3. Adds outputs, stripping `asset_range_proof_instructions` from those outputs where `input_id/input_factor` is specified.
    4. Adds `{payload_id:, blob:}` to the list of payloads in the `base_transaction`.
    5. Adds `excess` from the partial transaction to the `excess` value in the MUX entry in the `base_transaction`.
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
        1. Derive encryption key from Chain Core's master key: `EK = SHA3(master || SHA3(payload-id))`.
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

1. Core loads the `Root Import Key` — a ChainKD xpub.
2. Core generates a unique blinding selector `b` (256 bits of entropy).
3. Core derives one-time import pubkey: `IK = HD(RIKxpub, b)`.
4. Core returns a pair of the import key and a blinding selector `IK,b` (total 64 bytes).


### Export Disclosure

Bob receives `import_key` from Alice and uses it to build a disclosure object:

    disclosure = client.disclosures.build(import_key: import_key) do |d|
      d.disclose_output(output_id: "fa0e8fb0ad...",          fields: {asset_id: true, amount: true, data: true})
      d.disclose_transaction(transaction_id: "57ad0fea9...", fields: {asset_id: true, amount: true, data: true})
      d.disclose_account(account_alias: "bob",               fields: {asset_id: true, amount: true, data: true})
    end
    disclosure_serialized = disclosure.to_json

The resulting object is encrypted to an `import_key`, contains minimal metadata needed for decryption 
and can be safely transmitted to the receiving Core for import.

* `disclose_output` — adds proofs and decryption keys for a single output.
* `disclose_transaction` — adds proofs and decryption keys for each output in the transaction (omits outputs not decrypted by this Core).
* `disclose_account` — adds account xpubs, derivation path and root decryption keys for tracking all outputs for a given account.


### Import Disclosure

Alice receives `disclosure` object from Bob and attempts to import in the Core:

    disclosure_description = client.disclosures.import(disclosure)

Alice can decrypt and inspect the disclosure parameters without importing:

    disclosure_description = client.disclosures.decrypt(disclosure)
    disclosure_description.fields.asset_id # => true/false
    disclosure_description.scope           # => 'output'/'transaction_id'/'account'

TBD: querying all stored disclosures too?


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

For each access token there is a separate **access key** `AK`:

    AK = SHAKE128(RK || access_token, 32)

For each field, there's a separate root key:

    RDEK = SHAKE128("RDEK" || RCK, 32)   # Root Data Encryption Key
    RAEK = SHAKE128("RAEK" || RCK, 32)   # Root Asset ID Encryption Key
    RVEK = SHAKE128("RVEK" || RCK, 32)   # Root Value Encryption Key

For each account a deterministic account selector is made that's used to generate per-account keys:
    
    account_selector = m,n,xpub1,xpub2,...

    ADEK = SHAKE128("ADEK" || RDEK || accselector, 32)   # Account Data Encryption Key
    AAEK = SHAKE128("AAEK" || RAEK || accselector, 32)   # Account Asset ID Encryption Key
    AVEK = SHAKE128("AVEK" || RVEK || accselector, 32)   # Account Value Encryption Key

For each output, a key is derived using control program as a selector:

    ODEK = SHAKE128("ADEK" || ADEK || ctrlprog, 32)   # Output Data Encryption Key
    OAEK = SHAKE128("AAEK" || AAEK || ctrlprog, 32)   # Output Asset ID Encryption Key
    OVEK = SHAKE128("AVEK" || AVEK || ctrlprog, 32)   # Output Value Encryption Key


TBD: figure how to encrypt with unique key a retirement entry.
TBD: tweak the scheme to allow 2D derivation.


### Reference data encryption

TBD: split data in 2 parts: for the value range proof and the remainder. 

TBD: Encrypt the remainder and supply it separately from range proof.


### Payload encryption

#### Encrypt template payload

1. Core creates a master key `MK` upon initialization (shared by all transactions).
2. For each transaction payload, encryption key is derived: `EK = SHA3(MK || SHA3(payload-id))`
3. Encode the `payload` as a JSON document.
4. Encrypt-then-MAC:
    1. Compute keystream of the same length as plaintext: `keystream = SHAKE128(EK, len(payload))`
    2. Encrypt the payload with the keystream: `ct = payload XOR keystream`.
    3. Compute MAC on the ciphertext `ct`: `mac = SHAKE128(ct || EK, 32)`.
    4. Append MAC to the ciphertext: `ct’ = ct || mac`.

#### Decrypt template payload

1. For each transaction payload, encryption key is derived: `EK = SHA3(MK || SHA3(payload-id))`
2. Split ciphertext into raw ciphertext and MAC (last 32 bytes): `ct, mac`.
3. Compute MAC on the ciphertext `ct`: `mac’ = SHAKE128(ct || EK, 32)`.
4. Compare in constant time `mac’ == mac`. If not equal, return nil.
5. Compute keystream of the same length as ciphertext: `keystream = SHAKE128(EK, len(ciphertext))`
6. Decrypt the payload by XORing keystream with the ciphertext: `payload = ct XOR keystream`.
7. Return `payload`.


## Compatibility

Current SDK uses `signer.sign()` method to sign a partial transaction. We can keep this behavior and introduce an additional API:

    HSMSigner.sign() - existing behavior as is
    Client.sign()    - verifies payload and signs TXSIGHASH-based predicate via Client.signer.sign(txhash instead of checkpredicate)

When users upgrade to a new Chain Core, the tx template is changed, but the behavior of the application remains the same. Then, they can smoothly transition to a new API usage:

1. Configure Client.signer instead of a standalone HSMSigner instance.
2. Introduce a second round of tx template exchange to do `Client.sign` after other parties have participated.
3. If using confidential amounts, new signing mechanism is required.




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

### 2. 



