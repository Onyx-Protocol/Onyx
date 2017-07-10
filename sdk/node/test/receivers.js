
/* eslint-env mocha */

const assert = require('assert')
const chai = require('chai')
const chaiAsPromised = require('chai-as-promised')

chai.use(chaiAsPromised)
const expect = chai.expect

import { client, createAccount, createAsset } from './testHelpers'

describe('Receiver', () => {

  describe('Single receiver creation', () => {
    it('succesfully creates receiver by account alias', () => {
      return createAccount()
        .then((account) => client.accounts.createReceiver({ accountAlias: account.alias }))
        .then((resp) => expect(resp.controlProgram).not.to.be.empty)
    })

    it('succesfully creates receiver by account id', () => {
      return createAccount()
        .then((account) => client.accounts.createReceiver({ accountId: account.id }))
        .then((resp) => expect(resp.controlProgram).not.to.be.empty)
    })

    it('rejects receiver creation due to missing params', () => {
      return createAccount()
        .then((account) => expect(client.accounts.createReceiver({})).to.be.rejectedWith('CH002'))
    })
  })

  describe('Batch receiver creation', () => {
    let batchResponse = {}

    before(() => {
      return createAccount()
        .then((account) => client.accounts.createReceiverBatch([
          { accountId: account.id }, // success
          {}, // error
          { accountAlias: account.alias } // success
        ]))
        .then((resp) => {batchResponse = resp})
    })

    it('returns two successes', () => assert.equal(batchResponse.successes[1], null))
    it('returns one error', () => assert.deepEqual([batchResponse.errors[0], batchResponse.errors[2]], [null, null]))
  })

  describe('Pay to receiver', () => {
    let testAccountAlias, testAssetAlias, testAccountReceiver

    before(() => {
      return Promise.all([
        createAccount(),
        createAsset()
      ])
      .then((objects) => {
        testAccountAlias = objects[0].alias
        testAssetAlias = objects[1].alias
      })
      .then(() => client.accounts.createReceiver({ accountAlias: testAccountAlias }))
      .then((receiver) => testAccountReceiver = receiver)
    })

    it('pays assets to the receiving account', () => {
      return client.transactions.build(builder => {
        builder.issue({
          assetAlias: testAssetAlias,
          amount: 1,
        })
        builder.controlWithReceiver({
          receiver: testAccountReceiver,
          assetAlias: testAssetAlias,
          amount: 1,
        })
      })
    .then((issuance) => client.transactions.sign(issuance))
    .then((signed) => client.transactions.submit(signed))
    .then((tx) => expect(tx.id).not.to.be.empty)
    })
  })

  // These just test that the callback is engaged correctly. Behavior is
  // tested in the promises test.
  describe('Callback support', () => {

    it('Single receiver creation', (done) => {
      client.accounts.createReceiver(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('Batch receiver creation', (done) => {
      client.accounts.createReceiverBatch(
        [{}, {}], // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

  })
})
