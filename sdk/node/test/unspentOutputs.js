/* eslint-env mocha */

const chai = require('chai')
const chaiAsPromised = require('chai-as-promised')
chai.use(chaiAsPromised)
const expect = chai.expect

const {
  client,
  createAccount,
  createAsset,
  buildSignSubmit,
} = require('./testHelpers')

describe('Unspent output', () => {
  let asset, account1, account2, tx1, tx2

  before(() =>
    Promise.all([
      createAsset(),
      createAccount(),
      createAccount(),
    ]).then(res =>
      [asset, account1, account2] = res
    ).then(() =>
      buildSignSubmit(b => {
        b.issue({assetId: asset.id, amount: 1})
        b.controlWithAccount({
          accountId: account1.id,
          assetId: asset.id,
          amount: 1,
        })
      }).then(tx => tx1 = tx.id)
    ).then(() =>
      buildSignSubmit(b => {
        b.issue({assetId: asset.id, amount: 1})
        b.controlWithAccount({
          accountId: account2.id,
          assetId: asset.id,
          amount: 1,
        })
      }).then(tx => tx2 = tx.id)
    )
  )

  describe('Query with filter', () => {
    it('simple example', () =>
      client.unspentOutputs.query({
        filter: "account_id=$1",
        filterParams: [account1.id],
      }).then(page =>
        expect(page.items.map(output => output.transactionId)).to.include(tx1)
      )
    )
  })

  describe('queryAll', () => {
    it('simple example', () => {
      let queried = []

      return client.unspentOutputs.queryAll({
        filter: "asset_id=$1",
        filterParams: [asset.id],
      }, (utxo, next, done) => {
        queried.push(utxo.transactionId)
        next()
      }).then(() => {
        expect(queried).to.include(tx1)
        expect(queried).to.include(tx2)
      })
    })
  })

  describe('Callback support', () => {
    it('query', done => {
      client.unspentOutputs.query({}, done)
    })

    it('queryAll', done => {
      client.unspentOutputs.queryAll(
        {},
        (t, next, queryDone) => queryDone(),
        done
      )
    })
  })

})
