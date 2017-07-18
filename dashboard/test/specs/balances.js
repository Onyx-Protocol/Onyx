describe('balances', () => {
  describe('list view', () => {
    before(() => {
      return expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => expect(testHelpers.setUpObjects()).to.be.fulfilled)
        .then(() => browser.url('/balances'))
    })

    it('displays the correct page title', () => {
      browser.getText('.PageTitle').should.contain('Balances')
    })
  })
})
