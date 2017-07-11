const chain = require('chain-sdk')

let client

describe('tranasctions', () => {
  describe('list view', () => {
    before(() => {
      client = new chain.Client()

      return expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => expect(testHelpers.setUpObjects(client)).to.be.fulfilled)
        .then(() => expect(testHelpers.issueTransaction(client)).to.be.fulfilled)
        .then(() => browser.url('/transactions'))
    })

    it('does not display a welcome message', () => {
      browser.isExisting('.EmptyList').should.equal(false)
    })

    it('lists all blockchain transactions', () => {
      browser.getText('.ItemList').should.contain('alice')
      browser.getText('.ItemList').should.contain('gold')
      browser.getText('.ItemList').should.contain('100')
    })

    it('displays the correct page title', () => {
      browser.getText('.PageTitle').should.contain('Transactions')
      browser.getText('.PageTitle').should.contain('New transaction')
    })
  })
})
