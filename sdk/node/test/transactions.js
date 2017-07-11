
/* eslint-env mocha */

const assert = require('assert')
const chai = require('chai')
const chaiAsPromised = require('chai-as-promised')

chai.use(chaiAsPromised)
const expect = chai.expect

const { balanceByAssetAlias, client, createAccount, createAsset } = require('./testHelpers')

describe('Transaction', () => {

  describe('Issuance', () => {
    let goldAlias, silverAlias, aliceAlias, bobAlias

    before(() => {
      return Promise.all([
        createAsset('gold'),
        createAsset('silver'),
        createAccount('alice'),
        createAccount('bob')
      ])
      .then((objects) => {
        goldAlias = objects[0].alias
        silverAlias = objects[1].alias
        aliceAlias = objects[2].alias
        bobAlias = objects[3].alias
      })
      .then(() => client.transactions.build(builder => {
        builder.issue({
          assetAlias: goldAlias,
          amount: 100
        })
        builder.issue({
          assetAlias: silverAlias,
          amount: 200
        })
        builder.controlWithAccount({
          accountAlias: aliceAlias,
          assetAlias: goldAlias,
          amount: 100
        })
        builder.controlWithAccount({
          accountAlias: bobAlias,
          assetAlias: silverAlias,
          amount: 200
        })
      }))
      .then((issuance) => client.transactions.sign(issuance))
      .then((signed) => client.transactions.submit(signed))
    })

    it('issues 100 units of gold to alice', () => {
      return balanceByAssetAlias(client.balances.query({filter: `account_alias='${aliceAlias}'`}))
        .then((balance) => assert.deepEqual(balance, {[goldAlias]: 100}))
    })

    it('issues 200 units of silver to bob', () => {
      return balanceByAssetAlias(client.balances.query({filter: `account_alias='${bobAlias}'`}))
        .then((balance) => assert.deepEqual(balance, {[silverAlias]: 200}))
    })
  })

  describe('Atomic swap', () => {
    let goldAlias, silverAlias, aliceAlias, bobAlias

    before(() => {
      return Promise.all([
        createAsset('gold'),
        createAsset('silver'),
        createAccount('alice'),
        createAccount('bob')
      ])
      .then((objects) => {
        goldAlias = objects[0].alias
        silverAlias = objects[1].alias
        aliceAlias = objects[2].alias
        bobAlias = objects[3].alias
      })
      .then(() => client.transactions.build(builder => {
        builder.issue({
          assetAlias: goldAlias,
          amount: 100
        })
        builder.issue({
          assetAlias: silverAlias,
          amount: 200
        })
        builder.controlWithAccount({
          accountAlias: aliceAlias,
          assetAlias: goldAlias,
          amount: 100
        })
        builder.controlWithAccount({
          accountAlias: bobAlias,
          assetAlias: silverAlias,
          amount: 200
        })
      }))
      .then((issuance) => client.transactions.sign(issuance))
      .then((signed) => client.transactions.submit(signed))
      .then(() => client.transactions.build(builder => {
        builder.spendFromAccount({
          accountAlias: aliceAlias,
          assetAlias: goldAlias,
          amount: 10
        })
        builder.controlWithAccount({
          accountAlias: aliceAlias,
          assetAlias: silverAlias,
          amount: 20
        })
      }))
      .then((swapProposal) => {
        swapProposal.allowAdditionalActions = true
        return client.transactions.sign(swapProposal)
      })
      .then((swapProposal) =>
        client.transactions.build(builder => {
          builder.baseTransaction = swapProposal.rawTransaction
          builder.spendFromAccount({
            accountAlias: bobAlias,
            assetAlias: silverAlias,
            amount: 20
          })
          builder.controlWithAccount({
            accountAlias: bobAlias,
            assetAlias: goldAlias,
            amount: 10
          })
        }))
        .then((swapTx) => client.transactions.sign(swapTx))
        .then((signed) => client.transactions.submit(signed))
    })

    it('gives alice 20 silver for 10 gold', () => {
      return balanceByAssetAlias(client.balances.query({filter: `account_alias='${aliceAlias}'`}))
        .then((balance) => assert.deepEqual(balance, {[goldAlias]: 90, [silverAlias]: 20}))
    })

    it('gives bob 10 gold for 20 silver', () => {
      return balanceByAssetAlias(client.balances.query({filter: `account_alias='${bobAlias}'`}))
        .then((balance) => assert.deepEqual(balance, {[goldAlias]: 10, [silverAlias]: 180}))
    })

  })

  it('fails to build transactions for non-existent assets', () => {
    return expect(client.transactions.build(builder => {
      builder.issue({
        assetAlias: 'unobtanium',
        amount: 100
      })
    }))
    .to.be.rejectedWith('CH002')
  })

  it('fails to build an unbalanced transaction', () => {
    let testAccountAlias, testAssetAlias

    return Promise.all([
      createAccount('testAccount'),
      createAsset('testAsset')
    ])
    .then((objects) => {
      testAccountAlias = objects[0].alias
      testAssetAlias = objects[1].alias
    })
    .then(() =>
      client.transactions.build(builder => {
        builder.issue({
          assetAlias: testAssetAlias,
          amount: 1
        })
        builder.controlWithAccount({
          accountAlias: testAccountAlias,
          assetAlias: testAssetAlias,
          amount: 100
        })
      })
    )
    .then(issuance => client.transactions.sign(issuance))
    .then(signed => expect(client.transactions.submit(signed)).to.be.rejectedWith('CH735'))
  })

  describe('queryAll', () => {
    it('success example', () => {
      let created
      const queried = []

      return Promise.all([
        createAsset(),
        createAccount()
      ]).then(([asset, account]) =>
        client.transactions.build(builder => {
          builder.issue({assetAlias: asset.alias, amount: 1})
          builder.controlWithAccount({
            accountAlias: account.alias,
            assetAlias: asset.alias,
            amount: 1
          })
        })
      ).then(txtpl =>
        client.transactions.sign(txtpl)
      ).then(signed =>
        client.transactions.submit(signed)
      ).then(tx =>
        created = tx.id
      ).then(() =>
        client.transactions.queryAll({}, (tx, next, done) => {
          queried.push(tx.id)
          next()
        })
      ).then(() => {
        expect(queried).to.include(created)
      })
    })
  })

  describe('Builder function errors', () => {
    it('rejects via promise', () =>
      expect(
        client.transactions.build(() => {
          throw new Error("test error")
        })
      ).to.be.rejectedWith("test error")
    )

    it('rejects batch errors via promise', () =>
      expect(
        client.transactions.buildBatch([
          () => { /* do nothing */ },
          () => { throw new Error("test error") },
          () => { /* do nothing */ },
        ])
      ).to.be.rejectedWith("test error")
    )
  })

  // These just test that the callback is engaged correctly. Behavior is
  // tested in the promises test.
  describe('Callback support', () => {

    it('build', (done) => {
      client.transactions.build(
        () => {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('submit', (done) => {
      client.transactions.submit(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('query', done => {
      client.transactions.query({}, done)
    })

    it('queryAll', done => {
      client.transactions.queryAll(
        {},
        (t, next, queryDone) => queryDone(),
        done
      )
    })

  })
})
