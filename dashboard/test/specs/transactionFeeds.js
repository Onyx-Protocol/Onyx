const uuid = require('uuid')

describe('transaction feeds', () => {
  describe('list view', () => {
    before(() => {
      return expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => browser.url('/transaction-feeds'))
    })

    it('displays the correct page title', () => {
      browser.getText('.PageTitle').should.contain('Transaction feeds')
    })
  })

  describe('creating transaction feeds', () => {
    before(() => {
      return expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => browser.url('/transaction-feeds'))
    })

    it('can create a new transaction feed', () => {
      const alias = 'test-tx-feed-' + uuid.v4()
      browser.waitForVisible('.ItemList button')
      browser.scroll('.ItemList button')
      browser.click('.ItemList button')
      browser.setValue('input[name=alias]', alias)
      browser.scroll('.FormContainer button')
      browser.click('.FormContainer button')
      browser.waitForVisible('.ItemList')
      browser.getText('.ItemList').should.contain('Created transaction feed. Create another?')
      browser.getText('.ItemList').should.contain(alias)
    })
  })
})
