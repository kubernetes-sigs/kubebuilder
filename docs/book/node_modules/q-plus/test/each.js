var Q = require('..');
var assert = require('assert');

describe('#each', function() {
    it('should perform iterator with array', function(done) {
        var check = [false, false, false];
        Q().then(function() { return [4, 5, 6]; })
        .each(function(num, i) {
            check[i] = true;
        }).then(function(array) {
            assert.equal(array[0], 4);
            assert(check[0]); assert(check[1]); assert(check[2]);
        }).then(done, done)
    });

    it('should perform iterator with object', function(done) {
        var check = { one: false, two: false, three: false };
        Q({ one: 1, two: 2, three: 3 })
        .each(function(num, key) {
            assert(typeof key === 'string');
            check[key] = true;
        }).then(function(array) {
            assert.equal(array.two, 2);
            assert(check.one); assert(check.two); assert(check.three);
        }).then(done, done)
    });

    it('should allow promises as values', function(done) {
        var check = [false, false, false];
        Q([Q(0).delay(15), Q(1).delay(10), 2]).each(function(num, i) {
            return Q().delay(10).then(function() {
                check[i] = true;
            });
        }).then(function(array) {
            assert.equal(array[1], 1);
            assert(check[0]); assert(check[1]); assert(check[2]);
        }).then(done, done)
    });
});
