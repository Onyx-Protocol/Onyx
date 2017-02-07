const chain = require('chain-sdk')

const client = new chain.Client()
let signer

describe('empty states', () => {
  before(() => expect(testHelpers.resetCore()).to.be.fulfilled)

  describe('transactions', () => {
    it('displays a welcome message', () => {
      browser.url('/transactions')
      browser.getText('.component-EmptyList').should.contain('Welcome to Chain Core')
      browser.getText('.component-EmptyList').should.contain('New transaction')
    })
  })

  describe('assets', () => {
    it('displays documentation links', () => {
      browser.url('/assets')
      browser.getText('.component-EmptyList').should.contain('Learn more about how to use assets.')
      browser.getText('.component-EmptyList').should.contain('New asset')
    })
  })

  describe('accounts', () => {
    it('displays documentation links', () => {
      browser.url('/accounts')
      browser.getText('.component-EmptyList').should.contain('Learn more about how to use accounts.')
      browser.getText('.component-EmptyList').should.contain('New account')
    })
  })
})
