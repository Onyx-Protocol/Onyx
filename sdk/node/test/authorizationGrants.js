/* eslint-env mocha */

const chain = require('../dist/index.js')
const uuid = require('uuid')
const chai = require('chai')
const chaiAsPromised = require('chai-as-promised')

chai.use(chaiAsPromised)
const expect = chai.expect

const client = new chain.Client()

let tokenName, tokenGrant

describe('Authorization grant', () => {

  before('set up grant data', () => {
    tokenName = uuid.v4()
    return client.accessTokens.create({ id: tokenName }).then(resp => {
      tokenGrant = {
        guard_data: { id: resp.id },
        guard_type: 'access_token',
        policy: 'client-readwrite'
      }
    })
  })

  it('creation successful', () => {
    return client.authorizationGrants.create(tokenGrant)
      .then(resp => expect(resp.createdAt).not.to.be.empty)
  })

  it('creation rejected due to invalid ID', () => {
    return expect(client.authorizationGrants.create({
      guard_data: { id: 'invalidId' },
      guard_type: 'access_token',
      policy: 'client-readwrite'
    })).to.be.rejectedWith('CH303')
  })

  it('returned in list after creation', () => {
    return client.authorizationGrants.create(tokenGrant)
      .then(() => client.authorizationGrants.list())
      .then(resp => expect(resp.items.map(item => item.guardData.id)).to.contain(tokenName))
  })

  it('deletion successful', () => {
    return client.authorizationGrants.delete(tokenGrant)
      .then(resp => expect(resp.message).to.equal('ok'))
  })

  it('removed from list after deletion', () => {
    return client.authorizationGrants.create(tokenGrant)
      .then(() => client.authorizationGrants.delete(tokenGrant))
      .then(() => client.authorizationGrants.list())
      .then(resp => expect(resp.items.map(item => item.guardData.id)).to.not.contain(tokenName))
  })

  it('sanitizes X509 guard data', () => {
    return client.authorizationGrants.create({
        guardType: 'x509',
        guardData: {
          subject: {
            cn: tokenName,
            ou: 'test-ou',
          },
        },
        policy: 'client-readwrite',
      })
      .then(g => {
        delete g.createdAt // ignore timestamp

        expect(g).deep.equals({
          guardType: 'x509',
          guardData: {
            subject: {
              cn: tokenName,
              ou: ['test-ou'],
            }
          },
          policy: 'client-readwrite',
          protected: false,
        })
      })
  })

  // These just test that the callback is engaged correctly. Behavior is
  // tested in the promises test.
  describe('Callback support', () => {

    it('create', (done) => {
      client.authorizationGrants.create(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('delete', (done) => {
      client.authorizationGrants.delete(
        {}, // intentionally blank
        () => done() // intentionally ignore errors
      )
    })

    it('list', (done) => {
      client.authorizationGrants.list(done)
    })
  })
})
