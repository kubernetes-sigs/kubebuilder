# Q+

**Q+** is a utility add-on for [the promise library Q](https://github.com/kriskowal/q). It adds flow-control methods to work with data between promises.

## Examples

```js
var Q = require('q-plus');

Q(['1145d024','4b4897c2','c89a11ec'])
.mapSeries(function(id) {
    return Animal.duplicate(id);
}).then(function(animals) {
    console.log(animals); //= Array of duplicate animals
})

var africanMammalLocations = [];
Animal.where({ type: 'Mammal' })
.map(function(animal) {
    return animal.getHabitat();
})
.eachSeries(function(habitat) {
    if (habitat.continent == 'Africa')
        africanMammalLocations.push(habitat.name);
}).thenResolve(africanMammalLocations);
```

---------------------------------------
## Documentation

<a name="eachSeries" />
### Q(object).eachSeries(iterator)

```js
// Typical 'forEach' usage:
Q([1, 2, 3, 4]).eachSeries(function(num, i) {
    if (num * 3 < 10) storage.push(num);
});
// With a promise as iterator:
Q([{ name: 'Mark' }, { name: 'Sarah' }])
.eachSeries(function(person, i) {
    // use a return statement if using a promise:
    return People.new(person.name); 
});
// Using an object instead of an array:
Q({ one: 1, two: 2, three: 3 }).eachSeries(function(num, key) {
    console.log(key, num); //= one 1, two 2, three 3
});
```

---------------------------------------
<a name="mapSeries" />
### Q(object).mapSeries(iterator)

*TODO*

---------------------------------------
<a name="map" />
### Q(object).each(iterator)

*TODO*

---------------------------------------
<a name="map" />
### Q(object).map(iterator)

*TODO*

---------------------------------------
<a name="while" />
### Q.while(test, iterator)
### Q(value).while(test, iterator)

Repeatedly calls `iterator` until `test` returns false;

* **test**(`value`) `function` : synchronous truth test to perform before each execution of iterator. `value` is the return value of the last iterator.
* **iterator**(`value`) `function` : A function or promise which is called each time **test** passes. `value` is the return value of the last iterator.

```js
var count = 0;
var arr = [];

// Iterator can be a normal function or a promise
Q.while(function() {
    return count < 10;
}, function() {
    count++;
    arr.push(true);
}).then(function() {
    console.log(arr.length); //= 10
});

// Can be used in a promise chain, passing the return
// value of the last function/iterator to the next
Q(2).while(function(total) {
    return total <= 512;
}, function(total) {
    return Q.delay(2).then(function() {
        return total * 2;
    });
}).then(function(finalTotal) {
    console.log(finalTotal); //= 1024 (2^10)
});
```

---------------------------------------
<a name="until" />
### Q.until(test, iterator)
### Q(value).until(test, iterator)

The inverse of **while** -- repeatedly calls `iterator` until `test` returns true.

---------------------------------------
<a name="times" />
### Q.times(n, iterator)
### Q(value).times(n, iterator)

Calls the `iterator` n times, and accumulates results the same way you'd use  **map**.

```js
var getPage = function(page) {
    return Results.limit(10).offset(page * 10);
};

Q.times(10, getPage).then(function(pages) {
    console.log(pages[0]); //= [res1, ..., res10]
});

Q(true).times(3, function(isCool) {
    if (isCool) return "cool";
    return "not cool";
}).then(function(arr) {
    console.log(arr); //= ['cool','cool','cool']
});
```
