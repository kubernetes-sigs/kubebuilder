# search

This plugin is a default plugin for GitBook, it adds an interactive search bar to your book.

This plugin is backend agnostic.

### Disable this plugin

This is a default plugin and it can be disabled using a `book.json` configuration:

```
{
    plugins: ["-search"]
}
```

### Backends

| Backend | Plugin Name | Description |
| ------- | ----------- | ----------- |
| [Lunr](https://github.com/GitbookIO/plugin-lunr) | `lunr` | Index the content into a local/offlien index |
| [Algolia](https://github.com/GitbookIO/plugin-algolia) | `algolia` | Index the content in Algolia |

