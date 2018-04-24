# gitbook-plugin-sequence-diagrams

[![NPM](https://nodei.co/npm/gitbook-plugin-sequence-diagrams.png)](https://nodei.co/npm/gitbook-plugin-sequence-diagrams/)

[js-sequence-diagrams](https://github.com/bramp/js-sequence-diagrams) plugin for [GitBook](https://github.com/GitbookIO/gitbook)

## Installation

    $ npm install gitbook-plugin-sequence-diagrams

book.json add the plugin

```
{
  "plugins": ["sequence-diagrams"]
}
```

## Configuration

book.json add the js-sequence-diagrams options

```
"pluginsConfig": {
  "sequence-diagrams": {
    "theme": "simple"
  }
}
```

## Usage

put in your book block as

```
{% sequence %}
Alice->Bob: Hello Bob, how are you?
Note right of Bob: Bob thinks
Bob-->Alice: I am good thanks!
{% endsequence %}
```

### Extend the width

```
{% sequence width=770 %}
```
