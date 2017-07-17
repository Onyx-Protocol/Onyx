describe('access control', () => {
  describe('token list view', () => {
    before(() => {
      return expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => browser.url('/access-control'))
    })

    it('should redirect to /access-control?type=token', function() {
      browser.getUrl().should.contain('type=token')
    })

    it('displays the correct page', () => {
      browser.getText('.PageTitle').should.contain('Access control')
      browser.getText('thead').should.contain('TOKEN NAME')

    })
  })
  describe('certificate list view', () => {
    before(() => {
      return expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => browser.url('/access-control?type=certificate'))
    })

    it('displays the correct page', () => {
      browser.getText('.PageTitle').should.contain('Access control')
      browser.getText('thead').should.contain('CERTIFICATE')
    })
  })
})
