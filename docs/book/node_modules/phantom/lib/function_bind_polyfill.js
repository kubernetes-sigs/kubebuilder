'use strict';

/* eslint-disable no-extend-native, consistent-this, require-jsdoc, no-empty-function, no-invalid-this */

/**
 * Ensure that Function has bind() method (PhantomJS version <= 1.9 support)
 * This is a Polyfill replacement from MDN.
 * @see {@link https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Function/bind#Polyfill}
 */
if (!Function.prototype.bind) {
    Function.prototype.bind = function (oThis) {
        if (typeof this !== 'function') {
            throw new TypeError('Function.prototype.bind - what is trying to be bound is not callable');
        }

        var self = this;
        var aArgs = Array.prototype.slice.call(arguments, 1);

        function NoopFunction() {}

        function boundFunction() {
            return self.apply(this instanceof NoopFunction ? this : oThis, aArgs.concat(Array.prototype.slice.call(arguments)));
        }

        if (this.prototype) {
            NoopFunction.prototype = this.prototype;
        }
        boundFunction.prototype = new NoopFunction();

        return boundFunction;
    };
}

/* eslint-enable no-extend-native, consistent-this, require-jsdoc, no-empty-function, no-invalid-this */