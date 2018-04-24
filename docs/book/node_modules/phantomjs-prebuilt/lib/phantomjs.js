// Copyright 2013 The Obvious Corporation.

/**
 * @fileoverview Helpers made available via require('phantomjs') once package is
 * installed.
 */

var fs = require('fs')
var path = require('path')
var spawn = require('child_process').spawn
var Promise = require('es6-promise').Promise


/**
 * Where the phantom binary can be found.
 * @type {string}
 */
try {
  var location = require('./location')
  exports.path = path.resolve(__dirname, location.location)
  exports.platform = location.platform
  exports.arch = location.arch
} catch(e) {
  // Must be running inside install script.
  exports.path = null
}


/**
 * The version of phantomjs installed by this package.
 * @type {number}
 */
exports.version = '2.1.1'


/**
 * Returns a clean path that helps avoid `which` finding bin files installed
 * by NPM for this repo.
 * @param {string} path
 * @return {string}
 */
exports.cleanPath = function (path) {
  return path
      .replace(/:[^:]*node_modules[^:]*/g, '')
      .replace(/(^|:)\.\/bin(\:|$)/g, ':')
      .replace(/^:+/, '')
      .replace(/:+$/, '')
}


// Make sure the binary is executable.  For some reason doing this inside
// install does not work correctly, likely due to some NPM step.
if (exports.path) {
  try {
    // avoid touching the binary if it's already got the correct permissions
    var st = fs.statSync(exports.path)
    var mode = st.mode | parseInt('0555', 8)
    if (mode !== st.mode) {
      fs.chmodSync(exports.path, mode)
    }
  } catch (e) {
    // Just ignore error if we don't have permission.
    // We did our best. Likely because phantomjs was already installed.
  }
}

/**
 * Executes a script or just runs PhantomJS
 */
exports.exec = function () {
  var args = Array.prototype.slice.call(arguments)
  return spawn(exports.path, args)
}

/**
 * Runs PhantomJS with provided options
 * @example
 * // handy with WebDriver
 * phantomjs.run('--webdriver=4444').then(program => {
 *   // do something
 *   program.kill()
 * })
 * @returns {Promise} the process of PhantomJS
 */
exports.run = function () {
  var args = arguments
  return new Promise(function (resolve, reject) {
    try {
      var program = exports.exec.apply(null, args)
      var isFirst = true
      var stderr = ''
      program.stdout.on('data', function () {
        // This detects PhantomJS instance get ready.
        if (!isFirst) return
        isFirst = false
        resolve(program)
      })
      program.stderr.on('data', function (data) {
        stderr = stderr + data.toString('utf8')
      })
      program.on('error', function (err) {
        if (!isFirst) return
        isFirst = false
        reject(err)
      })
      program.on('exit', function (code) {
        if (!isFirst) return
        isFirst = false
        if (code == 0) {
          // PhantomJS doesn't use exit codes correctly :(
          if (stderr.indexOf('Error:') == 0) {
            reject(new Error(stderr))
          } else {
            resolve(program)
          }
        } else {
          reject(new Error('Exit code: ' + code))
        }
      })
    } catch (err) {
      reject(err)
    }
  })
}
