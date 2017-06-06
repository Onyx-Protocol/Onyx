
/* eslint-env mocha */

const assert = require('assert')
const chai = require('chai')
const chaiAsPromised = require('chai-as-promised')

chai.use(chaiAsPromised)
const expect = chai.expect

import { balanceByAssetAlias, client, createAccount, createAsset, signer } from '../testHelpers/util'

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
      .then((issuance) => signer.sign(issuance))
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
      return  Promise.all([
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
      .then((issuance) => signer.sign(issuance))
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
        return signer.sign(swapProposal)
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
        .then((swapTx) => signer.sign(swapTx))
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
    .then(issuance => signer.sign(issuance))
    .then(signed => expect(client.transactions.submit(signed)).to.be.rejectedWith('CH735'))
  })

  // These just test that the callback is engaged correctly. Behavior is
  // tested in the promises test.
  describe('Callback support', () => {

    it('Transaction query', (done) => {
      client.transactions.query(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('Transaction build', (done) => {
      client.transactions.build(
        () => {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('Batch transaction build', (done) => {
      client.transactions.buildBatch(
        [() => {}, () => {}], // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('Transaction submit', (done) => {
      client.transactions.submit(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('Batch transaction submit', (done) => {
      client.transactions.submitBatch(
        [{}, {}], // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

  })
})
