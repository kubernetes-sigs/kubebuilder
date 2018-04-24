"use strict";

Object.defineProperty(exports, "__esModule", {
    value: true
});

var _crypto = require("crypto");

var _crypto2 = _interopRequireDefault(_crypto);

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { default: obj }; }

function _classCallCheck(instance, Constructor) { if (!(instance instanceof Constructor)) { throw new TypeError("Cannot call a class as a function"); } }

/**
 * A simple command class that gets deserialized when it is sent to phantom
 */
var Command = function Command(id, target, name) {
    var params = arguments.length <= 3 || arguments[3] === undefined ? [] : arguments[3];

    _classCallCheck(this, Command);

    this.id = id || _crypto2.default.randomBytes(16).toString('hex');
    this.target = target;
    this.name = name;
    this.params = params;
    this.deferred = undefined;
};

exports.default = Command;