"use strict";

var _phantom = require("./phantom");

var _phantom2 = _interopRequireDefault(_phantom);

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { default: obj }; }

/**
 * Retuns a Promise of a new Phantom class instance
 * @param args command args to pass to phantom process
 * @param [config] configuration object
 * @param [config.phantomPath] path to phantomjs executable
 * @param [config.logger] object containing functions used for logging
 * @param [config.logLevel] log level to apply on the logger (if unset or default)
 * @returns {Promise}
 */
module.exports.create = function (args, config) {
  return new Promise(function (resolve) {
    return resolve(new _phantom2.default(args, config));
  });
};