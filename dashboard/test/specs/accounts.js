const chain = require('chain-sdk')
const uuid = require('uuid')

let signer

describe('accounts', () => {
  describe('list view', () => {
    before(() => {
      signer = new chain.HsmSigner()

      return expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => expect(testHelpers.setUpObjects(signer)).to.be.fulfilled)
        .then(() => browser.url('/accounts'))
    })

    it('does not display a welcome message', () => {
      browser.isExisting('.EmptyList').should.equal(false)
    })

    it('lists all accounts on the core', () => {
      browser.getText('.ItemList').should.contain('ACCOUNT ALIAS')
      browser.getText('.ItemList').should.contain('alice')
      browser.getText('.ItemList').should.contain('View details')
    })

    it('displays the correct page title', () => {
      browser.getText('.PageTitle').should.contain('Accounts')
      browser.getText('.PageTitle').should.contain('New account')
    })
  })

  describe('creating accounts', () => {
    before(() => expect(testHelpers.ensureConfigured()).to.be.fulfilled
      .then(() => browser.url('/accounts'))
    )

    it('can create a new account', () => {
      const alias = 'test-account-' + uuid.v4()
      browser.click('.ItemList button')
      browser.setValue('input[name=alias]', alias)
      browser.click('.FormContainer button')
      browser.waitForVisible('.AccountShow')
      browser.getText('.AccountShow').should.contain('Created account. Create another?')
      browser.getText('.AccountShow').should.contain(alias)
    })
  })
})
