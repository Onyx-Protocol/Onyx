const chain = require('chain-sdk')

const client = new chain.Client()
let signer

describe('accounts', () => {
  describe('list view', () => {
    before(() => {
      signer = new chain.HsmSigner()

      return expect(ensureConfigured()).to.be.fulfilled
        .then(() => expect(setUpObjects(signer)).to.be.fulfilled)
        .then(() => expect(issueTransaction(signer)).to.be.fulfilled)
        .then(() => browser.url('/assets'))
    })

    it('does not display a welcome message', () => {
      browser.isExisting('.component-EmptyList').should.equal(false)
    })

    it('lists all assets on the core', () => {
      browser.getText('.component-ItemList').should.contain('ASSET ALIAS')
      browser.getText('.component-ItemList').should.contain('gold')
      browser.getText('.component-ItemList').should.contain('View details')
    })

    it('displays the correct page title', () => {
      browser.getText('.component-PageTitle').should.contain('Assets')
      browser.getText('.component-PageTitle').should.contain('New asset')
    })
  })

  describe('empty state', () => {
    before(() =>
      expect(resetCore()).to.be.fulfilled
      .then(() => browser.url('/assets'))
    )

    it('displays a welcome message', () => {
      browser.getText('.component-EmptyList').should.contain('Learn more about how to use assets.')
      browser.getText('.component-EmptyList').should.contain('New asset')
    })
  })
})
