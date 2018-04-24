"use strict";

var _typeof = typeof Symbol === "function" && typeof Symbol.iterator === "symbol" ? function (obj) { return typeof obj; } : function (obj) { return obj && typeof Symbol === "function" && obj.constructor === Symbol ? "symbol" : typeof obj; };

var _webpage = require("webpage");

var _webpage2 = _interopRequireDefault(_webpage);

var _system = require("system");

var _system2 = _interopRequireDefault(_system);

require("./function_bind_polyfill.js");

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { default: obj }; }

/**
 * Stores all all pages and single instance of phantom
 */
var objectSpace = {
    phantom: phantom
};

var events = {};

/**
 * All commands that have a custom implementation
 */
var commands = {
    createPage: function createPage(command) {
        var page = _webpage2.default.create();
        objectSpace['page$' + command.id] = page;

        page.onClosing = function () {
            return delete objectSpace['page$' + command.id];
        };

        command.response = { pageId: command.id };
        completeCommand(command);
    },
    property: function property(command) {
        if (command.params.length > 1) {
            if (typeof command.params[1] === 'function') {
                (function () {
                    // If the second parameter is a function then we want to proxy and pass parameters too
                    var callback = command.params[1];
                    var args = command.params.slice(2);
                    syncOutObjects(args);
                    objectSpace[command.target][command.params[0]] = function () {
                        var params = [].slice.call(arguments).concat(args);
                        return callback.apply(objectSpace[command.target], params);
                    };
                })();
            } else {
                // If the second parameter is not a function then just assign
                objectSpace[command.target][command.params[0]] = command.params[1];
            }
        } else {
            command.response = objectSpace[command.target][command.params[0]];
        }

        completeCommand(command);
    },
    setting: function setting(command) {
        if (command.params.length === 2) {
            objectSpace[command.target].settings[command.params[0]] = command.params[1];
        } else {
            command.response = objectSpace[command.target].settings[command.params[0]];
        }

        completeCommand(command);
    },

    windowProperty: function windowProperty(command) {
        if (command.params.length === 2) {
            window[command.params[0]] = command.params[1];
        } else {
            command.response = window[command.params[0]];
        }
        completeCommand(command);
    },

    addEvent: function addEvent(command) {
        var type = getTargetType(command.target);

        if (isEventSupported(type, command.params[0].type)) {
            var listeners = getEventListeners(command.target, command.params[0].type);

            if (typeof command.params[0].event === 'function') {
                listeners.otherListeners.push(function () {
                    var params = [].slice.call(arguments).concat(command.params[0].args);
                    return command.params[0].event.apply(objectSpace[command.target], params);
                });
            }
        }

        completeCommand(command);
    },

    removeEvent: function removeEvent(command) {
        var type = getTargetType(command.target);

        if (isEventSupported(type, command.params[0].type)) {
            events[command.target][command.params[0].type] = null;
            objectSpace[command.target][command.params[0].type] = null;
        }

        completeCommand(command);
    },

    noop: function noop(command) {
        return completeCommand(command);
    },

    invokeAsyncMethod: function invokeAsyncMethod(command) {
        var target = objectSpace[command.target];
        target[command.params[0]].apply(target, command.params.slice(1).concat(function (result) {
            command.response = result;
            completeCommand(command);
        }));
    },

    invokeMethod: function invokeMethod(command) {
        var target = objectSpace[command.target];
        var method = target[command.params[0]];
        command.response = method.apply(target, command.params.slice(1));
        completeCommand(command);
    },

    defineMethod: function defineMethod(command) {
        var target = objectSpace[command.target];
        target[command.params[0]] = command.params[1];
        completeCommand(command);
    }
};

/**
 * Calls readLine() and blocks until a message is ready
 */
function read() {
    var line = _system2.default.stdin.readLine();
    if (line) {
        var command = JSON.parse(line, function (key, value) {
            if (value && typeof value === 'string' && value.substr(0, 8) === 'function' && value.indexOf('[native code]') === -1) {
                var startBody = value.indexOf('{') + 1;
                var endBody = value.lastIndexOf('}');
                var startArgs = value.indexOf('(') + 1;
                var endArgs = value.indexOf(')');
                return new Function(value.substring(startArgs, endArgs), value.substring(startBody, endBody));
            }
            return value;
        });

        // Call here to look for transform key
        transform(command.params);

        try {
            executeCommand(command);
        } catch (e) {
            command.error = e.message;
            completeCommand(command);
        }
    }
}

/**
 * Looks for transform key and uses objectSpace to call objects
 * @param object
 */
function transform(object) {
    for (var key in object) {
        if (object.hasOwnProperty(key)) {
            var child = object[key];
            if (child === null || child === undefined) {
                return;
            } else if (child.transform === true) {
                object[key] = objectSpace[child.parent][child.method](child.target);
            } else if ((typeof child === "undefined" ? "undefined" : _typeof(child)) === 'object') {
                transform(child);
            }
        }
    }
}

/**
 * Sync all OutObjects present in the array
 *
 * @param objects
 */
function syncOutObjects(objects) {
    objects.forEach(function (param) {
        if (param.target !== undefined) {
            objectSpace[param.target] = param;
        }
    });
}

/**
 * Executes a command.
 * @param command the command to execute
 */
function executeCommand(command) {
    if (commands[command.name]) {
        return commands[command.name](command);
    }
    throw new Error("'" + command.name + "' isn't a command.");
}

/**
 * Verifies if an event is supported for a type of target
 *
 * @param type
 * @param eventName
 * @returns {boolean}
 */
function isEventSupported(type, eventName) {
    return type === 'page' && eventName.indexOf('on') === 0;
}

/**
 * Gets an object containing all the listeners for an event of a target
 *
 * @param target the target id
 * @param eventName the event name
 */
function getEventListeners(target, eventName) {
    if (!events[target]) {
        events[target] = {};
    }

    if (!events[target][eventName]) {
        events[target][eventName] = {
            outsideListener: getOutsideListener(eventName, target),
            otherListeners: []
        };

        objectSpace[target][eventName] = triggerEvent.bind(null, target, eventName);
    }

    return events[target][eventName];
}

/**
 * Determines a targets type using its id
 *
 * @param target
 * @returns {*}
 */
function getTargetType(target) {
    return target.toString().split('$')[0];
}

/**
 * Executes all the listeners for an event from a target
 *
 * @param target
 * @param eventName
 */
function triggerEvent(target, eventName) {
    var args = [].slice.call(arguments, 2);
    var listeners = events[target][eventName];
    listeners.outsideListener.apply(null, args);
    listeners.otherListeners.forEach(function (listener) {
        listener.apply(objectSpace[target], args);
    });
}

/**
 * Returns a function that will notify to node that an event have been triggered
 *
 * @param eventName
 * @param targetId
 * @returns {Function}
 */
function getOutsideListener(eventName, targetId) {
    return function () {
        var args = [].slice.call(arguments, 0);
        _system2.default.stdout.writeLine('<event>' + JSON.stringify({ target: targetId, type: eventName, args: args }));
    };
}

/**
 * Completes a command by return a response to node and listening again for next command.
 * @param command
 */
function completeCommand(command) {
    _system2.default.stdout.writeLine('>' + JSON.stringify(command));
    // Prevent event-queue from clogging up by reads that block.
    setTimeout(read, 0);
}

read();