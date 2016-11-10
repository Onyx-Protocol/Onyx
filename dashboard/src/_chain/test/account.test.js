import assert from 'assert'
import nock from 'nock'

import { Account, AccountPage } from './account'
import Context from './context'

function nockHelper() {
  return nock('http://test-server').defaultReplyHeaders({
    'Content-Type': 'application/json',
    'Chain-Request-Id': '1'
  })
}

describe('Account', () => {

  let ctx = new Context({url: 'http://test-server'})

  describe('#create', () => {
    it('creates a new account', () => {
      nockHelper().post('/create-account').reply(200, {
        xpubs: ['a'],
        quorum: 1,
        tags: {label: 'hello'}
      })

      return Account.create(ctx, {
        xpubs: ['a'],
        quorum: 1,
        tags: {label: 'hello'}
      }).then((res) => {
        assert.deepEqual(res, {
          xpubs: ['a'],
          quorum: 1,
          tags: {label: 'hello'}
        })
      })
    })
  })

  describe('#query', () => {
    it('returns an AccountPage containing Accounts', () => {
      nockHelper().post('/list-accounts').reply(200, {
        items: [
          {id: 'alice'},
          {id: 'bob'},
          {id: 'carol'},
        ],
        query: {
          cursor: 'carol'
        },
        last_page: true
      })

      return Account.query(ctx).then((page) => {
        assert(page instanceof AccountPage)
        assert.equal(page.items.length, 3)
        assert(page.items[0] instanceof Account)
        assert.equal(page.items[0].id, 'alice')
        assert.equal(page.query.cursor, 'carol')
        assert(page.last_page)
      })
    })
  })

})
