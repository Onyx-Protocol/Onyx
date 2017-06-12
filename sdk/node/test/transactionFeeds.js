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

describe('Transaction feed', () => {
  let gold, silver, alice, bob, issuanceFeed, spendFeed

  before(() =>
    Promise.all([
      createAsset('gold'),
      createAsset('silver'),
      createAccount('alice'),
      createAccount('bob'),
      client.transactionFeeds.create({filter: "inputs(type='issue')"}),
      client.transactionFeeds.create({filter: "inputs(type='spend')"}),
    ]).then(res => {
      [
        gold,
        silver,
        alice,
        bob,
        issuanceFeed,
        spendFeed,
      ] = res

      return Promise.resolve()
    })
  )

  it('consumes filtered transactions', () => {
    const feedIssuances = []
    const feedSpends = []
    const submittedIssuances = []
    const submittedSpends = []

    return Promise.resolve()
      .then(() =>
        Promise.all([
          createAsset('gold'),
          createAsset('silver'),
          createAccount('alice'),
          createAccount('bob'),
          client.transactionFeeds.create({filter: "inputs(type='issue')"}),
          client.transactionFeeds.create({filter: "inputs(type='spend')"}),
        ])
      ).then(res =>
        [
          gold,
          silver,
          alice,
          bob,
          issuanceFeed,
          spendFeed,
        ] = res
      ).then(() =>
        Promise.all([

          // Start consuming issuances
          issuanceFeed.consume((tx, next, done) => {
            feedIssuances.push(tx.id)
            feedIssuances.length == 2 ? done(true) : next(true)
          }),

          // Start consuming spends
          spendFeed.consume((tx, next, done) => {
            feedSpends.push(tx.id)
            feedSpends.length == 2 ? done(true) : next(true)
          }),

          // Publish a series of transactions
          Promise.resolve().then(() =>
            buildSignSubmit(builder => {
              builder.issue({assetAlias: gold.alias, amount: 1})
              builder.controlWithAccount({accountAlias: alice.alias, assetAlias: gold.alias, amount: 1})
            })
          ).then(tx =>
            submittedIssuances.push(tx.id)
          ).then(() =>
            buildSignSubmit(builder => {
              builder.spendFromAccount({accountAlias: alice.alias, assetAlias: gold.alias, amount: 1})
              builder.controlWithAccount({accountAlias: bob.alias, assetAlias: gold.alias, amount: 1})
            })
          ).then(tx =>
            submittedSpends.push(tx.id)
          ).then(() =>
            buildSignSubmit(builder => {
              builder.issue({assetAlias: silver.alias, amount: 1})
              builder.controlWithAccount({accountAlias: bob.alias, assetAlias: silver.alias, amount: 1})
            })
          ).then(tx =>
            submittedIssuances.push(tx.id)
          ).then(() =>
            buildSignSubmit(builder => {
              builder.spendFromAccount({accountAlias: bob.alias, assetAlias: silver.alias, amount: 1})
              builder.controlWithAccount({accountAlias: alice.alias, assetAlias: silver.alias, amount: 1})
            })
          ).then(tx =>
            submittedSpends.push(tx.id)
          ),

        ])
      ).then(() => {
        expect(feedIssuances).to.deep.equal(submittedIssuances)
        expect(feedSpends).to.deep.equal(submittedSpends)
      })
  })

  describe('queryAll', () => {
    it('success example', () => {
      let created
      const queried = []

      return client.transactionFeeds.create({}).then(txfeed =>
        created = txfeed.id
      ).then(() =>
        client.transactionFeeds.queryAll({}, (txfeed, next, done) => {
          queried.push(txfeed.id)
          next()
        })
      ).then(() => {
        expect(queried).to.include(created)
      })
    })
  })

  // These just test that the callback is engaged correctly. Behavior is
  // tested in the promises test.
  describe('Callback support', () => {
    it('create', done => {
      client.transactionFeeds.create(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('query', done => {
      client.transactionFeeds.query({}, done)
    })

    it('queryAll', done => {
      client.transactionFeeds.queryAll(
        {},
        (t, next, queryDone) => queryDone(),
        done
      )
    })
  })
})
