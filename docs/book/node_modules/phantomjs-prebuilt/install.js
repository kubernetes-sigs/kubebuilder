// Copyright 2012 The Obvious Corporation.

/*
 * This simply fetches the right version of phantom for the current platform.
 */

'use strict'

var requestProgress = require('request-progress')
var progress = require('progress')
var extractZip = require('extract-zip')
var cp = require('child_process')
var fs = require('fs-extra')
var helper = require('./lib/phantomjs')
var kew = require('kew')
var path = require('path')
var request = require('request')
var url = require('url')
var util = require('./lib/util')
var which = require('which')
var os = require('os')

var originalPath = process.env.PATH

var checkPhantomjsVersion = util.checkPhantomjsVersion
var getTargetPlatform = util.getTargetPlatform
var getTargetArch = util.getTargetArch
var getDownloadSpec = util.getDownloadSpec
var findValidPhantomJsBinary = util.findValidPhantomJsBinary
var verifyChecksum = util.verifyChecksum
var writeLocationFile = util.writeLocationFile

// If the process exits without going through exit(), then we did not complete.
var validExit = false

process.on('exit', function () {
  if (!validExit) {
    console.log('Install exited unexpectedly')
    exit(1)
  }
})

// NPM adds bin directories to the path, which will cause `which` to find the
// bin for this package not the actual phantomjs bin.  Also help out people who
// put ./bin on their path
process.env.PATH = helper.cleanPath(originalPath)

var libPath = path.join(__dirname, 'lib')
var pkgPath = path.join(libPath, 'phantom')
var phantomPath = null

// If the user manually installed PhantomJS, we want
// to use the existing version.
//
// Do not re-use a manually-installed PhantomJS with
// a different version.
//
// Do not re-use an npm-installed PhantomJS, because
// that can lead to weird circular dependencies between
// local versions and global versions.
// https://github.com/Obvious/phantomjs/issues/85
// https://github.com/Medium/phantomjs/pull/184
kew.resolve(true)
  .then(tryPhantomjsInLib)
  .then(tryPhantomjsOnPath)
  .then(downloadPhantomjs)
  .then(extractDownload)
  .then(function (extractedPath) {
    return copyIntoPlace(extractedPath, pkgPath)
  })
  .then(function () {
    var location = getTargetPlatform() === 'win32' ?
        path.join(pkgPath, 'bin', 'phantomjs.exe') :
        path.join(pkgPath, 'bin' ,'phantomjs')

    try {
      // Ensure executable is executable by all users
      fs.chmodSync(location, '755')
    } catch (err) {
      if (err.code == 'ENOENT') {
        console.error('chmod failed: phantomjs was not successfully copied to', location)
        exit(1)
      }
      throw err
    }

    var relativeLocation = path.relative(libPath, location)
    writeLocationFile(relativeLocation)

    console.log('Done. Phantomjs binary available at', location)
    exit(0)
  })
  .fail(function (err) {
    console.error('Phantom installation failed', err, err.stack)
    exit(1)
  })

function exit(code) {
  validExit = true
  process.env.PATH = originalPath
  process.exit(code || 0)
}


function findSuitableTempDirectory() {
  var now = Date.now()
  var candidateTmpDirs = [
    process.env.npm_config_tmp,
    os.tmpdir(),
    path.join(process.cwd(), 'tmp')
  ]

  for (var i = 0; i < candidateTmpDirs.length; i++) {
    var candidatePath = candidateTmpDirs[i]
    if (!candidatePath) continue

    try {
      candidatePath = path.join(path.resolve(candidatePath), 'phantomjs')
      fs.mkdirsSync(candidatePath, '0777')
      // Make double sure we have 0777 permissions; some operating systems
      // default umask does not allow write by default.
      fs.chmodSync(candidatePath, '0777')
      var testFile = path.join(candidatePath, now + '.tmp')
      fs.writeFileSync(testFile, 'test')
      fs.unlinkSync(testFile)
      return candidatePath
    } catch (e) {
      console.log(candidatePath, 'is not writable:', e.message)
    }
  }

  console.error('Can not find a writable tmp directory, please report issue ' +
      'on https://github.com/Medium/phantomjs/issues with as much ' +
      'information as possible.')
  exit(1)
}


function getRequestOptions() {
  var strictSSL = !!process.env.npm_config_strict_ssl
  if (process.version == 'v0.10.34') {
    console.log('Node v0.10.34 detected, turning off strict ssl due to https://github.com/joyent/node/issues/8894')
    strictSSL = false
  }

  var options = {
    uri: getDownloadUrl(),
    encoding: null, // Get response as a buffer
    followRedirect: true, // The default download path redirects to a CDN URL.
    headers: {},
    strictSSL: strictSSL
  }

  var proxyUrl = process.env.npm_config_https_proxy ||
      process.env.npm_config_http_proxy ||
      process.env.npm_config_proxy
  if (proxyUrl) {

    // Print using proxy
    var proxy = url.parse(proxyUrl)
    if (proxy.auth) {
      // Mask password
      proxy.auth = proxy.auth.replace(/:.*$/, ':******')
    }
    console.log('Using proxy ' + url.format(proxy))

    // Enable proxy
    options.proxy = proxyUrl
  }

  // Use the user-agent string from the npm config
  options.headers['User-Agent'] = process.env.npm_config_user_agent

  // Use certificate authority settings from npm
  var ca = process.env.npm_config_ca
  if (!ca && process.env.npm_config_cafile) {
    try {
      ca = fs.readFileSync(process.env.npm_config_cafile, {encoding: 'utf8'})
        .split(/\n(?=-----BEGIN CERTIFICATE-----)/g)

      // Comments at the beginning of the file result in the first
      // item not containing a certificate - in this case the
      // download will fail
      if (ca.length > 0 && !/-----BEGIN CERTIFICATE-----/.test(ca[0])) {
        ca.shift()
      }

    } catch (e) {
      console.error('Could not read cafile', process.env.npm_config_cafile, e)
    }
  }

  if (ca) {
    console.log('Using npmconf ca')
    options.agentOptions = {
      ca: ca
    }
    options.ca = ca
  }

  return options
}

function handleRequestError(error) {
  if (error && error.stack && error.stack.indexOf('SELF_SIGNED_CERT_IN_CHAIN') != -1) {
      console.error('Error making request, SELF_SIGNED_CERT_IN_CHAIN. ' +
          'Please read https://github.com/Medium/phantomjs#i-am-behind-a-corporate-proxy-that-uses-self-signed-ssl-certificates-to-intercept-encrypted-traffic')
      exit(1)
  } else if (error) {
    console.error('Error making request.\n' + error.stack + '\n\n' +
        'Please report this full log at https://github.com/Medium/phantomjs')
    exit(1)
  } else {
    console.error('Something unexpected happened, please report this full ' +
        'log at https://github.com/Medium/phantomjs')
    exit(1)
  }
}

function requestBinary(requestOptions, filePath) {
  var deferred = kew.defer()

  var writePath = filePath + '-download-' + Date.now()

  console.log('Receiving...')
  var bar = null
  requestProgress(request(requestOptions, function (error, response, body) {
    console.log('')
    if (!error && response.statusCode === 200) {
      fs.writeFileSync(writePath, body)
      console.log('Received ' + Math.floor(body.length / 1024) + 'K total.')
      fs.renameSync(writePath, filePath)
      deferred.resolve(filePath)

    } else if (response) {
      console.error('Error requesting archive.\n' +
          'Status: ' + response.statusCode + '\n' +
          'Request options: ' + JSON.stringify(requestOptions, null, 2) + '\n' +
          'Response headers: ' + JSON.stringify(response.headers, null, 2) + '\n' +
          'Make sure your network and proxy settings are correct.\n\n' +
          'If you continue to have issues, please report this full log at ' +
          'https://github.com/Medium/phantomjs')
      exit(1)
    } else {
      handleRequestError(error)
    }
  })).on('progress', function (state) {
    try {
      if (!bar) {
        bar = new progress('  [:bar] :percent', {total: state.size.total, width: 40})
      }
      bar.curr = state.size.transferred
      bar.tick()
    } catch (e) {
      // It doesn't really matter if the progress bar doesn't update.
    }
  })
  .on('error', handleRequestError)

  return deferred.promise
}


function extractDownload(filePath) {
  var deferred = kew.defer()
  // extract to a unique directory in case multiple processes are
  // installing and extracting at once
  var extractedPath = filePath + '-extract-' + Date.now()
  var options = {cwd: extractedPath}

  fs.mkdirsSync(extractedPath, '0777')
  // Make double sure we have 0777 permissions; some operating systems
  // default umask does not allow write by default.
  fs.chmodSync(extractedPath, '0777')

  if (filePath.substr(-4) === '.zip') {
    console.log('Extracting zip contents')
    extractZip(path.resolve(filePath), {dir: extractedPath}, function(err) {
      if (err) {
        console.error('Error extracting zip')
        deferred.reject(err)
      } else {
        deferred.resolve(extractedPath)
      }
    })

  } else {
    console.log('Extracting tar contents (via spawned process)')
    cp.execFile('tar', ['jxf', path.resolve(filePath)], options, function (err) {
      if (err) {
        console.error('Error extracting archive')
        deferred.reject(err)
      } else {
        deferred.resolve(extractedPath)
      }
    })
  }
  return deferred.promise
}


function copyIntoPlace(extractedPath, targetPath) {
  console.log('Removing', targetPath)
  return kew.nfcall(fs.remove, targetPath).then(function () {
    // Look for the extracted directory, so we can rename it.
    var files = fs.readdirSync(extractedPath)
    for (var i = 0; i < files.length; i++) {
      var file = path.join(extractedPath, files[i])
      if (fs.statSync(file).isDirectory() && file.indexOf(helper.version) != -1) {
        console.log('Copying extracted folder', file, '->', targetPath)
        return kew.nfcall(fs.move, file, targetPath)
      }
    }

    console.log('Could not find extracted file', files)
    throw new Error('Could not find extracted file')
  })
}

/**
 * Check to see if the binary in lib is OK to use. If successful, exit the process.
 */
function tryPhantomjsInLib() {
  return kew.fcall(function () {
    return findValidPhantomJsBinary(path.resolve(__dirname, './lib/location.js'))
  }).then(function (binaryLocation) {
    if (binaryLocation) {
      console.log('PhantomJS is previously installed at', binaryLocation)
      exit(0)
    }
  }).fail(function () {
    // silently swallow any errors
  })
}

/**
 * Check to see if the binary on PATH is OK to use. If successful, exit the process.
 */
function tryPhantomjsOnPath() {
  if (getTargetPlatform() != process.platform || getTargetArch() != process.arch) {
    console.log('Building for target platform ' + getTargetPlatform() + '/' + getTargetArch() +
                '. Skipping PATH search')
    return kew.resolve(false)
  }

  return kew.nfcall(which, 'phantomjs')
  .then(function (result) {
    phantomPath = result
    console.log('Considering PhantomJS found at', phantomPath)

    // Horrible hack to avoid problems during global install. We check to see if
    // the file `which` found is our own bin script.
    if (phantomPath.indexOf(path.join('npm', 'phantomjs')) !== -1) {
      console.log('Looks like an `npm install -g` on windows; skipping installed version.')
      return
    }

    var contents = fs.readFileSync(phantomPath, 'utf8')
    if (/NPM_INSTALL_MARKER/.test(contents)) {
      console.log('Looks like an `npm install -g`')

      var phantomLibPath = path.resolve(fs.realpathSync(phantomPath), '../../lib/location')
      return findValidPhantomJsBinary(phantomLibPath)
      .then(function (binaryLocation) {
        if (binaryLocation) {
          writeLocationFile(binaryLocation)
          console.log('PhantomJS linked at', phantomLibPath)
          exit(0)
        }
        console.log('Could not link global install, skipping...')
      })
    } else {
      return checkPhantomjsVersion(phantomPath).then(function (matches) {
        if (matches) {
          writeLocationFile(phantomPath)
          console.log('PhantomJS is already installed on PATH at', phantomPath)
          exit(0)
        }
      })
    }
  }, function () {
    console.log('PhantomJS not found on PATH')
  })
  .fail(function (err) {
    console.error('Error checking path, continuing', err)
    return false
  })
}

/**
 * @return {?string} Get the download URL for phantomjs.
 *     May return null if no download url exists.
 */
function getDownloadUrl() {
  var spec = getDownloadSpec()
  return spec && spec.url
}

/**
 * Download phantomjs, reusing the existing copy on disk if available.
 * Exits immediately if there is no binary to download.
 * @return {Promise.<string>} The path to the downloaded file.
 */
function downloadPhantomjs() {
  var downloadSpec = getDownloadSpec()
  if (!downloadSpec) {
    console.error(
        'Unexpected platform or architecture: ' + getTargetPlatform() + '/' + getTargetArch() + '\n' +
        'It seems there is no binary available for your platform/architecture\n' +
        'Try to install PhantomJS globally')
    exit(1)
  }

  var downloadUrl = downloadSpec.url
  var downloadedFile

  return kew.fcall(function () {
    // Can't use a global version so start a download.
    var tmpPath = findSuitableTempDirectory()
    var fileName = downloadUrl.split('/').pop()
    downloadedFile = path.join(tmpPath, fileName)

    if (fs.existsSync(downloadedFile)) {
      console.log('Download already available at', downloadedFile)
      return verifyChecksum(downloadedFile, downloadSpec.checksum)
    }
    return false
  }).then(function (verified) {
    if (verified) {
      return downloadedFile
    }

    // Start the install.
    console.log('Downloading', downloadUrl)
    console.log('Saving to', downloadedFile)
    return requestBinary(getRequestOptions(), downloadedFile)
  })
}
