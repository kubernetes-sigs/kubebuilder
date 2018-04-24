var sinon = require('sinon')
var fs = require('fs')
var path = require('path')
var os = require('os')
var expect = require('chai').expect
var Linerstream = require('../')

describe('Split test', function() {
  describe('given text with line breaks', function() {
    it('should split on new lines', function(done) {
      var fixturePath = path.join(__dirname, 'data/newlines-big.txt')
      var inputStream = fs.createReadStream(fixturePath)
      var splitter = new Linerstream()
      expect(splitter).to.exist
      var output = inputStream.pipe(splitter)
      var validateLineSpy = sinon.spy(validateLine)

      output.on('finish', finishHandler)
      output.on('readable', readableHandler)

      function validateLine(line) {
        expect(line).to.exist
        expect(line).to.be.a('string')
        expect(line).to.not.be.empty
        expect(line).to.not.match(/\n|\r/)
      }

      function readableHandler() {
        var data
        while (true) {
          data = output.read()
          if (!data) {
            break
          }
          validateLineSpy(data)

        }
      }

      function finishHandler() {
        expect(validateLineSpy.callCount).to.be.above(1)
        done()
      }

    })
  })
})
