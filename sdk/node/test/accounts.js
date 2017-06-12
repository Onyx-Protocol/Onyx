/* eslint-env mocha */

const chain = require('../dist/index.js')
const uuid = require('uuid')
const assert = require('assert')
const chai = require('chai')
const chaiAsPromised = require('chai-as-promised')

chai.use(chaiAsPromised)
const expect = chai.expect

const client = new chain.Client()
const xAccountAlias = `x-${uuid.v4()}`
const yAccountAlias = `y-${uuid.v4()}`

let mockHsmKey

describe('Account', () => {

  before('set up API objects', () => {

    // Key and account creation
    return client.mockHsm.keys.create()
      .then(key => { mockHsmKey = key })
      .then(() => {
        return client.accounts.create({alias: xAccountAlias, rootXpubs: [mockHsmKey.xpub], quorum: 1, tags: {x: 0}})
      })
      .then(() => {
        return client.accounts.create({alias: yAccountAlias, rootXpubs: [mockHsmKey.xpub], quorum: 1, tags: {y: 0}})
      })
  })

  describe('Single account creation', () => {

    it('successful', () => {
      return client.accounts.create({alias: `alice-${uuid.v4()}`, rootXpubs: [mockHsmKey.xpub], quorum: 1})
        .then(resp => expect(resp.id).not.to.be.empty)
    })

    it('rejected due to missing key fields', () => {
      return expect(client.accounts.create({alias: 'david'})).to.be.rejectedWith('CH202')
    })
  })

  describe('Batch account creation', () => {
    let batchResponse = {}

    before(() => client.accounts.createBatch([
        {alias: `carol-${uuid.v4()}`, rootXpubs: [mockHsmKey.xpub], quorum: 1}, // success
        {alias: 'david'}, // failure
        {alias: `eve-${uuid.v4()}`, rootXpubs: [mockHsmKey.xpub], quorum: 1}, // success
      ])
      .then(resp => {batchResponse = resp})
    )

    it('returns two successes', () => assert.equal(batchResponse.successes[1], null))
    it('returns one error', () => assert.deepEqual([batchResponse.errors[0], batchResponse.errors[2]], [null, null]))
  })

  describe('Single account tags update', () => {

    it('successful', () => {
      return client.accounts.updateTags({
        alias: xAccountAlias,
        tags: {x: 1},
      })
      .then(() => {
        return client.accounts.query({
          filter: `alias='${xAccountAlias}'`
        })
      })
      .then(page => {
        assert.deepEqual(page.items[0].tags, {x: 1})
      })
    })

    it('rejected due to missing ID/Alias', () => {
      return expect(
        client.accounts.updateTags({
          // ID/Alias intentionally omitted
          tags: {x: 1},
        })
      ).to.be.rejectedWith('CH051')
    })
  })

  describe('Batch account tags update', () => {

    it('successful', () => {
      return client.accounts.updateTagsBatch([{
        alias: xAccountAlias,
        tags: {x: 2},
      }, {
        alias: yAccountAlias,
        tags: {y: 2},
      }])
      .then(() => {
        return client.accounts.query({
          filter: `alias='${xAccountAlias}' OR alias='${yAccountAlias}'`
        })
      })
      .then(page => {
        assert.deepEqual(page.items.find(i => i.alias.match(/^x-/)).tags, {x: 2})
        assert.deepEqual(page.items.find(i => i.alias.match(/^y-/)).tags, {y: 2})
      })
    })

    it('fails to update accounts tags with missing ID/Alias', () => {
      return client.accounts.updateTagsBatch([{
        alias: xAccountAlias,
        tags: {x: 3},
      }, {
        // ID/Alias intentionally omitted
        tags: {y: 3}
      }])
      .then(batch => {
        assert(batch.successes[0])
        assert(!batch.successes[1])
        assert(!batch.errors[0])
        assert(batch.errors[1])
      })
    })
  })

  describe('queryAll', () => {
    it('success example', () => {
      let created
      const queried = []

      return client.accounts.create({
        rootXpubs: [mockHsmKey.xpub],
        quorum: 1
      }).then(account =>
        created = account.id
      ).then(() =>
        client.accounts.queryAll({}, (account, next, done) => {
          queried.push(account.id)
          next()
        })
      ).then(() =>
        expect(queried).to.include(created)
      )
    })
  })

  // These just test that the callback is engaged correctly. Behavior is
  // tested in the promises test.
  describe('Callback support', () => {

    it('create', (done) => {
      client.accounts.create(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('createBatch', (done) => {
      client.accounts.createBatch(
        [{}, {}], // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('updateTags', (done) => {
      client.accounts.updateTags(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('updateTagsBatch', (done) => {
      client.accounts.updateTagsBatch(
        [{}, {}], // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('query', done => {
      client.accounts.query({}, done)
    })

    it('queryAll', done => {
      client.accounts.queryAll(
        {},
        (t, next, queryDone) => queryDone(),
        done
      )
    })

  })
})
