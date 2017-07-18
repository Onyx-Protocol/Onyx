const chain = require('chain-sdk')

let client

describe('unspent outputs', () => {
  describe('list view', () => {
    before(() => {
      client = new chain.Client()

      return expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => expect(testHelpers.setUpObjects(client)).to.be.fulfilled)
        .then(() => expect(testHelpers.issueTransaction(client)).to.be.fulfilled)
        .then(() => browser.url('/unspents'))
    })

    it('does not display a welcome message', () => {
      browser.isExisting('.EmptyList').should.equal(false)
    })

    it('displays the correct page title', () => {
      browser.getText('.PageTitle').should.contain('Unspent outputs')
    })
  })
})
