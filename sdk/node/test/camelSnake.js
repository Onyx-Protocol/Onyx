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

  it('does not convert all-caps keys', () => {
    expect(
      Connection.camelize({
        convert_me: 1,
        DONTCONVERTME: 1,
      })
    ).deep.equals({
      convertMe: 1,
      DONTCONVERTME: 1,
    })
  })
})

describe('snakeizer', () => {
  it('converts non-blacklisted keys and children', () => {
    const base = {
      testLeaf: 1,
      testChild: {
        convertThis: 2
      }
    }

    const snakeized = Connection.snakeize(base)
    expect(snakeized.testLeaf).equals(undefined)
    expect(snakeized.test_leaf).equals(1)
    expect(snakeized.test_child.convertThis).equals(undefined)
    expect(snakeized.test_child.convert_this).equals(2)
  })

  it('does not convert children of blacklisted keys', () => {
    const base = {
      testLeaf: 1,
      referenceData: {
        dontConvert: 2
      }
    }

    const snakeized = Connection.snakeize(base)
    expect(snakeized.testLeaf).equals(undefined)
    expect(snakeized.test_leaf).equals(1)
    expect(snakeized.reference_data.dontConvert).equals(2)
    expect(snakeized.reference_data.dont_Convert).equals(undefined)
  })

  it('does not convert all-caps keys', () => {
    expect(
      Connection.snakeize({
        convertMe: 1,
        DONTCONVERTME: 1,
      })
    ).deep.equals({
      convert_me: 1,
      DONTCONVERTME: 1,
    })
  })
})
