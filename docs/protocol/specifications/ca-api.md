# Confidential Assets API

* [Introduction](#introduction)
* [Roadmap](#roadmap)
* [Actions](#actions)
  * [Create Receiver](#create-receiver)
  * [Build Transaction](#build-transaction)
  * [Finalize Transaction](#finalize-transaction)
  * [Sign Transaction](#sign-transaction)
  * [Import Asset](#import-asset)
  * [List Asset Imports](#list-asset-imports)
  * [Create Issuance Key Spec](#create-issuance-key-spec)
  * [Create Disclosure Import Key](#create-disclosure-import-key)
  * [Export Disclosure](#export-disclosure)
  * [Import Disclosure](#import-disclosure)
* [Data structures](#data-structures)
  * [Transaction Template](#transaction-template)
  * [Signing Instructions](#signing-instructions)
  * [Entry Data](#entry-data)
  * [Disclosure](#disclosure)
* [Annotations](#annotations)
* [Encryption](#encryption)
  * [Key derivation](#key-derivation)
  * [Entry Shuffling](#entry-shuffling)
  * [Disclosure Import key](#disclosure-import-key)
  * [Reference data encryption](#reference-data-encryption)
  * [Payload encryption](#payload-encryption)
  * [Packet Encryption](#packet-encryption)
* [Examples](#examples)
* [Discussion](#discussion)
* [Swagger definitions](#swagger-definitions)


## Introduction

New design for transaction builder and related APIs that focuses on privacy.

### Goals

1. Native support for transaction entries.
2. Privacy-preserving transaction signing.
3. Support for confidential assets.
4. Support for constructing witnesses for custom control and issuance programs.
5. Built-in support for encrypting payload inside and outside the range proof.

### Addressed problems

1. Linkability of inputs and outputs in multi-party transfers.
2. Asset range proofs for outputs with asset ID that’s missing among inputs.
3. Automated verification for additional rounds of finalization.
4. Future extensibility to custom signing schemes.
5. Privacy of metadata (such as signing instructions, account IDs etc) for shared tx templates.

### Upgrading to confidential assets

All assets are encrypted by default. Application can opt out of encryption of asset IDs, amounts or reference data per output (e.g. to satisfy a contract that introspects these values).

Change outputs for accounts are always encrypted.

Existing unencrypted unspent outputs are spent first and upgraded to confidential assets automatically (using a combination of `Retirement` and `Upgrade` entries inserted by Chain Core automatically).

New receivers introduce versioning and use version 2 with relevant encryption keys.

## Roadmap

#### Phase 1: non-confidential transaction builder

* New receiver format v2 (without encryption keys).
* New transaction builder with necessary fields for non-confidential txs.

#### Phase 2: confidential transaction builder

* Encryption keys in v2 receivers.
* Transaction builder with support for encryption of assetids, amounts and reference data.

#### Phase 3: issuance specs

* Create an issuance key
* Export/import issuance pubkey ("issuance spec")

#### Phase 4: disclosures

* Generate a disclosure import key
* Configure and export disclosure to an import key
* Import a disclosure and use it to annotate transactions.














## Actions

### Create Receiver

#### Request

    POST /create-account-receiver

    [
      {
        "account_id":       <string>,
        "account_alias":    <string>,
        "expires_at":       <string:RFC3339>,
        "confidential": {
          "reference_data": <boolean>,
          "asset_id":       <boolean>,
          "amount":         <boolean>
        }
      }
    ]

One API call allows creating multiple individual receivers. SDK implementations may restrict usage to one receiver at a time and be expanded with batch support later.

Account is specified via `account_id` or `account_alias` (one and only one of them must be specified).

Expiration date `expires_at` is optional. Defaults to 30 days in the future. It is not possible to create non-expiring receivers via this API call.

Parameter `confidential` indicates which fields of the output should be encrypted by the user of the receiver. Parameter `confidential` and all its fields are optional. Default value for each field is `true`. If the value for a given field is `true`, a corresponding encryption key is generated and returned as a part of the receiver (see **Response**).

#### Response

    [
      {
        version:         2,
        control_program: <string:hex>,
        expires_at:      <string:RFC3339>,
        dek:             <string:hex>,
        aek:             <string:hex>,
        vek:             <string:hex>
      }
    ]

Receivers are versioned. Legacy receivers have implicit version 1. New receivers have version 2.

Raw `control_program` is computed as always, from account’s xpubs.

Expiration date `expires_at` is not necessarily the same as requested. It can be shortened by Chain Core.

Data-encryption key `dek` is included if `confidential.reference_data` was set to `true`.

Data-encryption key `aek` is included if `confidential.asset_id` was set to `true`.

Data-encryption key `vek` is included if `confidential.amount` was set to `true`.

See [Key Derivation](#key-derivation) section for details.

#### Ruby

    receiver = client.accounts.create_receiver(
      account_alias:   'my-account',
      expires_at:      '2017-01-01T00:00:00Z', # may also accept Time, Date and DateTime objects
      confidential: {
        asset_id:       false,
        amount:         true,
        reference_data: true
      }
    )

#### JS

    client.accounts.create_receiver({
      accountAlias:    'my-account',
      expiresAt:       '2017-01-01T00:00:00Z', // may also take Date object
      confidential: {
        asset_id:       false,
        amount:         true,
        reference_data: true
      }
    }).then(receiver => {
      ...
    })

#### Java

    Receiver r = new Account.ReceiverBuilder()
      .setAccountAlias("my-account")
      .setExpiresAt("2017-01-01T00:00:00Z") // also take native time and date objects?
      .setConfidentialAssetID(false)
      .setConfidentialAmount(true)
      .setConfidentialReferenceData(true)
      .create(client);


### Build Transaction

#### Request

    POST /build-transaction

    {
      version: 2,
      ttl:             <number:milliseconds>,
      actions: [
        {
          type: ("issue"                        |
                 "spend_account"                |
                 "spend_account_unspent_output" |
                 "control_account"              |
                 "control_receiver"             |
                 "control_program"              |
                 "retire")
        },

        // "issue" action:
        {
          type:           "issue",
          asset_id:       <string>,
          asset_alias:    <string>,
          amount:         <integer>,
          reference_data: <string>,
          confidential: {                  // new field
            asset_id:       <boolean>,
            amount:         <boolean>,
            reference_data: <boolean>
          },
          issuance_choices: [              // new field
            {
              asset_id:       <string>,
              asset_alias:    <string>,
            }
          ]
        },

        // "retire" action:
        {
          type:           "retire",        // new type
          asset_id:       <string>,
          asset_alias:    <string>,
          amount:         <integer>,
          reference_data: <string>,
          confidential: {
            asset_id:       <boolean>,
            amount:         <boolean>,
            reference_data: <boolean>
          }
        },
      ]
    }

New optional `version` is set to 2 for new SDKs, so Chain Core may support legacy SDK that use implicit version 1. 
The following updates apply to requests with `version=2`.

New action type `retire` is introduced that works like `control_program`, but without the `program` parameter.

Actions `issue`, `control_account`, `control_program`, `retire` have an optional `confidential` parameter with boolean fields `asset_id`, `amount`, `reference_data`. Confidentiality of `control_receiver` action is determined by the receiver, not by the user. Actions `spend_*` can only use `confidential.reference_data` field since confidentiality of asset ID and amount is already determined in the corresponding unspent outputs.

Parameter `confidential` indicates which fields of the output should be encrypted by the user of the receiver. Parameter `confidential` and all its fields are optional. Default value for each field is `true`. If the value for a given field is `true`, a corresponding encryption key is generated and used to encrypt the specified data.

For issuance actions, if `confidential.asset_id` is set to `true`, an array `issuance_choices` can be used. 
`issuance_choices` specify asset IDs (by ID or alias) among which the actually issued asset ID is hidden.
`issuance_choices` is allowed (but not required) to include the asset ID being issued. User can specify only asset IDs that 
they are an issuer of (and can sign each individual issuance program) or the asset IDs imported via [Import Issuance Key](#import-issuance-key) API.
Build fails if `issuance_choices` are specified while `confidential.asset_id` is set to `false`.

If the transfer is made to an account, it is encrypted with the keys associated with this account.

If the transfer is made to an arbitrary control program, it is encrypted with the keys derived from the root key and that control program.

If the transfer is made to a receiver, the output or retirement is encrypted with keys specified by the receiver. If some keys are omitted,
then no encryption takes place.

See [Key Derivation](#key-derivation) and [CA specification](ca.md) section for details.

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
3. Shuffle entries according to [Entry Shuffling](#entry-shuffling) specification.



#### Response

    {
      version: 2,
      entries: [
        ...
      ]
    }

See [Transaction template](#transaction-template) for the data structure description.

#### Ruby

    chain.transactions.build do |b|
        b.base_transaction tx 
        b.issue                ..., confidential: {reference_data: true, asset_id: true, amount: true}, issuance_choices: [
          {asset_id:    "a8fd0177d1..."},
          {asset_alias: "gold"}
        ]
        b.spend_from_account   ..., confidential: {reference_data: true}
        b.control_with_account ..., confidential: {reference_data: true, asset_id: true, amount: true}
        b.retire               ..., confidential: {reference_data: true, asset_id: true, amount: true}
    end

#### JS

    client.transactions.build(builder => {
       builder.issue({
        assetAlias: 'gold',
        amount: 10,
        confidential: {
          reference_data: true, 
          asset_id: true, 
          amount: true
        }, 
        issuance_choices: [
          {asset_id:    "a8fd0177d1..."},
          {asset_alias: "gold"}
        ]
      })
      builder.spendFromAccount({
        accountAlias: 'alice',
        assetAlias: 'gold',
        amount: 10,
        confidential: {reference_data: true}
      })
      builder.controlWithAccount({
        accountAlias: 'bob',
        assetAlias: 'gold',
        amount: 10,
        confidential: {reference_data: true, asset_id: true, amount: true}
      })
    })
    .then(payment => client.sign(payment))
    .then(signed => client.transactions.submit(signed))


#### Java

    Transaction.Template payment = new Transaction.Builder()
      .addAction(new Transaction.Action.Issue()
        .setAssetAlias("gold")
        .setAmount(10)
        .setConfidentialAssetID(true)
        .addIssuanceChoiceAssetID("a8fd0177d1...")
        .addIssuanceChoiceAssetAlias("gold")
      ).addAction(new Transaction.Action.SpendFromAccount()
        .setAccountAlias("alice")
        .setAssetAlias("gold")
        .setAmount(10)
        .setConfidentialReferenceData(true)
      ).addAction(new Transaction.Action.ControlWithAccount()
        .setAccountAlias("bob")
        .setAssetAlias("gold")
        .setAmount(10)
        .setConfidentialAssetID(false)
        .setConfidentialAmount(true)
        .setConfidentialReferenceData(true)
      ).build(client);



### Finalize Transaction

Finalization creates missing range proofs, shuffles outputs and 
verifies that transaction is balanced and ready to be signed.

Users normally do not explicitly finalize the transaction.
Finalization happens in the first half of the `sign` SDK function. 
The second half is sending the finalized transaction to the HSM for actual signing.
See [Sign transaction](#sign-transaction) for details.

#### Request

    POST /finalize-transaction

    {
      ... // transaction template
    }

See [Transaction template](#transaction-template) for the data structure description.

Process:

1. Send transaction template to Core to decrypt and verify [signing instructions](#signing-instructions):
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
        2. Send `signing_instructions` for inputs/issuances to the client.
2. Client receives [signing instructions](#signing-instructions) and sends them to the HSM signer:
    1. HSM signer signs using the signing instructions.
    2. Client places the signature over TXSIGHASH in the entry inside the transaction.

#### Response

    {
      finalized: true,
      ... // the rest of tx template with the updated signing instructions
    }

See [Transaction template](#transaction-template) for the data structure description.

#### Ruby

    // Both actions happen within `chain.transactions.sign(tx)`
    tx = chain.transactions.finalize(tx)
    hsmsigner.sign(tx)

#### JS

    // Both actions happen within `chain.transactions.sign(tx)`
    chain.transactions.finalize(tx).then(
      finaltx => hsmsigner.sign(finaltx)
    )

#### Java

    // Both actions happen within `client.signTransaction(tx)`
    Transaction.Template finaltx = client.finalize(tx);
    HsmSigner.sign(finaltx);




### Sign Transaction

Transaction signing is a function in the SDK that encapsulates two API requests:

1. [Finalize Transaction](#finalize-transaction): a request to Chain Core that verifies that the transaction is balanced, computes missing range proofs and shuffles the outputs.
2. HSM Sign: a request to the HSM that signs the inputs to authorize the transaction.

HSM signer returns [Transaction template](#transaction-template) with some inputs having [signing instructions](#signing-instructions) replaced with actual signatures.

#### Ruby

    signed_tx = chain.transactions.sign(tx)
    chain.transactions.submit(signed_tx)

#### JS

    chain.transactions.sign(tx).then(
      signedTx => chain.transactions.submit(signedTx)
    )

#### Java

    Transaction.Template signedTx = client.signTransaction(tx);
    Transaction.submit(client, signedTx);






### Import Asset

#### Request

    POST /import-asset

    {
      alias:               <string>,
      asset_id:            <string:hex>,
      issuance_key_spec: {                  // optional
        version:           1,
        asset_id:          <string:hex>,
        initial_block_id:  <string:hex>,
        issuance_program:  <string:hex>,
        reference_data:    <string:hex>,
        arguments:        [<string:hex>,...],
        issuance_key:      <string:hex>
      }
    }

Imports externally-defined `asset_id` and associates with a given `alias`.
`alias` is optional if the `asset_id` already exists in Chain Core with some alias.

If the optional `issuance_key_spec` is specified, stores it together with the given `asset_id`.

Validation:

1. `issuance_key_spec.version` must be set to 1.
2. `issuance_key_spec.asset_id` must be equal to `asset_id`.
3. `issuance_key_spec.asset_id` must be equal to asset ID computed from the defining `initial_block_id`, `issuance_program` and `reference_data`.
4. `arguments` must satisfy `issuance_program` for any transaction provided the issuance choice specifies the given `issuance_key`. See [VERIFYISSUANCEKEY](vm2.md#verifyissuancekey).

#### Response

If validation succeeded and asset ID was imported, returns the import details as submitted by the user.

    {
      alias:             <string>,
      asset_id:          <string:hex>,
      issuance_key_spec: {...}
    }


#### Ruby

    chain.assets.import_issuance_key(
      alias: 'acme_common',
      issuance_key_spec: {
        version:          1,
        asset_id:         'fa0bd0ad1241...',
        initial_block_id: 'a9f03712fad1...',
        issuance_program: '5604afe9baf0...',
        reference_data:   '9af9f9839102...',
        arguments: ['ad3703...', 'fe8a7210...'],
        issuance_key:     '048af9bd9e01...',
      }
    )

#### JS

    client.assets.import_issuance_key({
      alias: 'acme_common',
      issuance_key_spec: {
        version:          1,
        asset_id:         'fa0bd0ad1241...',
        initial_block_id: 'a9f03712fad1...',
        issuance_program: '5604afe9baf0...',
        reference_data:   '9af9f9839102...',
        arguments: ['ad3703...', 'fe8a7210...'],
        issuance_key:     '048af9bd9e01...',
      }
    })

#### Java

    new Asset.IssuanceKeyImport()
      .setAlias("acme_common")
      .setIssuanceKeySpec(spec) // opaque object from `Create Issuance Key Spec`
      .create(client);



### List Asset Imports

TBD: need to figure if we need to support queries. We need some minimal API to show imported assets and their issuance key specs in the dashboard.


### Create Issuance Key Spec

#### Request

    POST /create-issuance-key

    {
      asset_alias: <string>,
      asset_id:    <string:hex>,
    }

Requests creation of an issuance key for a given asset ID (either `asset_alias` or `asset_id` must be specified).

Chain Core returns signing instructions to generate valid a issuance key spec (that specifies reusable `arguments` for the `issuance_program`).

1. If the issuance program is defined by **1 public key**, that key is used without modifications as an `issuance_key`.
2. If the issuance program is defined by **N public keys** with the quorum of the **same size N**:
  1. All public keys are decoded as EC [points](ca.md#point).
  2. Each public key is hashed via [ScalarHash](ca.md#scalarhash) to form a per-key _shielding scalar_.
  3. Public key points are multiplied by their shielding scalars and added together:
  
        issuance_key = Sum[ScalarHash(P_i)·P_i, for i = 1..N]

  4. The resulting key is encoded as a standard EdDSA public key ([RFC8032](https://tools.ietf.org/html/rfc8032)).
3. If the issuance program is defined by **M-of-N multisig condition** or an **arbitrary issuance program**, API returns an error. Support for threshold issuance keys or more complex configurations may be introduced in future versions of the SDK.

The result of executing `signing_instructions` is a valid list of `arguments` satisfying the `issuance_program` in the context of an [asset issuance choice](blockchain.md#asset-issuance-choice).

The complete issuance spec must be imported ([Import Asset](#import-asset)) after signing to be usable by issuer themselves in their confidential issuances.


#### Response

    {
      version:          1,
      asset_id:         'fa0bd0ad1241...',
      initial_block_id: 'a9f03712fad1...',
      issuance_program: '5604afe9baf0...',
      reference_data:   '9af9f9839102...',
      signing_instructions: {
        ...
      },
      issuance_key:     '048af9bd9e01...',
    }

See [signing instructions](#signing-instructions) for format details.


#### Ruby

    spec = chain.assets.create_issuance_key(
      alias: 'acme_common'
    )

    chain.sign_issuance_key_spec(spec)

    chain.assets.import_issuance_key(
      issuance_key_spec: spec
    )

#### JS

    client.assets.create_issuance_key({
      alias: 'acme_common'
    }).then(
      spec => chain.sign_issuance_key_spec(spec)
    )

    ...

    client.assets.import_issuance_key({
      issuance_key_spec: spec
    })

#### Java

    spec = new Asset.IssuanceKeySpec()
      .setAlias("acme_common")
      .create(client);

    client.sign(spec)

    new Asset.IssuanceKeyImport()
      .setIssuanceKeySpec(spec)
      .create(client);




### Create Disclosure Import Key

The Core that needs to import a disclosure must first
generate a recipient key to ensure that the document is 
encrypted in-transit (without assuming direct secure connection between two Cores).

See [Disclosure Import key](#disclosure-import-key) for details on key derivation and encoding.

#### Request

    POST /disclosures/create-import-key

    {}

#### Response

    {
      type:     "disclosureimportkey1",
      key:      <string:hex>,  // 32-byte public key
      selector: <string:hex>   // 32-byte selector
    }  

#### Ruby

    import_key = client.disclosures.create_import_key()
    import_key_serialized = import_key.to_json

#### JS

    client.disclosures.create_import_key()

#### Java

    Disclosure.ImportKey importKey = Disclosure.createImportKey(client)


### Export Disclosure

Bob receives a [disclosure import key](#disclosure-import-key) from Alice and uses it to build a disclosure object.

The resulting object is encrypted to an `import_key`, contains minimal metadata needed for decryption
and can be safely transmitted to the receiving Core for import.

* `disclose_entry` — adds proofs and decryption keys for a single output/retirement/issuance.
* `disclose_transaction` — adds proofs and decryption keys for each decryptable entry in the transaction (omits entries not decrypted by this Core).

#### Request

    POST /disclosures/export

    {
      import_key: {...}, // disclosure import key structure
      disclosures: [
        {
          kind:              ("entry" | "transaction"),
          entry_id:          <string:hex>,               // if kind == "entry"
          transaction_id:    <string:hex>,               // if kind == "transaction"
          asset_id:          <boolean>,
          amount:            <boolean>,
          reference_data:    <boolean>,
        }
      ]
    }

#### Response

    {
        type:         "encdisclosure1",
        sender_key:   <string:hex>,
        selector:     <string:hex>,
        ciphertext:   <string:hex>
    }

See [Disclosure](#disclosure) for details of the encrypted and cleartext structures.

#### Ruby

    encrypted_disclosure = client.disclosures.build(import_key: {...}) do |d|
      d.disclose_entry(
        entry_id:       "fa0e8fb0ad...",
        asset_id:        true,
        amount:          true,
        reference_data:  true
      )
      d.disclose_transaction(
        transaction_id: "57ad0fea9...",
        asset_id:        true,
        amount:          true,
        reference_data:  true
      )
    end

#### JS

    client.disclosures.build({import_key: {...}}, builder => {
      builder.disclose_entry({
        entryID:        "fa0e8fb0ad...",
        asset_id:        true,
        amount:          true,
        reference_data:  true
      })
      builder.disclose_transaction({
        transaction_id: "57ad0fea9...",
        asset_id:        true,
        amount:          true,
        reference_data:  true
      })
    })
    .then(encdisclosure => ...)

#### Java

    Disclosure.EncryptedDisclosure disclosure = new Disclosure.Builder()
      .addEntryDisclosure(new Disclosure.EntryDisclosureRequest()
        .setEntryID("fa0e8fb0ad...")
        .setAssetID(true)
        .setAmount(true)
        .setReferenceData(true)
      ).addTransactionDisclosure(new Disclosure.TransactionDisclosureRequest()
        .setTransactionID("57ad0fea9...")
        .setAssetID(true)
        .setAmount(true)
        .setReferenceData(true)
      ).build(client);





### Import Disclosure

Alice receives `disclosure` object from Bob and attempts to import in the Core:

    cleartext_disclosure = client.disclosures.import(
      alias: "Bob's transaction",
      encrypted_disclosure: {...}
    )

TBD: Should we support querying all stored disclosures too?

#### Request

    POST /disclosures/import

    {
      alias: <string>,
      encrypted_disclosure: {
        type:         "encdisclosure1",
        sender_key:   <string:hex>,
        selector:     <string:hex>,
        ciphertext:   <string:hex>
      }
    }

#### Response

    {
        alias: <string>,      # alias as specified during import
        type: "disclosure1",  # version of the cleartext disclosure
        items: [
            {
                scope: "output"/"retirement"/"issuance",
                ...
            },
        ]
    }

See [Cleartext Disclosure](#cleartext-disclosure) for details.

#### Ruby

    cleartext_disclosure = client.disclosures.import(
      alias: "Bob's transaction",
      encrypted_disclosure: {...}
    )

#### JS

    client.disclosures.import({
       alias: "Bob's transaction",
       encrypted_disclosure: {...}
    })
    .then(cleartextdisclosure => ...)

#### Java

    Disclosure disclosure = new Disclosure.Import(client, "Bob's transaction", encdisclosure)



### Decrypt Disclosure

Alice can decrypt and inspect the disclosure without importing it:

    cleartext_disclosure = client.disclosures.decrypt(disclosure)
    cleartext_disclosure.scope                       # => 'output'/'transaction_id'/'account'
    cleartext_disclosure.items[0].asset_id.asset_id  # => "fae9f0af..."

#### Request

    POST /disclosures/decrypt

    {
      encrypted_disclosure: {
        type:         "encdisclosure1",
        sender_key:   <string:hex>,
        selector:     <string:hex>,
        ciphertext:   <string:hex>
      }
    }

#### Response

    {
        type: "disclosure1",   # version of the cleartext disclosure
        items: [
            {
                scope: "output"/"retirement"/"issuance",
                ...
            },
        ]
    }

See [Cleartext Disclosure](#cleartext-disclosure) for details.

#### Ruby

    cleartext_disclosure = client.disclosures.decrypt(
      encrypted_disclosure: {...}
    )

#### JS

    client.disclosures.import({
       encrypted_disclosure: {...}
    })
    .then(cleartextdisclosure => ...)

#### Java

    Disclosure disclosure = new Disclosure.Decrypt(client, encdisclosure)











## Data structures

### Transaction Template

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
                value_range_proof: "9df90af8a0c...",
                program: {
                    vm_version: 1, 
                    bytecode: "..."
                },
                data:               "da1a00fa9e628...", # raw data
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
                            signing_instructions: {
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



### Signing instructions

Signing instructions is a structured representation of data necessary for creation of valid _arguments_ to be used in blockchain data structures such as block headers, transaction inputs, issuance entries and issuance choices.

Signing instructions form a template for a _list of program arguments_, with minimal instructions necessary to compute the arguments.
Additional _contexts_ (per-program `program_context` and per-signature `signature_context`) are provided to allow HSM to verify the instructions and perform additional validation logic.

New versions of the signing instructions can expand the definition of _context_ to enable additional rules governing the signing process.

_Completed_ instructions have all their _arguments_ of type `data`, meaning, all signatures are computed and nothing is left for signing.
Instructions may be partially completed: either some arguments are `type:data` items, or some of `type:multisig` items contain some keys precomputed.

    {
        arguments: [ // stack of arguments for the program
            {
                type: "data",         // indicates that a raw piece of data is already computed
                data: <string:hex>,
            },
            {
                type:     "sig",
                hash:     <string:hex>, // raw hash of the message to be signed
                pubkey:   <string:hex>, // EdDSA pubkey

                // `xpub` and `path` are optional, in case the EdDSA pubkey is derived from the ChainKD xpub:
                xpub:     <string:hex>, // ChainKD extended pubkey
                path:     [<string:hex>], // sequence of selectors for non-hardened derivation

                signature_context: {
                  hash_type: ("txsighash" | "msghash"),

                  // if hash_type=msghash:
                  hash_function: ("sha3" | "sha2"),
                  message: <string:hex>,  // raw message being signed
                }
            },
            {
                type:      "multisig",
                hash:      <string:hex>

                quorum: 1,
                keys: [
                    {
                        pubkey:   <string:hex>, // EdDSA pubkey
                        xpub:     <string:hex>, // ChainKD xpub
                        path:     [<string:hex>], // sequence of selectors for non-hardened derivation
                    },
                    {
                        pubkey:   <string:hex>, // EdDSA pubkey
                        sig:      <string:hex>, // already computed signature for a given pubkey
                    }
                ],
                signature_context: {
                  hash_type: ("txsighash" | "msghash"),

                  // if hash_type=msghash:
                  hash_function: ("sha3" | "sha2"),
                  message: <string:hex>,  // raw message being signed
                }
            },
        ],
        program_context: {
            type:            ("tx"|"block"|"issuancechoice"),
            entry_type:      "input",            // type of entry in the "tx" context
            vm_version:       2,                 // version of the VM (affects allowed types of witness components)
            program:          <string:hex>,      // program bytecode
            position:         0,                 // position of the destination of the current entry (0 for all inputs)
            asset_id:         <string:hex>,      // asset id of the current entry (a)
            asset_commitment: <string:hex>,      // asset commitment of the current entry (b) 
            amount:           <integer>,         // number of asset units in the current entry
            value_commitment: <string:hex>       // value commitment of the current entry (b)
        },
    }
 

### Entry data

The raw data string associated with an output/retirement/issuance entry is composed of the following encrypted strings:

    <encrypted-asset-id><encrypted-amount><encrypted-refdata>

If the asset ID is not confidential, its ciphertext is omitted from the data attachment.
Likewise, if the amount is not confidential, its ciphertext is omitted from the data attachment.

#### Encode data

1. If the asset ID is confidential, [encrypt it](ca.md#encrypt-asset-id) and set `ea` to a 64-byte ciphertext. Otherwise, set `ea` to an empty string.
2. If the amount is confidential, [encrypt it](ca.md#encrypt-value) and set `ev` to a 40-byte ciphertext. Otherwise, set `ea` to an empty string.
3. If the data is confidential, [encrypt it](#reference-data-encryption) and set `ed` to the ciphertext. Otherwise, set `ed` to the cleartext data.
4. Concatenate `d = ea || ev || ed`.
5. Attach `d` to the transaction entry (output, retirement or issuance).

#### Decode data

1. If the asset ID is confidential, extract first 64 bytes of the data attachment and [decrypt them](ca.md#decrypt-asset-id).
2. If the amount is confidential, extract next 40 bytes of the data attachment and [decrypt them](ca.md#decrypt-value).
3. If the data is confidential, [decrypt](#reference-data-encryption) the remaining bytes of the data. Otherwise, set the cleartext to the remaining bytes without modification.


### Disclosure

#### Encrypted Disclosure

    {
        type: "encdisclosure1",              # version of the encrypted disclosure
        sender_key: "fe9af9bc3923...",       # ephemeral sender pubkey
        selector:   "589af9b1a730...",       # blinding selector
        ciphertext: "cc9e012f7a8f99ea9b...", # encrypted hex of the Cleartext Disclosure object
    }

#### Cleartext Disclosure

    {
        type: "disclosure1",                     # version of the cleartext disclosure
        items: [
            {
                scope: "output"/"retirement"/"issuance",
                ...
            },
            {
                scope: "output",                 # disclosure for a single output/retirement/issuance
                entry_id: "...",                 # ID of the output (hex-encoded)
                transaction_id: "...",           # ID of the transaction (hex-encoded)
                reference_data: {
                    cleartext: "...",            # Hex-encoded decrypted reference data
                    dek: "...",                  # Data Encryption Key for this entry
                },
                asset_id: {
                    asset_id: "...",             # Hex-encoded cleartext asset ID
                    aek: "...",                  # Asset Encryption Key for this entry
                },
                amount: {
                    amount: ...,                 # Hex-encoded cleartext asset ID
                    vek: "...",                  # Value Encryption Key for this entry
                }
            },
            {
                scope: "account",                # disclosure for an account
                account_id: "...",               # ID of the output (hex-encoded)
                account_xpubs: [...],            # List of xpubs forming an account
                account_quorum: 1,               # Number of keys required for signing in the account
                reference_data: {
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



## Annotations


Outputs, inputs, retirements:

Field                         | Description
------------------------------|---------------------------
`asset_id_commitment`         | Always present
`value_commitment`            | Always present
`asset_id`                    | Present when stored in plaintext or decrypted
`amount`                      | Present when stored in plaintext or decrypted
`asset_id_confidential`       | `true` if asset ID is confidential (encrypted)
`amount_confidential`         | `true` if amount is confidential (encrypted) on-chain
`data`                        | Hex-encoded raw data (encrypted or cleartext) on-chain
`reference_data`              | Present when stored in plaintext or decrypted
`reference_data_confidential` | `true` if reference data is encrypted on-chain

Issuance:

Field                         | Description
------------------------------|---------------------------
`asset_id_commitment`         | Always present
`value_commitment`            | Always present
`asset_id`                    | Present when stored in plaintext or decrypted
`amount`                      | Present when stored in plaintext or decrypted
`asset_id_confidential`       | `true` if asset ID is confidential (encrypted)
`amount_confidential`         | `true` if amount is confidential (encrypted) on-chain
`asset_id_candidates`         | Array of candidate asset IDs. For non-confidential issuance contains issued `asset_id`.
`data`                        | Hex-encoded raw data (encrypted or cleartext) on-chain
`reference_data`              | Present when stored in plaintext or decrypted
`reference_data_confidential` | `true` if reference data is encrypted on-chain

TBD: add annotated issuances/inputs/outputs/retirements to the Swagger spec.




## Encryption

### Key derivation

Core manages two hierarchies of keys for encrypting transaction templates and asset amounts.
The first hierarchy is used to protect transaction building process and is not tied to any
account. The second hierarchy is deliberately tied to accounts and control programs in order
to simplify indexing and tracking of the payments.

Core is initialized with a **root secret key** `RK` which is stored in the Core's DB.

    RK = random

From that key, Core creates a **root import key** `RIK`, an xprv/xpub pair using RK as a seed:

    RIK = ChainKD(seed: "Root Import Key" || RK)

For accounts and outputs Core creates **root confidentiality key** `RCK`:

    RCK = TupleHash128({RK}, S="Root Confidentiality Key", 32 bytes)

To encrypt fields, `RCK` is expanded to a vector key containing 3 32-byte keys:

    RDEK, RAEK, RVEK = TupleHash128({RCK}, S="Expanded Root Key", 3·32 bytes)

    RDEK — Root Data Encryption Key
    RAEK — Root Asset ID Encryption Key
    RVEK — Root Value Encryption Key

For each account a deterministic account selector is made that's used to generate per-account keys:

    (ADEK|AAEK|AVEK) = TupleHash128({(RDEK|RAEK|RVEK),m,n,xpub1,xpub2,xpub3}, S="Account Key", 32 bytes)
                       (m,n encoded as little-endian 64-bit unsigned integers)

For each account output ("internal output"), a key is derived using control program and a per-account key:

    (ODEK|OAEK|OVEK) = TupleHash128({(ADEK|AAEK|AVEK),control_program}, S="Internal Output Key", 32 bytes)

For each output for an arbitrary control program, a key is derived using control program and a corresponding root key:

    (ODEK|OAEK|OVEK) = TupleHash128({(RDEK|RAEK|RVEK),control_program}, S="External Output Key", 32 bytes)

For the retirement entry, a key is derived using the serialized `value_source` as selector and root keys:

    (TDEK|TAEK|TVEK) = TupleHash128({(RDEK|RAEK|RVEK),value_source}, S="Retirement Key", 32 bytes)

Asset ID-specific key:

    (SDEK|SAEK|SVEK) = TupleHash128({(RDEK|RAEK|RVEK),assetid}, S="Asset ID Key", 32 bytes)

For each issuance, a key is derived using the anchor ID as a selector and asset ID-specific keys:

    (YDEK|YAEK|YVEK) = TupleHash128({(SDEK|SAEK|SVEK),anchorID}, S="Issuance Key", 32 bytes)

For each access token there is a separate _access key_ `AK`:

    AK = TupleHash128({RK, access_token}, S="AK", 32 bytes)

For encryption of a transaction template payload, a unique key is derived from the _access key_ and _payload ID_:

    PK = TupleHash128({AK, payloadID}, S="PK", 32 bytes)


### Entry Shuffling

To shuffle transaction entries before finalization:

1. Compute transaction ID `tempTxID` using the current order of entries.
2. For each entry, compute the [entry ID](blockchain.md#entry-id) `tempEntryID` (note: output entries will have their entry ID changed after shuffling).
3. Compute the per-tx key using Core's root key `RK`, tx id and payloadID:
    
        k = TupleHash128({RK, tempTxID, payloadID}, 32)

4. Compute per-entry sort descriptor:

        d = TupleHash128({k, tempEntryID}, 32)

5. Sort all entries using `d` interpreted as big-endian integer, lower values first.


### Disclosure Import key

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
            type: "disclosureimportkey1",
            key: IKpub,
            selector: b
        }

#### Encrypt Disclosure

1. Verify that import key’s `type` equals `importkey1`.
2. Serialize cleartext disclosure as `data`.
3. Generate a sender private key using Core's root key `RK` as a seed:

        r = ScalarHash("DH", RK, IKpub, b, data)

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

##### Decrypt Disclosure

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


### Payload encryption

Payload is encrypted/decrypted using [Packet Encryption](#packet-encryption) algorithm with the _payload encryption key_ `PK` derived from payload ID and _access key_ `AK`:

    PK = TupleHash128({AK, payloadID}, S="PK", 32 bytes)


### Packet Encryption

#### Encrypt Packet

1. Compute keystream of the same length as cleartext: `keystream = SHAKE128(EK, len(payload))`
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











## Examples

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
      b.control_with_receiver receiver: bob_rcvr,     asset_id: 'USD',  amount: 140, reference_data: "Hello, world!"
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


### Confidential issuance

Confidential issuance works by hiding the issued asset ID among a specified set of asset IDs.
For instance, `AliceIOU` can be issued among `{AliceIOU,BobIOU,CarolIOU}` so that the fact that
Alice is issuing more debt can be kept private from the market and only disclose it to regulators
and concerned counter-parties.

There are three ways to issue assets confidentially:

1. You use only _your_ asset IDs (which you can issue).
2. You use other issuers’ asset IDs that support a specific _issuance key_ in their issuance programs. 
   This is an option for newer asset IDs created with a built-in issuance key support.
3. You use other issuers’ asset IDs that have publicly available signed signature programs with _issuance key_ support.
   This is an option for legacy asset IDs created without built-in issuance key support.

To support second and third option, user must import other issuers’ assets with necessary issuance public keys and witness data:

    chain.assets.import_issuance_key(
      alias: 'BobIOU',
      issuance_key_spec: {
        version:          1,
        asset_id:         'fa0bd0ad1241...',
        initial_block_id: 'a9f03712fad1...',     # the initial_block_id, issuance_program, reference_data define asset ID
        issuance_program: '5604afe9baf0...',
        reference_data:   '9af9f9839102...',
        arguments: ['ad3703...', 'fe8a7210...'], # VM arguments to satisfy issuance program
        issuance_key:     '048af9bd9e01...',     # issuance public key
      }
    )

Structure `issuance_key_spec` is published by the issuer so that other issuers could use it.

To create a set of issuance candidates, Alice uses optional `issuance_choices` field.
She can refer to imported or her own assets by `asset_alias` or `asset_id`.

    tx = client.transactions.build do |b|
      b.issue asset_alias: 'AliceIOU', amount: 10, issuance_choices: [ # optional override to Core's default behaviour to e.g. include all imported and local asset types
        {asset_alias: 'AliceIOU2'},
        {asset_alias: 'BobIOU'},
        {asset_id:    'CarlIOU'}
      ]
      ...
    end

TBD: to create `issuance_key_spec` we need to generate an issuance key, and to support multisig we need to set up a threshold key (ChainTS).
For now we'll only support either non-confidential issuance, or same-issuer assets with transient issuance keys (generated randomly per-issuance).


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



## Discussion

### Compatibility

Current SDK uses `signer.sign()` method to sign a partial transaction. We can keep this behavior and introduce an additional API:

    HSMSigner.sign() - existing behavior as is
    Client.transactions.sign()    - verifies payload and signs TXSIGHASH-based predicate via Client.signer.sign(txhash instead of checkpredicate)

When users upgrade to a new Chain Core, the tx template is changed, but the behavior of the application remains the same. Then, they can smoothly transition to a new API usage:

1. Configure Client.signer instead of a standalone HSMSigner instance.
2. Introduce a second round of tx template exchange to do `Client.sign` after other parties have participated.
3. If using confidential amounts, new signing mechanism is required.


### Encryption of the exported disclosure

Possible options:

1. Encrypting to a Core-generated "Import Key" (as in this spec).
2. Not encrypting and relying on security of the transmission channels.

Arguments for encryption:

1. Encrypting disclosure ensures that data cannot be intercepted between two Cores without assuming that Cores connect directly to each other.
2. Encryption allows concrete auditability of data access and policies: import keys can be signed by long-term keys to provide a delegation chain. This is more important for manual handling of disclosures where a user of one Core exports and transmits it to other people who import it to the target Core.

Arguments against encryption:

1. Recipient must generate a receiving key first, before the exporter can create a disclosure.
2. We may figure a more generalized way for encryption of arbitrary data between Cores, and that would be a custom use of it.

### Inline data VS out of band data

Possible options:

1. Automatically split data for inline (INL) and out of band (OOB) pieces.
2. Keep these explicitly separate.

We choose to keep these fields separate since the inline data has special features and considerations:

* Inline data does not require out-of-band transmission. Therefore, nodes can receive the data from raw blockchain data w/o setting up additional channels.
* Inline data has limited size: around 3-4 Kb.
* Inline data requires reveal of the numeric amount, so these two fields cannot be indepdently disclosed.

In the present specification we omit support for inline data entirely for simplicity of the interface and
intend to introduce it as an additional feature that allows applications to optimize bandwidth usage.


### FAQ


Jeff asks on May 9, 2017:

> - TX template v2:
>     - `signing_instructions` seems to be a misnomer...is `witness_data` a better/more generic term?

Maybe `program_arguments_instructions`? Using `witness_something` covers range proofs, and we win nothing by grouping them together - different parts of the witness structures are filled in at different points in time and may get moved around in the template (e.g. ARP instructions can get consumed by another party and moved to their encrypted payload).

>     - How are payloads encrypted? How do we configure this encryption at the Chain Core level?

Oops, added [Payload Encryption](#payload-encryption) spec. Chain Core will derive necessary keys and encrypt/decrypt these transparently to the SDK user.

> - SDK
>     - Is it `client.sign` or `client.transactions.sign`?

I'd keep it `client.transactions.sign` to be more specific about what is being signed. In case we introduce some other signing of different things later.

>     - For a transitional period when both types of templates must be supported (Chain Core 1.3), how do we instruct `build` to produce a v2 template instead of a v1 template? Do we solve this with flags, or with a different namespace for methods?

...

>     - Is there a syntax for specifying non-encrypted outputs, similar to the `confidential` flag in issuance? Is there even a use case for such a thing?

Yes.

> - Confidential issuance
>     - During confidential issuance, can we make `issuance_choices` optional, and let Chain Core provide sane defaults?

Made optional. Good call.

> - Trackable addresses
>     - How will indexing work in practice? Do we need to test every known account xpub against the selector present in an incoming output entry?

Trackable keys must be publicly visible in the UTXO and needed for Account Disclosures.

Account Disclosures must be designed with Ivy-oriented account definitions in mind. So we can postpone those until Ivy is done.

> - Transaction data structure
>     - What does the entry-based tx data structure look like, from the perspective of an SDK?

Should be an opaque type to hide versioned content inside for future-proofing.





## Swagger definitions

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
          version:
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


      TxHeaderTemplate:
        type: object
        required:
          - type
          - version
          - mintime
          - maxtime
        properties:
          type:
            type: string
            description: Type of the entry (required to be `txheader`).
            enum:
              - txheader
          version:
            type: integer
            description: Blockchain version of the transaction.
          mintime:
            type: integer
            description: Minimum allowed timestamp of a block including the transaction.
          maxtime:
            type: integer
            description: Zero or a maximum allowed timestamp of a block including the transaction.
          txid:
            type: string
            description: Optional transaction ID in hex which is available if the template is finalized.

      Mux1Template:
        type: object
        required:
          - type
        properties:
          type:
            type: string
            description: Type of the entry (required to be `mux1`).
            enum:
              - mux1
          program:
            type: string
            description: The raw hex of the control program.

      Mux2Template:
        type: object
        required:
          - type
        properties:
          type:
            type: string
            description: Type of the entry (required to be `mux2`).
            enum:
              - mux2
          program:
            type: string
            description: The raw hex of the control program.
          excess_factor:
            type: string
            description: The raw excess scalar to be turned into excess commitment.
      
      Issuance1Template:
        type: object
        required:
          - type
          - asset_id
          - amount
        properties:
          type:
            type: string
            description: Type of the entry (required to be `issuance1`).
            enum:
              - issuance1
          asset_id:
            type: string
            description: Hex-encoded asset ID.
          amount:
            type: integer
            description: Amount of units of a specified asset ID.
          data:
            $ref: '#/definitions/DataTemplate'
          signing_instructions:
            type: object
            description: An opaque object describing signing instructions for issuing.

      Issuance2Template:
        type: object
        required:
          - type
          - asset_commitment
          - value_commitment
        properties:
          type:
            type: string
            description: Type of the entry (required to be `issuance2`).
            enum:
              - issuance2
          asset_commitment:
            type: string
            description: Hex-encoded asset commitment (64 bytes).
          value_commitment:
            type: string
            description: Hex-encoded value commitment (64 bytes).
          asset_range_proof:
            type: string
            description: Raw hex-encoded rangeproof for the asset_commitment.
          value_range_proof:
            type: string
            description: Raw hex-encoded rangeproof for the value_commitment.
          asset_range_proof_instructions:
            type: object
            description: Optional instructions for generating an asset range proof.
            required:
              - asset_id
              - blinding_factor
              - issuance_key
              - choices
            properties:
              asset_id:
                type: string
                description: Hex-encoded asset ID.
              blinding_factor:
                type: string
                description: Hex-encoded blinding scalar.
              choices:
                type: array
                description: Issuance choices and corresponding witnesses.
                items:
                  type: object
                  required:
                    - asset_id
                    - issuance_key
                    - signing_instructions
                  properties:
                    asset_id:
                      type: string
                      description: Hex-encoded asset ID in this choice.
                    issuance_key:
                      type: string
                      description: Issuance public key, hex-encoded.
                    issuance_privkey:
                      type: string
                      description: Issuance private key, hex-encoded. Only present when a transient key is used (generated by Core).
                    signing_instructions:
                      type: object
                      description: Opaque signing instructions for each issuance program.
          delegate_program:
            $ref: '#/definitions/Program'
          data:
            $ref: '#/definitions/DataTemplate'
          signing_instructions:
            type: object
            description: An opaque object describing signing instructions for the `delegate_program`.


      Input1Template:
        type: object
        required:
          - type
          - spent_output_id
          - spent_output
        properties:
          type:
            type: string
            description: Type of the entry (required to be `input1`).
            enum:
              - input1
          spent_output_id:
            type: string
            description: Hex ID of the spent output.
          data:
            $ref: '#/definitions/DataTemplate'
          spent_output:
            type: object
            description: Minimal information about the contents of the output necessary for constructing a transaction witness.
            required:
              - asset_id
              - amount
              - program
              - data
              - exthash
            properties:
              asset_id:
                type: string
                description: Hex-encoded asset ID.
              amount:
                type: integer
                description: Amount of units of a specified asset ID.
              program:
                $ref: '#/definitions/Program'
              data:
                type: string
                description: Raw hex-encoded 32-byte data.
              exthash:
                type: string
                description: Extension hash
          signing_instructions:
            type: object
            description: An opaque object describing signing instructions for the input.

      Input2Template:
        type: object
        required:
          - type
          - spent_output_id
          - spent_output
        properties:
          type:
            type: string
            description: Type of the entry (required to be `input1`).
            enum:
              - input2
          spent_output_id:
            type: string
            description: Hex ID of the spent output.
          data:
            $ref: '#/definitions/DataTemplate'
          spent_output:
            type: object
            description: Minimal information about the contents of the output necessary for constructing a transaction witness.
            required:
              - asset_commitment
              - value_commitment
              - program
              - data
              - exthash
            properties:
              asset_commitment:
                type: string
                description: Hex-encoded asset commitment (64 bytes).
              value_commitment:
                type: string
                description: Hex-encoded value commitment (64 bytes).
              program:
                $ref: '#/definitions/Program'
              data:
                type: string
                description: Raw hex-encoded 32-byte data.
              exthash:
                type: string
                description: Extension hash
          signing_instructions:
            type: object
            description: An opaque object describing signing instructions for the input.

      Output1Template:
        type: object
        required:
          - type
          - asset_id
          - amount
          - program
        properties:
          type:
            type: string
            description: Type of the entry (required to be `output1`).
            enum:
              - output1
          asset_id:
            type: string
            description: Hex-encoded asset ID.
          amount:
            type: integer
            description: Amount of units of a specified asset ID.
          program:
            $ref: '#/definitions/Program'
          data:
            $ref: '#/definitions/DataTemplate'

      Output2Template:
        type: object
        required:
          - type
          - asset_commitment
          - value_commitment
          - program
        properties:
          type:
            type: string
            description: Type of the entry (required to be `output2`).
            enum:
              - output2
          asset_commitment:
            type: string
            description: Hex-encoded asset commitment (64 bytes).
          value_commitment:
            type: string
            description: Hex-encoded value commitment (64 bytes).
          asset_range_proof:
            type: string
            description: Raw hex-encoded rangeproof for the asset_commitment.
          value_range_proof:
            type: string
            description: Raw hex-encoded rangeproof for the value_commitment.
          asset_range_proof_instructions:
            type: object
            description: Optional instructions for generating an asset range proof.
            required:
              - asset_id
              - blinding_factor
            properties:
              asset_id:
                type: string
                description: Hex-encoded asset ID.
              blinding_factor:
                type: string
                description: Hex-encoded blinding scalar.
          program:
            $ref: '#/definitions/Program'
          data:
            $ref: '#/definitions/DataTemplate'


      Retirement1Template:
        type: object
        required:
          - type
          - asset_id
          - amount
        properties:
          type:
            type: string
            description: Type of the entry (required to be `retirement1`).
            enum:
              - retirement1
          asset_id:
            type: string
            description: Hex-encoded asset ID.
          amount:
            type: integer
            description: Amount of units of a specified asset ID.
          upgrade_program:
            $ref: '#/definitions/Program'

      Retirement2Template:
        type: object
        required:
          - type
          - asset_commitment
          - value_commitment
        properties:
          type:
            type: string
            description: Type of the entry (required to be `retirement2`).
            enum:
              - retirement2
          asset_commitment:
            type: string
            description: Hex-encoded asset commitment (64 bytes).
          value_commitment:
            type: string
            description: Hex-encoded value commitment (64 bytes).
          asset_range_proof:
            type: string
            description: Raw hex-encoded rangeproof for the asset_commitment.
          value_range_proof:
            type: string
            description: Raw hex-encoded rangeproof for the value_commitment.
          asset_range_proof_instructions:
            type: object
            description: Optional instructions for generating an asset range proof.
            required:
              - asset_id
              - blinding_factor
            properties:
              asset_id:
                type: string
                description: Hex-encoded asset ID.
              blinding_factor:
                type: string
                description: Hex-encoded blinding scalar.

      Upgrade1Template:
        type: object
        required:
          - type
          - upgrade_program
          - asset_id
          - amount
        properties:
          type:
            type: string
            description: Type of the entry (required to be `upgrade1`).
            enum:
              - upgrade1
          upgrade_program:
            type: string
            description: Program that identifies a Retirement1 entry in this transaction.
          asset_id:
            type: string
            description: Hex-encoded asset ID.
          amount:
            type: integer
            description: Amount of units of a specified asset ID.
          exthash:
            type: string
            description: Extension hash
          signing_instructions:
            type: object
            description: An opaque object describing signing instructions for the upgrade program.


      InputPlaceholder:
        description: Specifies asset amount necessary on the left side of the transaction
          (which must by provided via an issuance or an input entry).
        type: object
        required:
          - type
          - asset_id
          - amount
        properties:
          type:
            type: string
            description: Type of the entry (required to be `input-placeholder`).
            enum:
              - input-placeholder
          asset_id:
            type: string
            description: Hex-encoded asset ID.
          amount:
            type: integer
            description: Amount of units of a specified asset ID.
        
      OutputPlaceholder:
        description: Specifies asset amount necessary on the right side of the transaction
          (which must by consumed via an output or a retirement entry).
        type: object
        required:
          - type
          - asset_id
          - amount
        properties:
          type:
            type: string
            description: Type of the entry (required to be `output-placeholder`).
            enum:
              - output-placeholder
          asset_id:
            type: string
            description: Hex-encoded asset ID.
          amount:
            type: integer
            description: Amount of units of a specified asset ID.

      Program:
        type: object
        required:
          - vm_version
          - bytecode
        properties:
          vm_version:
            type: integer
            description: VM version.
          bytecode:
            type: string
            description: Hex-encoded raw bytecode of the program.

      DataTemplate:
        type: object
        required:
          - hash
        properties:
          hash:
            type: string
            description: Raw hex-encoded 32-byte data.
          raw_contents:
            type: string
            description: Raw hex-encoded data string, preimage of the `hash` property.
          reference_data:
            type: string
            description: Hex-encoded 32-byte cleartext reference data which
              is equal, or a portion of `raw_contents`.


      DisclosureImportKey:
        type: object
        required:
          - type
          - key
          - selector
        properties:
          type:
            type: string
            description: Versioned type of the object. Current type is "disclosureimportkey1".
          key:
            type: string
            description: EdDSA public key in hex.
          selector:
            type: string
            description: Pseudo-random selector in hex used to derive a corresponding private key.

      EncryptedDisclosure:
        type: object
        required:
          - type
          - sender_key
          - selector
          - ciphertext
        properties:
          type:
            type: string
            description: Versioned type of the object. Current type is "encdisclosure1".
          sender_key:
            type: string
            description: EdDSA public key in hex used to reconstruct an encryption key.
          selector:
            type: string
            description: Pseudo-random selector in hex as specified in DisclosureImportKey.
          ciphertext:
            type: string
            description: Hex-encoded ciphertext of the CleartextDisclosure.

      CleartextDisclosure:
        type: object
        required:
          - type
          - items
        properties:
          type:
            type: string
            description: Versioned type of the object. Current type is "disclosure1".
          items:
            type: array
            items:
              $ref: '#/definitions/DisclosureItem'
          alias:
            type: string
            description: Optional alias for the disclosure. Specified during import by the receiving party.

      DisclosureItem:
        type: object
        description: There are several types of items in disclosure. Since Swagger 2.0
          does not allow for polymorphic types, the individual properties are not listed here.
          Please refer to the definitions of:
          DisclosureItemEntry,
          DisclosureItemAccount.

      DisclosureItemEntry:
        type: object
        required:
          - scope
          - entry_id
          - transaction_id
        properties:
          scope:
            type: string
            description: Type of entry being disclosed.
            enum:
              - output
              - retirement
              - issuance
          entry_id:
            type: string
            description: Hash of the entry in hex.
          transaction_id:
            type: string
            description: Hash of the transaction including this entry in hex.
          reference_data:
            type: object
            required:
              - cleartext
              - dek
            properties:
              cleartext:
                type: string
                description: Cleartext contents of the data.
              dek:
                type: string
                description: Data encryption key in hex. An empty string if data is not encrypted.
          asset_id:
            type: object
            required:
              - asset_id
              - aek
            properties:
              asset_id:
                type: string
                description: Cleartext asset ID in hex.
              aek:
                type: string
                description: Asset ID encryption key in hex.
          amount:
            type: object
            required:
              - amount
              - vek
            properties:
              amount:
                type: integer
                description: Cleartext amount.
              vek:
                type: string
                description: Value encryption key in hex.

      DisclosureItemAccount:
        type: object
        required:
          - scope
          - account_id
          - account_xpubs
          - account_quorum
        properties:
          scope:
            type: string
            description: Contains "account" string to identify the account-scoped disclosure item.
            enum:
              - account
          account_id:
            type: string
            description: Core's internal account identifier.
          account_xpubs:
            type: array
            items:
              type: string
            description: A list of xpubs from which the account's control
              program pubkeys will be derived.
          account_quorum:
            type: integer
            description: The number of signatures required for spending
              funds controlled by the account's control programs.
          reference_data:
            type: object
            required:
              - dek
            properties:
              dek:
                type: string
                description: Data encryption key in hex. An empty string if data is not encrypted.
          asset_id:
            type: object
            required:
              - aek
            properties:
              aek:
                type: string
                description: Asset ID encryption key in hex.
          amount:
            type: object
            required:
              - vek
            properties:
              vek:
                type: string
                description: Value encryption key in hex.


