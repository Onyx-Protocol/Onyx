const chain = require('chain-sdk')

const client = new chain.Client()
let signer

describe('tranasctions', () => {
  describe('list view', () => {
    before(() => {
      signer = new chain.HsmSigner()

      return expect(ensureConfigured()).to.be.fulfilled
        .then(() => expect(setUpObjects(signer)).to.be.fulfilled)
        .then(() => expect(issueTransaction(signer)).to.be.fulfilled)
        .then(() => browser.url('/transactions'))
    })

    it('does not display a welcome message', () => {
      browser.isExisting('.component-EmptyList').should.equal(false)
    })

    it('lists all blockchain transactions', () => {
      browser.getText('.component-ItemList').should.contain('alice')
      browser.getText('.component-ItemList').should.contain('gold')
      browser.getText('.component-ItemList').should.contain('100')
    })

    it('displays the correct page title', () => {
      browser.getText('.component-PageTitle').should.contain('Transactions')
      browser.getText('.component-PageTitle').should.contain('New transaction')
    })
  })
})
