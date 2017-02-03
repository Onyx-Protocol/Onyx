const chain = require('chain-sdk')

const client = new chain.Client()
let signer

describe('tranasctions', () => {
  describe('list view', () => {
    before(() => {
      signer = new chain.HsmSigner()

      expect(setUpObjects(client, signer)).to.be.fulfilled
        .then(() => expect(issueTransaction(client, signer)).to.be.fulfilled)
        .then(() => browser.url('/transactions'))
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

  describe('empty state', () => {
    before(() =>
      expect(client.config.reset()
        .then(() => {
          sleep(1000)
          return client.config.configure({ isGenerator: true })
        })
      ).to.be.fulfilled
      .then(() => browser.url('/transactions'))
    )

    it('displays a welcome message', () => {
      browser.getText('.component-EmptyList').should.contain('Welcome to Chain Core')
      browser.getText('.component-EmptyList').should.contain('New transaction')
    })
  })
})
