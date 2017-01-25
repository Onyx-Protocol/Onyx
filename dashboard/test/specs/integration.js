describe('dashboard', () => {
  describe('homepage', () => {
    before(() => {
      browser.url('/')
    })

    it('should load the page', function() {
      browser.getTitle().should.equal('Chain Core Dashboard')
    })

    it('should redirect to /transactinos', function() {
      browser.getUrl().should.contain('/transactions')
    })
  })
})
