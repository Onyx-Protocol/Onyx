
/* eslint-env mocha */

const assert = require('assert')
const chai = require('chai')
const chaiAsPromised = require('chai-as-promised')

chai.use(chaiAsPromised)
const expect = chai.expect

import { client, createAccount, createAsset } from './testHelpers'

describe('Transaction batch', () => {
  let buildBatchResponse, signedBatchResponse, submittedBatchResponse = {}

  before('', () => {
    let aliceAlias, bobAlias, goldAlias, silverAlias

    return Promise.all([
      createAccount('alice'),
      createAccount('bob'),
      createAsset('gold'),
      createAsset('silver')
    ])
    .then((objects) => {
      aliceAlias = objects[0].alias
      bobAlias = objects[1].alias
      goldAlias = objects[2].alias
      silverAlias = objects[3].alias
    })
    .then(() => client.transactions.buildBatch([
      // Should succeed
      (builder) => {
        builder.issue({
          assetAlias: goldAlias,
          amount: 100
        })
        builder.controlWithAccount({
          accountAlias: aliceAlias,
          assetAlias: goldAlias,
          amount: 100
        })
      },

      // Should fail at the build step
      (builder) => {
        builder.issue({
          assetAlias: 'foobar'
        })
      },

      // Should fail at the submit step
      (builder) => {
        builder.issue({
          assetAlias: goldAlias,
          amount: 50
        })
        builder.controlWithAccount({
          accountAlias: aliceAlias,
          assetAlias: goldAlias,
          amount: 100
        })
      },

      // Should succeed
      (builder) => {
        builder.issue({
          assetAlias: silverAlias,
          amount: 50
        })
        builder.controlWithAccount({
          accountAlias: bobAlias,
          assetAlias: silverAlias,
          amount: 50
        })
      }]))
    .then(buildBatch => {
      buildBatchResponse = buildBatch
      return client.transactions.signBatch(buildBatch.successes)
    })
    .then(signedBatch => {
      signedBatchResponse = signedBatch
      return client.transactions.submitBatch(signedBatch.successes)
    })
    .then(submitBatch => {
      submittedBatchResponse = submitBatch
    })
  })

  describe('Build batch', () => {
    it('returns three successes', () => assert.equal(buildBatchResponse.successes[1], null))

    it('returns one error', () => {
      return assert.deepEqual([buildBatchResponse.errors[0],
        buildBatchResponse.errors[2],
        buildBatchResponse.errors[3]],
        [null, null, null])
    })
  })

  describe('Signed batch', () => {
    it('returns three sucesses', () => assert(signedBatchResponse.successes.indexOf(null) == -1))

    it('returns no errors', () => {
      return assert.deepEqual([signedBatchResponse.errors[0],
        signedBatchResponse.errors[1],
        signedBatchResponse.errors[2]],
        [null, null, null])
    })
  })

  describe('Submitted batch', () => {
    it('returns two successes', () => assert.equal(submittedBatchResponse.successes[1], null))

    it('returns one error', () => {
      return assert.deepEqual([submittedBatchResponse.errors[0],
        submittedBatchResponse.errors[2]],
        [null, null])
    })
  })
  // These just test that the callback is engaged correctly. Behavior is
  // tested in the promises test.
  describe('Callback support', () => {

    it('Batch transaction build', (done) => {
      client.transactions.buildBatch(
        [() => {}, () => {}], // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('Batch transaction sign', (done) => {
      client.transactions.signBatch(
        [], // intentionally blank
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
