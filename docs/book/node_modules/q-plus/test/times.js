var Q = require('..');
var assert = require('assert');

describe('#times', function() {
    it('should iterate n times', function(done) {
        Q.times(5, function(i) {
            return i;
        }).then(function(arr) {
            assert.equal(arr[0], 0);
            assert.equal(arr.length, 5);
        }).then(done, done);
    });

    it('should work in a chain', function(done) {
        Q(2).times(5, function(num) {
            return num;
        }).then(function(arr) {
            assert.equal(arr[0], 2);
            assert.equal(arr[4], 2);
            assert.equal(arr.length, 5);
        }).then(done, done);
    });
});
