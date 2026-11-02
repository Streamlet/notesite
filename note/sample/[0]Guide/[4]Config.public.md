# Configuration

notesite reads TOML. Pass the file with `--config`; the default is `site.toml`.

```toml
[server]
port = 80

[template]
template_root = "template/sample"
static_dirs = ["assets"]
index_template = "index.template.html"
category_template = "category.template.html"
content_template = "content.template.html"
404 = "404.html"
500 = "500.html"

[note]
note_root = "note/sample"
category_config_file = ".cat"
resource_config_file = ".res"
note_file_pattern = "^(?:\\[.*?\\])*(.*)\\.public\\.(?:txt|html|md)$"
```

## `[server]`

- `port` — TCP port. Set it to 0 when using a socket.
- `sock` — Unix socket path. Set the port to 0 when using a socket.

Exactly one listener must be configured. The checked-in sample uses port 80; if that port is unavailable, set a higher port such as 8080 and use it in the URL.

## `[template]`

- `template_root` — directory containing the templates and `static_dirs`.
- `static_dirs` — directories whose files are served at the URL root.
- `index_template`, `category_template`, `content_template` — required template file names, relative to `template_root`.
- `404`, `500` — optional error page names.

## `[note]`

- `note_root` — directory containing the notes.
- `category_config_file` — marker file for categories; `.cat` in this sample.
- `resource_config_file` — marker file for resource directories; `.res` in this sample.
- `note_file_pattern` — regular expression selecting public files.

## Category config

Place a `.cat` file in a directory to make it a category:

```toml
name = "guide"
display_name = "Guide"
index = "[0]index.md"
# note_file_pattern = "..."
```

- `name` sets the URL segment and fallback label.
- `display_name` overrides the label.
- `index` is the note whose translated HTML fills the category page.
- `note_file_pattern` overrides the site pattern for this category and its descendants.

## Resource config

Place a `.res` file in a directory to serve it as static files:

```toml
name = "images"
```

`name` is the URL prefix and defaults to the directory name. Resource directories are hidden from navigation.

## What is watched

- Template files reload automatically.
- Notes are read per request.
- Category and resource configs rebuild the route tree.
- The site config is read once at startup; restart after changing it.

See [Config reference](/reference/config) for every field.
