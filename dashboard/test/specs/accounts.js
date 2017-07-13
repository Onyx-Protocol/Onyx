const uuid = require('uuid')

describe('accounts', () => {
  describe('list view', () => {
    before(() => {
      return expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => expect(testHelpers.setUpObjects()).to.be.fulfilled)
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
      browser.waitForVisible('.ItemList button')
      browser.scroll('.ItemList button')
      browser.click('.ItemList button')
      browser.setValue('input[name=alias]', alias)
      browser.scroll('.FormContainer input[type=radio][value=generate]')
      browser.click('.FormContainer input[type=radio][value=generate]')
      browser.scroll('.FormContainer button')
      browser.click('.FormContainer button')
      browser.waitForVisible('.AccountShow')
      browser.getText('.AccountShow').should.contain('Created account. Create another?')
      browser.getText('.AccountShow').should.contain(alias)
    })
  })

  describe('creating receivers', () => {
    before(() => expect(testHelpers.ensureConfigured()).to.be.fulfilled
      .then(() => testHelpers.createAccount('receiver-test'))
      .then((account) => browser.url('/accounts/' + account.id))
    )

    it('can create a new receiver', () => {
      browser.waitForVisible('.AccountShow')
      browser.scroll('button=Create receiver')
      browser.click('button=Create receiver')
      browser.waitForVisible('.ReceiverModal')
      browser.getText('.ReceiverModal').should.contain('Copy this one-time use receiver to use in a transaction')
    })
  })
})
