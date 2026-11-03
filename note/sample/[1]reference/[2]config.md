# Config reference

## Site config

| Section | Field | Required | Notes |
| --- | --- | --- | --- |
| `[server]` | `port` | one of port/sock | TCP port; set to 0 when using a socket |
| | `sock` | one of port/sock | Unix socket path |
| `[template]` | `template_root` | yes | Template and asset root |
| | `static_dirs` | no | Directories served at the URL root |
| | `index_template` | yes | Root page template |
| | `category_template` | yes | Category page template |
| | `content_template` | yes | Note template |
| | `404` | no | Not-found page |
| | `500` | no | Error page |
| `[note]` | `note_root` | yes | Notes root |
| | `category_config_file` | yes | Category marker file (`.cat`) |
| | `resource_config_file` | yes | Resource marker file (`.res`) |
| | `note_file_pattern` | yes | Public file pattern |

## Category config

| Field | Notes |
| --- | --- |
| `name` | URL segment and fallback label |
| `display_name` | Navigation label; overrides `name` |
| `index` | File whose translated HTML fills the category page |
| `note_file_pattern` | Pattern override for this category and its descendants |

A subdirectory is a category only if it contains the configured marker. The root is always the root; its `name` and `display_name` are ignored.

## Resource config

| Field | Notes |
| --- | --- |
| `name` | URL prefix; defaults to the directory name |

Resource trees are served as static files and are not added to navigation.

## `note_file_pattern`

Capture group 1 becomes the URL segment and display name. Capture group 2, when present, overrides the display name. A group-1 value ending in `.` produces a directory-style URL with a trailing `/`. Generated URLs are lowercased.

The sample config uses:

```toml
note_file_pattern = "^(?:\\[.*?\\])(.*)\\.(?:txt|html|md)$"
```

## Static directories

Each entry in `static_dirs` is flattened to the URL root. With `static_dirs = ["assets"]`, `assets/site.css` is served as `/site.css`. Files are served directly; directory listings are not generated.

## Rebuild behavior

- Note files: read per request, cached by modification time and size.
- Templates: reloaded when they change.
- `.cat` and `.res`: rebuild the route tree.
- `site.toml`: read once; restart required.
