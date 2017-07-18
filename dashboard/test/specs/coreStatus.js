describe('core status', () => {
  describe('displaying status', () => {
    before(() =>
      expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => browser.url('/core'))
    )

    it('should load the page', function() {
      browser.getText('.PageTitle').should.contain('Core')
    })
  })
  describe('resetting data', () => {
    before(() =>
      expect(testHelpers.ensureConfigured()).to.be.fulfilled
        .then(() => browser.url('/core'))
    )

    it('should reset data', function() {
      browser.waitForVisible('.CoreIndex')
      browser.scroll('button=Delete all data')
      browser.click('button=Delete all data')
      browser.alertText().should.contain('Are you sure you want to delete all data on this core?')
      browser.alertAccept()
      browser.waitForVisible('.container')
      browser.getText('.container').should.contain('Configure Chain Core')
    })
  })
})
