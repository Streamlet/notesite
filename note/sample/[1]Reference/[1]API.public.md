# Template API

## PageData

```go
type PageData struct {
    Globals
    *BasicItem
    Content string
}
```

## BasicItem

| Member | Type | Notes |
| --- | --- | --- |
| `.Uri` | string | Registered URL; directory URLs end in `/` |
| `.Name` | string | Display label; empty on the root item |
| `.Parent` | `*BasicItem` | nil on the root item |
| `.Children` | `[]*BasicItem` | Navigation children; resource directories are excluded |
| `.IsAncestor` | bool | True on every item along the active root-to-page path, including the current item |
| `.Root` | method | Walks up to the root item |
| `.Ancestors` | method | Root-first slice including the current item and the root |
| `.Prev` / `.Next` | method | Adjacent sibling; nil at boundaries and on the root |
| `.HasChildren` | method | True when the item has navigation children |

## Globals

| Member | Notes |
| --- | --- |
| `.CurrentYear` | Current year, for example `2026` |
| `.Engine` | Product name and version, for example `notesite/v0.7` |

## Example

```html
<nav>
  {{ range .Root.Children }}
    <a href="{{ .Uri }}" class="{{ if .IsAncestor }}active{{ end }}">{{ .Name }}</a>
  {{ end }}
</nav>
```

## Notes

- `.Content` is already-rendered HTML and is inserted raw.
- Templates use `text/template`, not `html/template`: values are not escaped automatically. Apply `| html` to author-controlled text and attributes.
- The root item has an empty `.Name` and a nil `.Parent`.
- A category's index note is separate from `.Children` when its filename does not match the public pattern. If it does match, it also appears in navigation and is separately routable.
- `404` and `500` templates receive only `Globals`; page-data fields are unavailable there.

## Previous and next

```html
{{ with .Prev }}<a href="{{ .Uri }}">Previous: {{ .Name }}</a>{{ end }}
{{ with .Next }}<a href="{{ .Uri }}">Next: {{ .Name }}</a>{{ end }}
```

These helpers follow sibling order inside a category. Root-level items have no meaningful reading order.
