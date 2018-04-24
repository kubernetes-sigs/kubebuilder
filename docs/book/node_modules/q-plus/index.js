/*!
 * Q library (c) 2009-2012 Kris Kowal under the terms of the MIT
 * license found at http://github.com/kriskowal/q/raw/master/LICENSE
 * Inspired by async (https://github.com/caolan/async)
 */

(function (definition) {
    // CommonJS
    if (typeof exports === "object" && typeof module === "object") {
        module.exports = definition(require('q'));
    // RequireJS
    } else if (typeof define === "function" && define.amd) {
        define(definition);
    // <script>
    } else if (typeof self.Q !== "undefined") {
        self.Q = definition(self.Q);
    } else {
        throw new Error("Environment not recognized.");
    }

})(function(Q) {

var setMethods = function(q) {
    var fn = q.makePromise.prototype;

    fn.each = each;
    fn.map = map;
    fn.eachSeries = eachSeries;
    fn.forEach = eachSeries;
    fn.mapSeries = mapSeries;
    fn.while = whileFn;
    fn.until = untilFn;
    fn.times = times;
    fn.timesSeries = timesSeries;

    q.while = function(test, fn) {
        return this().while(test, fn);
    };
    q.until = function(test, fn) {
        return this().until(test, fn);
    };
    q.times = function(n, fn) {
        return this().times(n, fn);
    };
    q.timesSeries = function(n, fn) {
        return this().timesSeries(n, fn);
    };
};

setMethods(Q);
var newQ = Q;

/*
// Goal: Q+ = Q, Q+(Q) = Q
// Problem: newQ doesn't carry over Q methods, so Q.delay wouldn't work
var newQ = function(x) {
    // Use provided Q
    if (x && x.defer && x.reject) {
        Q = x;
        x = 'f3b642dc';
    }
    // Set methods id it hasn't been done
    if (!Q.makePromise.prototype.eachSeries) {
        setMethods(Q);
    }
    // Return the Q library or a promise
    if (x === 'f3b642dc') return Q;
    return Q(x);
};
*/

// Reduce function that accepts Arrays & Plain Objects
function reduce(arr, fn, accu) {
    if (typeof arr === 'object' && !Array.isArray(arr)) {
        for (var key in arr) {
            accu = fn(accu, arr[key], key);
        }
    } else {
        for(var i = 0; i < arr.length; i++) {
            accu = fn(accu, arr[i]);
        }
    }
    return accu;
}

/**
 * Executes function for each element of an array or object,
 * running all functions or promises in parallel.
 * @promise {(array|object)}
 * @param   {function} fn : The function called per iteration
 *                     @param value : Value of element
 *                     @param key|index : Key if object, index if array
 * @returns {object} Original object
 */
function each(fn) {
    var i = 0;
    return this.then(function(object) {
        return Q.all(reduce(object, function(array, value, key) {
            var fnv = Q.isPromise(value) ? value.then(function(v) {
                return fn(v, key || i++);
            }) : fn(value, key || i++);

            return array.concat(fnv);
        }, []))
        .thenResolve(object);
    });
};

/**
 * Transforms an array or object into a new array using the iterator
 * function, running all functions or promises in parallel.
 * @promise {(array|object)}
 * @param   {function} fn : The function called per iteration
 *                     @param value : Value of element
 *                     @param [key] : Key of element
 * @returns {array} Transformed array
 */
function map(fn) {
    // Allow promise as iterator
    //fn = Q.promised(fn);

    return this.then(function(object) {
        return Q.all(reduce(object, function(array, value, key) {
            var fnv = Q.isPromise(value) ? value.then(function(v) {
                return fn(v, key);
            }) : fn(value, key);

            return array.concat(fnv);
        }, []));
    });
};

/**
 * Executes function for each element of an array or object, running
 * any promises in series only after the last has been completed.
 * @promise {(array|object)}
 * @param   {function} fn : The function called per iteration
 *                     @param value : Value of element
 *                     @param key|index : Key if object, index if array
 * @returns {object} Original object
 */
function eachSeries(fn) {
    var i = 0;
    return this.then(function(object) {
        return reduce(object, function(newPromise, value, key) {
            // Allow value to be a promise
            if (Q.isPromise(value)) return newPromise.then(function() {
                return value;
            }).then(function(v) {
                return fn(v, key || i++);
            });

            return newPromise.then(function() {
                return fn(value, key || i++);;
            });
        }, Q())
        .thenResolve(object)
    });
}

/**
 * Transforms an array or object into a new array using the iterator function,
 * running any promises in series only after the last has been completed.
 * @promise {(array|object)}
 * @param   {function} fn : The function called per iteration
 *                     @param value : Value of element
 *                     @param [key] : Key of element
 * @returns {array} Transformed array
 */
function mapSeries(fn) {
    var newArray = [];
    // Allow iterator return to be a promise
    function push(value, key) {
        value = fn(value, key);
        if (Q.isPromise(value)) return value.then(function(v) {
            newArray.push(v);
        });
        newArray.push(value);
    }

    return this.then(function(object) {
        return reduce(object, function(newPromise, value, key) {
            // Allow value to be a promise
            if (Q.isPromise(value)) return newPromise.then(function() {
                return value;
            }).then(function(v) {
                return push(v, key);
            });

            return newPromise.then(function() {
                return push(value, key)
            });
        }, Q());
    }).thenResolve(newArray);
};

/**
 * Repeatedly call a function while a test function returns true.
 * @promise {*}        last
 * @param   {function} test : synchronous truth test
 *                     @param value : Value from last iteration
 * @param   {function} fn : The function called per iteration
 *                     @param value : Value from last iteration
 * @returns {*} Value from last iteration
 */
function whileFn(test, fn) {
    return this.then(function(last) {
        if (test(last)) {
            return Q(fn(last)).then(function(value) {
                return Q(value).while(test, fn);
            });
        }
        return last;
    });
};

/**
 * Repeatedly call a function while a test function returns false.
 * Same options as #while
 */
function untilFn(test, fn) {
    return this.then(function(last) {
        if (!test(last)) {
            return Q(fn(last)).then(function(value) {
                return Q(value).until(test, fn);
            });
        }
        return last;
    });
};

/**
 * Calls the callback function n times, and accumulates results in the same manner you would use with map.
 * @promise {*}        last
 * @param   {number}   n : How many times to iterate
 * @param   {function} fn : The function called per iteration
 *                     @param value : Last return value
 * @returns {array} New array
 */
function times(n, fn) {
    var counter = [];
    return this.then(function(last) {
        for (var i = 0; i < n; i++) {
            counter.push(last || i);
        }
        return Q(counter).map(fn);
    });
};

/**
 * Calls the callback function n times, and accumulates results in the same manner you would use with map.
 * Same as #times
 */
function timesSeries(n, fn) {
    var counter = [];
    return this.then(function(last) {
        for (var i = 0; i < n; i++) {
            counter.push(last || i);
        }
        return Q(counter).mapSeries(fn);
    });
};

return newQ;

});
