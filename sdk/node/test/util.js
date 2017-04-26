/* eslint-env mocha */

const expect = require('chai').expect
const util = require('../dist/util')

describe('util package', () => {

  describe('sanitizeX509GuardData', () => {

    it('arrayifies attributes', () => {
      expect(
        util.sanitizeX509GuardData({
          subject: {
            C: 'foo',
            O: 'foo',
            OU: 'foo',
            L: 'foo',
            ST: 'foo',
            STREET: 'foo',
            POSTALCODE: 'foo',
            SERIALNUMBER: 'foo',
            CN: 'foo',
          }
        })
      ).deep.equals(
        {
          subject: {
            C: ['foo'],
            O: ['foo'],
            OU: ['foo'],
            L: ['foo'],
            ST: ['foo'],
            STREET: ['foo'],
            POSTALCODE: ['foo'],
            SERIALNUMBER: 'foo',
            CN: 'foo',
          }
        }
      )
    })

    describe('error cases', () => {

      it('throws an error if there are multiple top-level attributes', () => {
        expect(() => {
          util.sanitizeX509GuardData({
            subject: {},
            foobar: {},
          })
        }).to.throw(Error)
      })

      it('throws an error if there is a top-level field that is not "subject"', () => {
        expect(() => {
          util.sanitizeX509GuardData({
            foobar: {},
          })
        }).to.throw(Error)
      })

      it('throws an error if there are invalid subject attributes', () => {
        expect(() => {
          util.sanitizeX509GuardData({
            subject: {
              C: 'valid',
              Foo: 'invalid',
            },
          })
        }).to.throw(Error)
      })

      it('throws an error if there are invalid array attributes', () => {
        expect(() => {
          util.sanitizeX509GuardData({
            subject: {
              CN: ['invalid'],
            },
          })
        }).to.throw(Error)
      })

    })

  })

})
