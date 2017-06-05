/* eslint-env mocha */

const chain = require('../dist/index.js')
const uuid = require('uuid')
const chai = require('chai')
const chaiAsPromised = require('chai-as-promised')

chai.use(chaiAsPromised)
const expect = chai.expect

const client = new chain.Client()

describe('MockHSM key', () => {

  it('succesfully creates key', () => {
    return client.mockHsm.keys.create()
      .then((resp) => expect(resp).not.to.be.empty)
  })

  it('rejects key creation due to duplicate alias', () => {
    return client.mockHsm.keys.create({ alias: `key-${uuid.v4()}` })
      .then((resp) => {
        return expect(client.mockHsm.keys.create({ alias: resp.alias })).to.be.rejectedWith('CH050')
      })
  })

  it('returns key in list after key creation', () => {
    let keyAlias
    return client.mockHsm.keys.create({ alias: `key-${uuid.v4()}` })
      .then((key) => {
        keyAlias = key.alias
        return client.mockHsm.keys.query({})
      })
      .then(resp => {
        return expect(resp.items.map(item => item.alias)).to.contain(keyAlias)
      })
  })

  // These just test that the callback is engaged correctly. Behavior is
  // tested in the promises test.
  describe('Callback style', () => {

    it('Key creation', (done) => {
      client.mockHsm.keys.create(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('Key querying', (done) => {
      client.mockHsm.keys.query(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

  })
})
