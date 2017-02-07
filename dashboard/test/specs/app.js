describe('dashboard', () => {
  describe('homepage', () => {
    before(() =>
      expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => browser.url('/'))
    )

    it('should load the page', function() {
      browser.getTitle().should.equal('Chain Core Dashboard')
    })

    it('should redirect to /transactions', function() {
      browser.getUrl().should.contain('/transactions')
    })
  })
})
