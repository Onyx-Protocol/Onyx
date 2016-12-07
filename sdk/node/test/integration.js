const chain = require('../index.js')
const uuid = require('uuid')
const assert = require('assert')

const balanceByAssetAlias = (balances) => {
  let res = {}
  return Promise.resolve(balances)
  .then((balance) => {
    balance.items.forEach((item) => {
      res[item.sum_by['asset_alias']] = item.amount
    })
    return res
  })
}

describe('Chain SDK integration test', function() {
  it('integration test', function() {
    const client = new chain.Client()
    const signer = new chain.HsmSigner()

    const aliceAlias = `alice-${uuid.v4()}`
    const bobAlias = `bob-${uuid.v4()}`
    const goldAlias = `gold-${uuid.v4()}`
    const silverAlias = `silver-${uuid.v4()}`
    const bronzeAlias = `bronze-${uuid.v4()}`
    const copperAlias = `copper-${uuid.v4()}`

    let aliceKey, bobKey, goldKey, silverKey, otherKey, aliceId

    return Promise.resolve()

    // Access tokens

    // TBD

    // Key creation and signer setup

    .then(() => Promise.all([
      client.mockHsm.keys.create({alias: aliceAlias}),
      client.mockHsm.keys.create({alias: bobAlias}),
      client.mockHsm.keys.create({alias: goldAlias}),
      client.mockHsm.keys.create({alias: silverAlias}),
      client.mockHsm.keys.create({alias: bronzeAlias}),
      client.mockHsm.keys.create({alias: copperAlias}),
      client.mockHsm.keys.create(),
    ])).then(keys => {
      aliceKey = keys[0]
      bobKey = keys[1]
      goldKey = keys[2]
      silverKey = keys[3]
      otherKey = keys[4]

      signer.addKey(aliceKey.xpub, client.mockHsm.signerUrl)
      signer.addKey(bobKey.xpub, client.mockHsm.signerUrl)
      signer.addKey(goldKey.xpub, client.mockHsm.signerUrl)
      signer.addKey(silverKey.xpub, client.mockHsm.signerUrl)
    })

    // Account creation

    .then(() => Promise.all([
      client.accounts.create({alias: aliceAlias, root_xpubs: [aliceKey.xpub], quorum: 1}),
      client.accounts.create({alias: bobAlias, root_xpubs: [bobKey.xpub], quorum: 1})
    ])).then(accounts => {
      aliceId = accounts[0].id
    })

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

    // Asset creation

    .then(() => Promise.all([
      client.assets.create({alias: goldAlias, root_xpubs: [goldKey.xpub], quorum: 1}),
      client.assets.create({alias: silverAlias, root_xpubs: [silverKey.xpub], quorum: 1})
    ]))

    .then(() => client.assets.create({alias: 'unobtanium'}))
    .catch(exception => {
      // Request is missing key fields
      assert.ok(exception instanceof Error)
    })

    // Batch asset creation

    .then(() =>
      client.assets.createBatch([
        {alias: bronzeAlias, root_xpubs: [otherKey.xpub], quorum: 1}, // success
        {alias: 'unobtanium'},
        {alias: copperAlias, root_xpubs: [otherKey.xpub], quorum: 1}, // success
      ])
    ).then(batchResponse => {
      assert.equal(batchResponse.successes.length, 2)
      assert.equal(batchResponse.errors.length, 1)
    })

    // Basic issuance

    .then(() =>
      client.transactions.build( function(builder){
        builder.issue({
          asset_alias: goldAlias,
          amount: 100
        })
        builder.issue({
          asset_alias: silverAlias,
          amount: 200
        })
        builder.controlWithAccount({
          account_alias: aliceAlias,
          asset_alias: goldAlias,
          amount: 100
        })
        builder.controlWithAccount({
          account_alias: bobAlias,
          asset_alias: silverAlias,
          amount: 200
        })
      })
      .then((issuance) => signer.sign(issuance))
      .then((signed) => client.transactions.submit(signed))
    )

    .then(() => Promise.all([
      balanceByAssetAlias(client.balances.query({filter: `account_alias='${aliceAlias}'`})),
      balanceByAssetAlias(client.balances.query({filter: `account_alias='${bobAlias}'`}))
    ]))
    .then(balances => {
      assert.deepEqual(balances[0], {[goldAlias]: 100})
      assert.deepEqual(balances[1], {[silverAlias]: 200})
    })

    // Bad singleton build call

    .then(() => client.transactions.build( function(builder) {
      builder.issue({
        asset_alias: "unobtanium",
        amount: 100
      })
    }))
    .catch(exception => {
      // Non-existent asset
      assert.ok(exception instanceof Error)
    })

    // Bad singleton submit call

    .then(() => client.transactions.build( function(builder) {
      builder.issue({
        asset_alias: goldAlias,
        amount: 1
      })
      builder.controlWithAccount({
        account_alias: aliceAlias,
        asset_alias: goldAlias,
        amount: 100
      })
    }))
    .then((issuance) => signer.sign(issuance))
    .then((signed) => client.transactions.submit(signed))
    .catch(exception => {
      // unbalanced transaction
      assert.ok(exception instanceof Error)
    })

    // Atomic swap

    .then(() => client.transactions.build( function(builder) {
      builder.spendFromAccount({
        account_alias: aliceAlias,
        asset_alias: goldAlias,
        amount: 10
      })
      builder.controlWithAccount({
        account_alias: aliceAlias,
        asset_alias: silverAlias,
        amount: 20
      })
    })
    .then((swapProposal) => {
      swapProposal.allow_additional_actions = true
      return signer.sign(swapProposal)
    })
    .then((swapProposal) =>
      client.transactions.build( function(builder) {
        builder.baseTransaction(swapProposal.raw_transaction)
        builder.spendFromAccount({
          account_alias: bobAlias,
          asset_alias: silverAlias,
          amount: 20
        })
        builder.controlWithAccount({
          account_alias: bobAlias,
          asset_alias: goldAlias,
          amount: 10
        })
      }))
      .then((swapTx) => signer.sign(swapTx))
      .then((signed) => client.transactions.submit(signed))
    )

    .then(() => Promise.all([
      balanceByAssetAlias(client.balances.query({filter: `account_alias='${aliceAlias}'`})),
      balanceByAssetAlias(client.balances.query({filter: `account_alias='${bobAlias}'`}))
    ]))
    .then(balances => {
      assert.deepEqual(balances[0], {[goldAlias]: 90, [silverAlias]: 20})
      assert.deepEqual(balances[1], {[goldAlias]: 10, [silverAlias]: 180})
    })

    // Batch transaction TBD

    // Control program creation

    .then(() => client.accounts.createControlProgram({alias: aliceAlias}))
    .then((cp) => assert(cp.control_program))

    .then(() => client.accounts.createControlProgram({id: aliceId}))
    .then((cp) => assert(cp.control_program))

    .then(() => client.accounts.createControlProgram())
    .catch(exception => {
      // Bad parameters
      assert.ok(exception instanceof Error)
    })

    // Pay to control program

    .then(() => client.accounts.createControlProgram({alias: aliceAlias}))
    .then((cp) => client.transactions.build( function(builder) {
      builder.issue({
        asset_alias: goldAlias,
        amount: 1
      })
      builder.controlWithProgram({
        asset_alias: goldAlias,
        amount: 1,
        control_program: cp.control_program
      })
    }))
    .then((issuance) => signer.sign(issuance))
    .then((signed) => client.transactions.submit(signed))

    // Transaction feeds

    .then(() => Promise.all([
      client.transactionFeeds.create({
        alias: 'issuances',
        filter: "inputs(type='issue')"
      })
      client.transactionFeeds.create({
        alias: 'spends',
        filter: "inputs(type='spend')"
      })
    ]))

  })
})
