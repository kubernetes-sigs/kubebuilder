'use strict';

Object.defineProperty(exports, "__esModule", {
    value: true
});

var _createClass = function () { function defineProperties(target, props) { for (var i = 0; i < props.length; i++) { var descriptor = props[i]; descriptor.enumerable = descriptor.enumerable || false; descriptor.configurable = true; if ("value" in descriptor) descriptor.writable = true; Object.defineProperty(target, descriptor.key, descriptor); } } return function (Constructor, protoProps, staticProps) { if (protoProps) defineProperties(Constructor.prototype, protoProps); if (staticProps) defineProperties(Constructor, staticProps); return Constructor; }; }();

function _classCallCheck(instance, Constructor) { if (!(instance instanceof Constructor)) { throw new TypeError("Cannot call a class as a function"); } }

/**
 * Page class that proxies everything to phantomjs
 */
var Page = function () {
    function Page(phantom, pageId) {
        _classCallCheck(this, Page);

        this.target = 'page$' + pageId;
        this.phantom = phantom;
    }

    /**
     * Add an event listener to the page on phantom
     *
     * @param event The name of the event (Ej. onResourceLoaded)
     * @param [runOnPhantom=false] Indicate if the event must run on the phantom runtime or not
     * @param listener The event listener. When runOnPhantom=true, this listener code would be run on phantom, and thus,
     * all the closure info wont work
     * @returns {*}
     */


    _createClass(Page, [{
        key: 'on',
        value: function on(event, runOnPhantom, listener) {
            var mustRunOnPhantom = void 0;
            var callback = void 0;
            var args = void 0;

            if (typeof runOnPhantom === 'function') {
                args = [].slice.call(arguments, 2);
                mustRunOnPhantom = false;
                callback = runOnPhantom.bind(this);
            } else {
                args = [].slice.call(arguments, 3);
                mustRunOnPhantom = runOnPhantom;
                callback = mustRunOnPhantom ? listener : listener.bind(this);
            }

            return this.phantom.on(event, this.target, mustRunOnPhantom, callback, args);
        }

        /**
         * Removes an event listener
         *
         * @param event the event name
         * @returns {*}
         */

    }, {
        key: 'off',
        value: function off(event) {
            return this.phantom.off(event, this.target);
        }

        /**
         * Invokes an asynchronous method
         */

    }, {
        key: 'invokeAsyncMethod',
        value: function invokeAsyncMethod() {
            return this.phantom.execute(this.target, 'invokeAsyncMethod', [].slice.call(arguments));
        }

        /**
         * Invokes a method
         */

    }, {
        key: 'invokeMethod',
        value: function invokeMethod() {
            return this.phantom.execute(this.target, 'invokeMethod', [].slice.call(arguments));
        }

        /**
         * Defines a method
         */

    }, {
        key: 'defineMethod',
        value: function defineMethod(name, definition) {
            return this.phantom.execute(this.target, 'defineMethod', [name, definition]);
        }

        /**
         * Gets or sets a property
         */

    }, {
        key: 'property',
        value: function property() {
            return this.phantom.execute(this.target, 'property', [].slice.call(arguments));
        }

        /**
         * Gets or sets a setting
         */

    }, {
        key: 'setting',
        value: function setting() {
            return this.phantom.execute(this.target, 'setting', [].slice.call(arguments));
        }
    }]);

    return Page;
}();

exports.default = Page;


var asyncMethods = ['includeJs', 'open'];

var methods = ['addCookie', 'clearCookies', 'close', 'deleteCookie', 'evaluate', 'evaluateAsync', 'evaluateJavaScript', 'injectJs', 'openUrl', 'reload', 'render', 'renderBase64', 'sendEvent', 'setContent', 'setProxy', 'stop', 'switchToFrame', 'switchToMainFrame', 'goBack', 'uploadFile'];

asyncMethods.forEach(function (method) {
    Page.prototype[method] = function () {
        return this.invokeAsyncMethod.apply(this, [method].concat([].slice.call(arguments)));
    };
});

methods.forEach(function (method) {
    Page.prototype[method] = function () {
        return this.invokeMethod.apply(this, [method].concat([].slice.call(arguments)));
    };
});