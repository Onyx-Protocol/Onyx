describe('tutorial', () => {
  before(() =>
    expect(testHelpers.ensureConfigured()).to.be.fulfilled
      .then(() => browser.url('/'))
  )

  it('should start/resume/end the tutorial', function() {
    browser.waitForVisible('.ItemList')
    browser.scroll('a=Tutorial')
    browser.click('a=Tutorial')
    browser.waitForVisible('.TutorialModal')
    browser.getText('.TutorialModal').should.contain('Would you like a brief tutorial?')
    browser.scroll('button=Start 5-minute tutorial')
    browser.click('button=Start 5-minute tutorial')
    browser.getUrl().should.contain('/mockhsms')
    browser.waitForVisible('.TutorialHeaderOpened')
    browser.getText('.TutorialHeaderOpened').should.contain('Tutorial - Creating keys')
    browser.url('/transactions')
    browser.getText('.TutorialHeaderOpened').should.contain('Resume tutorial')
    browser.scroll('a=Resume tutorial')
    browser.click('a=Resume tutorial')
    browser.getText('.TutorialHeaderOpened').should.contain('End tutorial')
    browser.scroll('a=End tutorial')
    browser.click('a=End tutorial')
    browser.isExisting('.TutorialHeaderOpened').should.equal(false)
  })
})
