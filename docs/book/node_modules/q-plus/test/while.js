var Q = require('..');
var assert = require('assert');

describe('#while', function() {
    it('should iterate while test is true', function(done) {
        var count = 0;
        var arr = [];
        Q.while(function() {
            return count < 10;
        }, function() {
            count++;
            arr.push(1);
        }).then(function() {
            assert.equal(count, 10);
            assert.equal(arr.length, 10);
        }).then(done, done);
    });

    it('should iterate with promise', function(done) {
        var count = 0;
        Q.while(function() {
            return count < 10;
        }, function() {
            count++;
            return Q.delay(5);
        }).then(function() {
            assert.equal(count, 10);
        }).then(done, done);
    });

    it('should work in chain with return value', function(done) {
        Q(2).while(function(total) {
            return total < 1024;
        }, function(total) {
            return Q.delay(2).then(function() {
                return total * 2;
            });
        }).then(function(finalTotal) {
            assert.equal(finalTotal, 1024);
        }).then(done, done);
    });
});
