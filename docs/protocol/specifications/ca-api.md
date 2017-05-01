# Confidential Assets API

* [Introduction](#introduction)
* [Goals](#goals)
* [Addressed problems](#addressed-problems)
* [Overview](#overview)
* [Data structures](#data-structures)
* [Actions](#actions)
* [Encryption](#encryption)
* [Compatibility](#compatibility)

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

When transaction is built, it must be finalized, signed and submitted. 
Flag `finalize` is true by default, so it can be omitted.

    tx = client.transactions.sign(tx, finalize: true)
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

When no one else is adding to the transaction, it must be finalized and signed:

    tx3 = client.transactions.sign(tx2, finalize: true)

Each party has to finalize the transaction, so it becomes fully signed.

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
        finalized: false, // set to true after `sign(tx, finalize:true)`
        balanced:  false, // set when no placeholder inputs/outputs left
        signed:    false, // set when all signatures and proofs are provided
        transaction: [
            {
                type:    "txheader",
                version:  2,  // core protocol version
                mintime:  X,
                maxtime:  Y,
                txid:     "0fa8127a9fe8d89b12..." // only added after `sign(tx,finalize:true)`
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


TBD: how delegations to nested programs stack up witnesses, with buildcode composing all calls to produce the witness.

TBD: how nested clauses are handled







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

1. If `sign(tx, finalize:false)`, does the following checks, but signs not TXSIGHASH, but CHECKOUTPUT-based predicate composed out of the encrypted partial transaction.
2. If `finalize:true` (which is default), then:
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




## Encryption

### Key derivation

Core manages two hierarchies of keys for encrypting transaction templates and asset amounts.
The first hierarchy is used to protect transaction building process and is not tied to any
account. The second hierarchy is deliberately tied to accounts and control programs in order
to simplify indexing and tracking of the payments.

Core is initialized with a **root secret key** `RK` which is stored in the Core's DB.

    RK = random

For each access token there is a separate **access key** `AK`:

    AK = SHAKE128(RK || access_token, 32)

For each account

* Core derives per-output encryption keys using output control programs
* Access is restricted within Core by acess tokens.
* Lite Cores allow more precise control over private data (enc keys, account xpubs) by allowing many Lite Cores handle encryption keys while one bigger Core only verifies and slices the blockchain.

TBD: figure how to encrypt with unique key a retirement entry.

### Reference data encryption

TBD: split data in 2 parts: for the value range proof and the remainder. 
Encrypt the remainder and 


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

    HSMSigner.sign()             - existing behavior as is
    Client.sign(finalize: false) - verifies payload and signs CHECKPREDICATE-based predicate via Client.signer.sign() 
    Client.sign(finalize: true)  - verifies payload and signs TXSIGHASH-based predicate via Client.signer.sign()

When users upgrade to a new Chain Core, the tx template is changed, but the behavior of the application remains the same. Then, they can smoothly transition to a new API usage:

1. Configure Client.signer instead of a standalone HSMSigner instance.
2. Introduce a second round of tx template exchange to do `Client.sign` after other parties have participated.
3. If using confidential amounts, new signing mechanism is required.



