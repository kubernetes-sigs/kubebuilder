var Q = require('..');
var assert = require('assert');

describe('#map', function() {
    it('should perform iterator with array', function(done) {
        Q([0, 1, 2]).map(function(num) {
            return num + 10;
        }).then(function(array) {
            assert.equal(array[0], 10);
            assert.equal(array[1], 11);
            assert.equal(array[2], 12);
        }).then(done, done)
    });

    it('should perform iterator with object', function(done) {
        Q({ one: 1, two: 2, three: 3 })
        .map(function(num, key) {
            assert(typeof key === 'string');
            return num + 10;
        }).then(function(array) {
            assert.equal(array[0], 11);
            assert.equal(array[1], 12);
            assert.equal(array[2], 13);
        }).then(done, done)
    });

    it('should allow promises as values', function(done) {
        Q([0, Q(1).delay(10), 2]).map(function(num, key) {
            return Q.delay(10).then(function() {
                return num + 10;
            });
        }).then(function(array) {
            assert.equal(array[0], 10);
            assert.equal(array[1], 11);
            assert.equal(array[2], 12);
        }).then(done, done)
    });
});
