const chain = require('../dist/index.js')
const uuid = require('uuid')
const assert = require('assert')
const chai = require("chai")
const chaiAsPromised = require("chai-as-promised")
chai.use(chaiAsPromised)

const expect = chai.expect

// Helper function
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

describe('Promise style', function() {
  it('works', function() {
    const client = new chain.Client()
    const signer = new chain.HsmSigner()

    const aliceAlias = `alice-${uuid.v4()}`
    const bobAlias = `bob-${uuid.v4()}`
    const goldAlias = `gold-${uuid.v4()}`
    const silverAlias = `silver-${uuid.v4()}`
    const bronzeAlias = `bronze-${uuid.v4()}`
    const copperAlias = `copper-${uuid.v4()}`
    const issuancesAlias = `issuances-${uuid.v4()}`
    const spendsAlias = `spends-${uuid.v4()}`
    const tokenId = `token-${uuid.v4()}`

    let aliceKey, bobKey, goldKey, silverKey, otherKey, aliceId

    return Promise.resolve()

    // Access tokens

    .then(() =>
      expect(client.accessTokens.create({
        type: 'client',
        id: tokenId
      })).to.be.fulfilled
    )
    .then(resp => {
      expect(resp.token).to.not.be.empty
    })
    .then(() =>
      expect(client.accessTokens.create({
        type: 'client',
        id: tokenId
      }))
      // Using same ID twice will trigger a duplicate ID error
      .to.be.rejectedWith('CH302')
    )
    .then(() => expect(client.accessTokens.query()).to.be.fulfilled )
    .then(resp => expect(resp.items.map(item => item.id)).to.contain(tokenId))

    .then(() => expect(client.accessTokens.query({type: 'client'})).to.be.fulfilled )
    .then(resp => expect(resp.items.map(item => item.id)).to.contain(tokenId))

    .then(() => expect(client.accessTokens.query({type: 'network'})).to.be.fulfilled )
    .then(resp => expect(resp.items.map(item => item.id)).to.not.contain(tokenId))

    .then(() => expect(client.accessTokens.delete(tokenId)).to.be.fulfilled )
    .then(() => expect(client.accessTokens.query()).to.be.fulfilled )
    .then(resp => expect(resp.items.map(item => item.id)).to.not.contain(tokenId))

    // Key creation and signer setup

    .then(() => expect(Promise.all([
      client.mockHsm.keys.create({alias: aliceAlias}),
      client.mockHsm.keys.create({alias: bobAlias}),
      client.mockHsm.keys.create({alias: goldAlias}),
      client.mockHsm.keys.create({alias: silverAlias}),
      client.mockHsm.keys.create({alias: bronzeAlias}),
      client.mockHsm.keys.create({alias: copperAlias}),
      client.mockHsm.keys.create(),
    ])).to.be.fulfilled)

    .then(keys => {
      aliceKey = keys[0]
      bobKey = keys[1]
      goldKey = keys[2]
      silverKey = keys[3]
      otherKey = keys[6]

      signer.addKey(aliceKey, client.mockHsm.signerConnection)
      signer.addKey(bobKey, client.mockHsm.signerConnection)
      signer.addKey(goldKey, client.mockHsm.signerConnection)
      signer.addKey(silverKey, client.mockHsm.signerConnection)
    })

    // Account creation

    .then(() => expect(Promise.all([
      client.accounts.create({alias: aliceAlias, root_xpubs: [aliceKey.xpub], quorum: 1}),
      client.accounts.create({alias: bobAlias, root_xpubs: [bobKey.xpub], quorum: 1})
    ])).to.be.fulfilled)

    .then(accounts => {
      aliceId = accounts[0].id
    })

    .then(() =>
      expect(client.accounts.create({alias: 'david'}))
      // Request is missing key fields
      .to.be.rejectedWith('CH202')
    )

    // Batch account creation

    .then(() =>
      expect(client.accounts.createBatch([
        {alias: `carol-${uuid.v4()}`, root_xpubs: [otherKey.xpub], quorum: 1}, // success
        {alias: 'david'},
        {alias: `eve-${uuid.v4()}`, root_xpubs: [otherKey.xpub], quorum: 1}, // success
      ])).to.be.fulfilled
    ).then(batchResponse => {
      assert.equal(batchResponse.successes[1], null)
      assert.deepEqual([batchResponse.errors[0], batchResponse.errors[2]], [null, null])
    })

    // Asset creation

    .then(() => expect(Promise.all([
      client.assets.create({alias: goldAlias, root_xpubs: [goldKey.xpub], quorum: 1}),
      client.assets.create({alias: silverAlias, root_xpubs: [silverKey.xpub], quorum: 1})
    ])).to.be.fulfilled)

    .then(() =>
      expect(client.assets.create({alias: 'unobtanium'}))
      // Request is missing key fields
      .to.be.rejectedWith('CH202')
    )

    // Batch asset creation

    .then(() =>
      expect(client.assets.createBatch([
        {alias: bronzeAlias, root_xpubs: [otherKey.xpub], quorum: 1}, // success
        {alias: 'unobtanium'},
        {alias: copperAlias, root_xpubs: [otherKey.xpub], quorum: 1}, // success
      ])).to.be.fulfilled
    ).then(batchResponse => {
      assert.equal(batchResponse.successes[1], null)
      assert.deepEqual([batchResponse.errors[0], batchResponse.errors[2]], [null, null])
    })

    // Basic issuance

    .then(() =>
      expect(client.transactions.build( function(builder){
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
      })).to.be.fulfilled
      .then((issuance) => expect(signer.sign(issuance)).to.be.fulfilled)
      .then((signed) => expect(client.transactions.submit(signed)).to.be.fulfilled)
    )

    .then(() => expect(Promise.all([
      balanceByAssetAlias(client.balances.query({filter: `account_alias='${aliceAlias}'`})),
      balanceByAssetAlias(client.balances.query({filter: `account_alias='${bobAlias}'`}))
    ])).to.be.fulfilled)
    .then(balances => {
      assert.deepEqual(balances[0], {[goldAlias]: 100})
      assert.deepEqual(balances[1], {[silverAlias]: 200})
    })

    // Bad singleton build call

    .then(() =>
      expect(client.transactions.build( function(builder) {
        builder.issue({
          asset_alias: "unobtanium",
          amount: 100
        })
      }))
      // Non-existent asset
      .to.be.rejectedWith('CH002')
    )

    // Bad singleton submit call

    .then(() =>
      expect(client.transactions.build( function(builder) {
        builder.issue({
          asset_alias: goldAlias,
          amount: 1
        })
        builder.controlWithAccount({
          account_alias: aliceAlias,
          asset_alias: goldAlias,
          amount: 100
        })
      })).to.be.fulfilled
      .then(issuance => expect(signer.sign(issuance)).to.be.fulfilled)
      .then(signed =>
        expect(client.transactions.submit(signed))
        // unbalanced transaction
        .to.be.rejectedWith('CH735')
      )
    )

    // Atomic swap

    .then(() =>
      expect(client.transactions.build( function(builder) {
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
      }))
      .to.be.fulfilled
      .then((swapProposal) => {
        swapProposal.allow_additional_actions = true
        return expect(signer.sign(swapProposal)).to.be.fulfilled
      })
      .then((swapProposal) =>
        expect(client.transactions.build( function(builder) {
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
        .to.be.fulfilled)
        .then((swapTx) => expect(signer.sign(swapTx)).to.be.fulfilled)
        .then((signed) => expect(client.transactions.submit(signed)).to.be.fulfilled)
      )

    .then(() => Promise.all([
      balanceByAssetAlias(client.balances.query({filter: `account_alias='${aliceAlias}'`})),
      balanceByAssetAlias(client.balances.query({filter: `account_alias='${bobAlias}'`}))
    ]))
    .then(balances => {
      assert.deepEqual(balances[0], {[goldAlias]: 90, [silverAlias]: 20})
      assert.deepEqual(balances[1], {[goldAlias]: 10, [silverAlias]: 180})
    })

    // Batch transactions

    .then(() => expect(client.transactions.buildBatch([
        // Should succeed
        (builder) => {
          builder.issue({
            asset_alias: goldAlias,
            amount: 100
          })
          builder.controlWithAccount({
            account_alias: aliceAlias,
            asset_alias: goldAlias,
            amount: 100
          })
        },

        // Should fail at the build step
        (builder) => {
          builder.issue({
            asset_alias: 'foobar'
          })
        },

        // Should fail at the submit step
        (builder) => {
          builder.issue({
            asset_alias: goldAlias,
            amount: 50
          })
          builder.controlWithAccount({
            account_alias: aliceAlias,
            asset_alias: goldAlias,
            amount: 100
          })
        },

        // Should succeed
        (builder) => {
          builder.issue({
            asset_alias: silverAlias,
            amount: 50
          })
          builder.controlWithAccount({
            account_alias: bobAlias,
            asset_alias: silverAlias,
            amount: 50
          })
        }])).to.be.fulfilled
    )
    .then(buildBatch => {
      assert.equal(buildBatch.successes[1], null)
      assert.deepEqual([buildBatch.errors[0], buildBatch.errors[2], buildBatch.errors[3]], [null, null, null])
      return expect(signer.signBatch(buildBatch.successes)).to.be.fulfilled
    })
    .then(signedBatch => {
      assert(!signedBatch.successes.includes(null))
      assert.deepEqual([signedBatch.errors[0], signedBatch.errors[1], signedBatch.errors[2]], [null, null, null])
      return expect(client.transactions.submitBatch(signedBatch.successes)).to.be.fulfilled
    })
    .then(submitBatch => {
      assert.equal(submitBatch.successes[1], null)
      assert.deepEqual([submitBatch.errors[0], submitBatch.errors[2]], [null, null])
    })

    // Control program creation

    .then(() =>
      expect(client.accounts.createControlProgram({alias: aliceAlias})).to.be.fulfilled)
    .then((cp) => assert(cp.control_program))

    .then(() =>
      expect(client.accounts.createControlProgram({id: aliceId})).to.be.fulfilled)
    .then((cp) => assert(cp.control_program))

    .then(() =>
      // Empty alias/id
      expect(client.accounts.createControlProgram({}))
      .to.be.rejectedWith('CH003'))

    .then(() =>
      // Non-existent alias
      expect(client.accounts.createControlProgram({alias: "unobtalias"}))
      .to.be.rejectedWith('CH002'))

    // Pay to control program

    .then(() =>
      expect(client.accounts.createControlProgram({alias: aliceAlias})).to.be.fulfilled)
    .then((cp) =>
      expect(client.transactions.build( function(builder) {
        builder.issue({
          asset_alias: goldAlias,
          amount: 1
        })
        builder.controlWithProgram({
          asset_alias: goldAlias,
          amount: 1,
          control_program: cp.control_program
        })
      })).to.be.fulfilled)
    .then((issuance) => expect(signer.sign(issuance)).to.be.fulfilled)
    .then((signed) => expect(client.transactions.submit(signed)).to.be.fulfilled)

    // Transaction feeds

    .then(() => expect(Promise.all([
      client.transactionFeeds.create({
        alias: issuancesAlias,
        filter: "inputs(type='issue')"
      }),
      client.transactionFeeds.create({
        alias: spendsAlias,
        filter: "inputs(type='spend')"
      })
    ])).to.be.fulfilled)
  })
})
