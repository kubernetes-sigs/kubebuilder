'use strict';

Object.defineProperty(exports, "__esModule", {
    value: true
});

var _createClass = function () { function defineProperties(target, props) { for (var i = 0; i < props.length; i++) { var descriptor = props[i]; descriptor.enumerable = descriptor.enumerable || false; descriptor.configurable = true; if ("value" in descriptor) descriptor.writable = true; Object.defineProperty(target, descriptor.key, descriptor); } } return function (Constructor, protoProps, staticProps) { if (protoProps) defineProperties(Constructor.prototype, protoProps); if (staticProps) defineProperties(Constructor, staticProps); return Constructor; }; }();

var _crypto = require('crypto');

var _crypto2 = _interopRequireDefault(_crypto);

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { default: obj }; }

function _classCallCheck(instance, Constructor) { if (!(instance instanceof Constructor)) { throw new TypeError("Cannot call a class as a function"); } }

var OutObject = function () {
    function OutObject(phantom) {
        _classCallCheck(this, OutObject);

        this._phantom = phantom;
        this.target = 'OutObject$' + _crypto2.default.randomBytes(16).toString('hex');
    }

    _createClass(OutObject, [{
        key: 'property',
        value: function property(name) {
            return this._phantom.execute(this.target, 'property', [name]);
        }
    }]);

    return OutObject;
}();

exports.default = OutObject;