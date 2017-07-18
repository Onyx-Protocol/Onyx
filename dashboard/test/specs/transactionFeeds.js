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
})
