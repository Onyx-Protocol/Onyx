/* eslint-env mocha */

const Connection = require('../dist/connection.js')
const chai = require('chai')
const expect = chai.expect


describe('camelizer', () => {
  it('converts non-blacklisted keys and children', () => {
    const base = {
      test_leaf: 1,
      test_child: {
        convert_this: 2
      }
    }

    const camelized = Connection.camelize(base)
    expect(camelized.test_leaf).equals(undefined)
    expect(camelized.testLeaf).equals(1)
    expect(camelized.testChild.convert_this).equals(undefined)
    expect(camelized.testChild.convertThis).equals(2)
  })

  it('does not convert children of blacklisted keys', () => {
    const base = {
      test_leaf: 1,
      reference_data: {
        dont_convert: 2
      }
    }

    const camelized = Connection.camelize(base)
    expect(camelized.test_leaf).equals(undefined)
    expect(camelized.testLeaf).equals(1)
    expect(camelized.referenceData.dont_convert).equals(2)
    expect(camelized.referenceData.dontConvert).equals(undefined)
  })

})

describe('snakeizer', () => {
  it('converts non-blacklist keys and children', () => {

  })
})
