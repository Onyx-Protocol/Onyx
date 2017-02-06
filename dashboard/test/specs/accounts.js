const chain = require('chain-sdk')
const uuid = require('uuid')

const client = new chain.Client()
let signer

describe('accounts', () => {
  describe('list view', () => {
    before(() => {
      signer = new chain.HsmSigner()

      return expect(ensureConfigured()).to.be.fulfilled
        .then(() => expect(setUpObjects(signer)).to.be.fulfilled)
        .then(() => browser.url('/accounts'))
    })

    it('does not display a welcome message', () => {
      browser.isExisting('.component-EmptyList').should.equal(false)
    })

    it('lists all accounts on the core', () => {
      browser.getText('.component-ItemList').should.contain('ACCOUNT ALIAS')
      browser.getText('.component-ItemList').should.contain('alice')
      browser.getText('.component-ItemList').should.contain('View details')
    })

    it('displays the correct page title', () => {
      browser.getText('.component-PageTitle').should.contain('Accounts')
      browser.getText('.component-PageTitle').should.contain('New account')
    })
  })

  describe('creating accounts', () => {
    before(() => expect(ensureConfigured()).to.be.fulfilled
      .then(() => browser.url('/accounts'))
    )

    it('can create a new account', () => {
      const alias = 'test-account-' + uuid.v4()
      browser.click('.component-ItemList button')
      browser.setValue('input[name=alias]', alias)
      browser.click('.component-FormContainer button')
      browser.waitForVisible('.component-AccountShow')
      browser.getText('.component-AccountShow').should.contain('Created account. Create another?')
      browser.getText('.component-AccountShow').should.contain(alias)
    })
  })
})
