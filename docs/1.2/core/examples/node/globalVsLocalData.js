const chain = require('chain-sdk')

const client = new chain.Client()
const signer = new chain.HsmSigner()

let assetKey, aliceKey, bobKey

Promise.all([
  client.mockHsm.keys.create(),
  client.mockHsm.keys.create(),
  client.mockHsm.keys.create(),
]).then(keys => {
  assetKey = keys[0]
  aliceKey = keys[1]
  bobKey   = keys[2]

  signer.addKey(assetKey, client.mockHsm.signerConnection)
  signer.addKey(aliceKey, client.mockHsm.signerConnection)
  signer.addKey(bobKey, client.mockHsm.signerConnection)
}).then(() => {
  return (
    // snippet create-accounts-with-tags
    client.accounts.create({
      alias: 'alice',
      rootXpubs: [aliceKey.xpub],
      quorum: 1,
      tags: {
        type: 'checking',
        first_name: 'Alice',
        last_name: 'Jones',
        user_id: '12345',
        status: 'enabled'
      }
    }).then(() =>
      client.accounts.create({
        alias: 'bob',
        rootXpubs: [bobKey.xpub],
        quorum: 1,
        tags: {
          type: 'checking',
          first_name: 'Bob',
          last_name: 'Smith',
          user_id: '67890',
          status: 'enabled'
        }
      })
    )
    // endsnippet
  )
}).then(() =>
  // snippet create-asset-with-tags-and-definition
  client.assets.create({
    alias: 'acme_bond',
    rootXpubs: [assetKey.xpub],
    quorum: 1,
    tag: {
      internal_rating: 'B',
    },
    definition: {
      type: 'security',
      sub_type: 'corporate-bond',
      entity: 'Acme Inc.',
      maturity: '2016-09-01T18:24:47+00:00'
    }
  })
  // endsnippet
).then(() => {
  // snippet build-tx-with-tx-ref-data
  const buildPromise = client.transactions.build(builder => {
    builder.issue({assetAlias: 'acme_bond', amount: 100})
    builder.controlWithAccount({accountAlias: 'alice', assetAlias: 'acme_bond', amount: 100})
    builder.transactionReferenceData({externalReference: '12345'})
  })
  // endsnippet

  return buildPromise
}).then(
  txTemplate => signer.sign(txTemplate)
).then(
  txTemplate => client.transactions.submit(txTemplate)
).then(() => {
  // snippet build-tx-with-action-ref-data
  const buildPromise = client.transactions.build(builder => {
    builder.issue({assetAlias: 'acme_bond', amount: 100})
    builder.retire({
      assetAlias: 'acme_bond',
      amount: 100,
      referenceData: {external_reference: '12345'}
    })
  })
  // endsnippet

  return buildPromise
}).then(
  txTemplate => signer.sign(txTemplate)
).then(
  txTemplate => client.transactions.submit(txTemplate)
).catch(err =>
  process.nextTick(() => { throw err })
)
