const chain = require('../index.js')
const uuid = require('uuid')
const assert = require('assert')

describe('Chain SDK integration test', function() {
  it('integration test', function() {
    const client = new chain.Client()
    const signer = new chain.HsmSigner()

    const aliceAlias = `alice-${uuid.v4()}`
    const bobAlias = `bob-${uuid.v4()}`
    const goldAlias = `gold-${uuid.v4()}`
    const silverAlias = `silver-${uuid.v4()}`

    let aliceKey, bobKey, goldKey, silverKey, otherKey

    return Promise.resolve()

    // Access tokens

    // TBD

    // Key creation and signer setup

    .then(() => Promise.all([
      client.mockHsm.keys.create({alias: aliceAlias}),
      client.mockHsm.keys.create({alias: bobAlias}),
      client.mockHsm.keys.create({alias: goldAlias}),
      client.mockHsm.keys.create({alias: silverAlias}),
      client.mockHsm.keys.create(),
    ])).then(keys => {
      aliceKey = keys[0]
      bobKey = keys[1]
      goldKey = keys[2]
      silverKey = keys[3]
      otherKey = keys[4]

      signer.addKey(aliceKey, client.mockHsm.signerUrl)
      signer.addKey(bobKey, client.mockHsm.signerUrl)
      signer.addKey(goldKey, client.mockHsm.signerUrl)
      signer.addKey(silverKey, client.mockHsm.signerUrl)
    })

    // Account creation

    .then(() => Promise.all([
      client.accounts.create({alias: aliceAlias, root_xpubs: [aliceKey.xpub], quorum: 1}),
      client.accounts.create({alias: bobAlias, root_xpubs: [bobKey.xpub], quorum: 1})
    ]))

    .then(() => client.accounts.create({alias: 'david'}))
    .catch(exception => {
      // Request is missing key fields
      assert.ok(exception instanceof Error)
    })

    // Batch account creation

    .then(() =>
      client.accounts.createBatch([
        {alias: `carol-${uuid.v4()}`, root_xpubs: [otherKey.xpub], quorum: 1}, // success
        {alias: 'david'},
        {alias: `eve-${uuid.v4()}`, root_xpubs: [otherKey.xpub], quorum: 1}, // success
      ])
    ).then(batchResponse => {
      assert.equal(batchResponse.successes.length, 2)
      assert.equal(batchResponse.errors.length, 1)
    })

  })
})
