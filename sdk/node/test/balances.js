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

describe('Balance', () => {
  let asset1, asset2, account

  before(() =>
    Promise.all([
      createAsset(),
      createAsset(),
      createAccount(),
    ]).then(res =>
      [asset1, asset2, account] = res
    ).then(() =>
      buildSignSubmit(b => {
        b.issue({assetId: asset1.id, amount: 1})
        b.controlWithAccount({
          accountId: account.id,
          assetId: asset1.id,
          amount: 1,
        })
      })
    ).then(() =>
      buildSignSubmit(b => {
        b.issue({assetId: asset2.id, amount: 2})
        b.controlWithAccount({
          accountId: account.id,
          assetId: asset2.id,
          amount: 2,
        })
      })
    )
  )

  describe('Query with filter', () => {
    it('simple example', () =>
      client.balances.query({
        filter: "asset_id=$1",
        filterParams: [asset1.id],
        sumBy: ['asset_id'],
      }).then(page =>
        expect(page.items[0].amount).to.equal(1)
      )
    )
  })

  describe('queryAll', () => {
    it('simple example', () => {
      let queried = []

      return client.balances.queryAll({
        filter: "account_id=$1",
        filterParams: [account.id],
        sumBy: ['asset_id']
      }, (balance, next, done) => {
        queried.push(balance)
        next()
      }).then(() => {
        expect(queried.find(b => b.sumBy.assetId == asset1.id).amount).to.equal(1)
        expect(queried.find(b => b.sumBy.assetId == asset2.id).amount).to.equal(2)
      })
    })
  })

  describe('Callback support', () => {

    it('query', done => {
      client.balances.query({}, done)
    })

    it('queryAll', done => {
      client.balances.queryAll(
        {},
        (t, next, queryDone) => queryDone(),
        done
      )
    })

  })

})
