var Transform = require('stream').Transform
var util = require('util')
var os = require('os')

function Liner(opts) {
  opts = opts || {}
  opts.objectMode = true
  Transform.call(this, opts)
}

util.inherits(Liner, Transform)
Liner.prototype._transform = function transform(chunk, encoding, done) {
  var data = chunk.toString()
  if (this._lastLineData) {
    data = this._lastLineData + data
  }
  var lines = data.split(os.EOL)
  this._lastLineData = lines.splice(lines.length - 1, 1)[0]

  lines.forEach(this.push.bind(this))
  done()
}

Liner.prototype._flush = function flush(done) {
  if (this._lastLineData) {
    this.push(this._lastLineData)
  }
  this._lastLineData = null
  done()
}

module.exports = Liner
