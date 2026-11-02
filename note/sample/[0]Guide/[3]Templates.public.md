# Templates and assets

The site has three page templates and two optional error pages. Their file names are configured in `site.toml`; nothing is hard-coded to `.html`.

| Template | Used for |
| --- | --- |
| `index_template` | the root page `/` |
| `category_template` | every category directory |
| `content_template` | every note |
| `404` / `500` | missing pages and render errors |

## Page data

A page template receives `PageData`:

- `.Content` — translated HTML for the current note or category index
- `.Name`, `.Uri`, `.Parent`, `.Children`, `.IsAncestor` — navigation data
- `.Root`, `.Ancestors`, `.Prev`, `.Next`, `.HasChildren` — navigation helpers
- `.CurrentYear`, `.Engine` — year and `notesite/<version>`

The full field and method list is in [Template API](/reference/api).

## SSI includes

Templates can include shared fragments before parsing:

```html
<!--#include file="partials/head.html" -->
```

- Paths are relative to the file containing the directive.
- Includes are expanded recursively and may contain Go template actions.
- A fragment can be included only once per page; a second include of the same file is dropped.
- Includes are expanded before the template is parsed, so the included text shares the page's data.

This sample uses `partials/head.html`, `partials/header.html`, and `partials/footer.html`. The error pages include only the footer because they receive no page data.

## Static assets

`static_dirs` lists directories under `template_root` whose files are served as-is. The sample uses `static_dirs = ["assets"]`; each listed directory is flattened to the URL root:

```text
template/sample/assets/site.css  ->  /site.css
template/sample/assets/site.js   ->  /site.js
```

Link files directly; do not link a directory such as `/assets/`. Assets are read from disk on each request, so CSS and JavaScript changes are picked up without a restart.

## Reload behavior

- Template files are parsed at startup and re-parsed when they change.
- Notes are read per request and cached by modification time and size.
- `.cat` and `.res` changes trigger a route-tree rebuild.
- `site.toml` is loaded once; restart the server after changing it.

A broken template aborts the reload and a running server keeps its previous templates. At startup it prevents the server from starting.

## H1 convention

This sample puts the page H1 in the note content, including the category index notes, and does not render another H1 in the templates. This avoids duplicated headings; it is a template convention, not an engine rule.

## Error pages

`404.html` and `500.html` are parsed and executed once with `Globals` only. They can use `.CurrentYear` and `.Engine`, but not `.Name`, `.Root`, or other page data.
