const chain = require('chain-sdk')

let client

describe('transactions', () => {
  before(() => {
    client = new chain.Client()

    return expect(testHelpers.ensureConfigured()).to.be.fulfilled
      .then(() => expect(testHelpers.setUpObjects(client)).to.be.fulfilled)
      .then(() => expect(testHelpers.issueTransaction(client)).to.be.fulfilled)
  })

  describe('list view', () => {
    before(() => {
      return browser.url('/transactions')
    })

    it('does not display a welcome message', () => {
      browser.isExisting('.EmptyList').should.equal(false)
    })

    it('lists all blockchain transactions', () => {
      browser.getText('.ItemList').should.contain('alice')
      browser.getText('.ItemList').should.contain('gold')
      browser.getText('.ItemList').should.contain('100')
    })

    it('displays the correct page title', () => {
      browser.getText('.PageTitle').should.contain('Transactions')
      browser.getText('.PageTitle').should.contain('New transaction')
    })
  })

  describe('New transaction form', () => {
    beforeEach(() => {
      return browser.url('/transactions/create')
    })

    it('disables the submit button with no actions', () => {
      browser.waitForVisible('button=Submit transaction')
      browser.getAttribute('button=Submit transaction', 'disabled').should.equal('true')
    })

    it('returns an error with incomplete actions', () => {
      browser.waitForVisible('.AddActionDropdown button')
      browser.scroll('.AddActionDropdown button')
      browser.click('.AddActionDropdown button')
      browser.click('=Issue')

      browser.scroll('button=Submit transaction')
      browser.click('button=Submit transaction')
      browser.waitForVisible('.ErrorBanner')
      browser.getText('.ErrorBanner').should.contain('One or more actions had an error')
    })

    it('returns an unbalanced transaction error with a single action', () => {
      browser.waitForVisible('.AddActionDropdown button')
      browser.scroll('.AddActionDropdown button')
      browser.click('.AddActionDropdown button')
      browser.click('=Issue')

      browser.setValue('.ActionItem:nth-child(1) .ObjectSelectorField.Asset input', 'gold')
      browser.setValue('input[name="actions[0].amount"]', 1)

      browser.scroll('button=Submit transaction')
      browser.click('button=Submit transaction')
      browser.waitForVisible('.ErrorBanner')
      browser.getText('.ErrorBanner').should.contain('leaves assets to be taken without requiring payment')
    })

    it('successfully submits a transaction', () => {
      browser.waitForVisible('.AddActionDropdown button')

      browser.scroll('.AddActionDropdown button')
      browser.click('.AddActionDropdown button')
      browser.click('=Issue')
      browser.setValue('.ActionItem:nth-child(1) .ObjectSelectorField.Asset input', 'gold')
      browser.setValue('input[name="actions[0].amount"]', 1)

      browser.scroll('.AddActionDropdown button')
      browser.click('.AddActionDropdown button')
      browser.click('=Retire')
      browser.setValue('.ActionItem:nth-child(2) .ObjectSelectorField.Asset input', 'gold')
      browser.setValue('input[name="actions[1].amount"]', 1)

      browser.scroll('button=Submit transaction')
      browser.click('button=Submit transaction')

      browser.waitForVisible('.TransactionDetail')
      browser.getText('.TransactionDetail ').should.contain('Submitted transaction. Create another?')
    })

    it('generates transactions that allow additional actions', () => {
      browser.waitForVisible('.AddActionDropdown button')

      browser.scroll('.AddActionDropdown button')
      browser.click('.AddActionDropdown button')
      browser.click('=Issue')
      browser.setValue('.ActionItem:nth-child(1) .ObjectSelectorField.Asset input', 'gold')
      browser.setValue('input[name="actions[0].amount"]', 1)

      browser.scroll('.AddActionDropdown button')
      browser.click('.AddActionDropdown button')
      browser.click('=Control with account')
      browser.setValue('.ActionItem:nth-child(2) .ObjectSelectorField.Asset input', 'gold')
      browser.setValue('.ActionItem:nth-child(2) .ObjectSelectorField.Account input', 'alice')
      browser.setValue('input[name="actions[1].amount"]', 1)

      browser.click('=Show advanced options')
      browser.scroll('#submit_action_generate')
      browser.click('#submit_action_generate')

      browser.scroll('button=Generate transaction hex')
      browser.click('button=Generate transaction hex')

      browser.waitForVisible('.GeneratedTxHex')
      browser.getText('.GeneratedTxHex ').should.contain('Use the following hex string as the base transaction for a future transaction')
    })

  })
})
