# linerstream

Split a readable stream by newline characters

[![NPM](https://nodei.co/npm/linerstream.png)](https://nodei.co/npm/linerstream/)

[![Build Status](https://travis-ci.org/nisaacson/linerstream.png)](https://travis-ci.org/nisaacson/linerstream)
[![Dependency Status](https://david-dm.org/nisaacson/linerstream/status.png)](https://david-dm.org/nisaacson/linerstream)
[![Code Climate](https://codeclimate.com/github/nisaacson/linerstream.png)](https://codeclimate.com/github/nisaacson/linerstream)

# Installation
```bash
npm install -S linerstream
```

# Usage

Create an instance of linestream and pipe a readable stream into that instance

```javascript
var Linerstream = require('linerstream')
// splitter is an instance of require('stream').Transform
var opts = {
  highWaterMark: 2
}
var splitter = new Linerstream(opts) // opts is optional

var readStream = fs.createReadStream('/file/with/line/breaks.txt')
var lineByLineStream = readStream.pipe(splitter)
lineByLineStream.on('data', function(chunk) {
  console.dir(chunk)  // no line breaks here :)
})
```



