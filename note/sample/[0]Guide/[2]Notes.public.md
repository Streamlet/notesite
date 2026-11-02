# Notes and URLs

notesite maps the note folder tree directly to URLs. There is no database and no publish step: if a file matches the public pattern, it is part of the site.

## Public files

The default pattern is:

```toml
note_file_pattern = "^(?:\\[.*?\\])*(.*)\\.public\\.(?:txt|html|md)$"
```

Only matching files are routed. Leading `[0]`, `[1]` groups are ignored for the name and exist only to control lexical order. The captured name becomes both the display label and the URL segment; the router lowercases URLs.

| File | URL | Label |
| --- | --- | --- |
| `[0]Start.public.md` | `/start` | Start |
| `[1]Notes.public.md` in a category with `name = "guide"` | `/guide/notes` | Notes |
| `[2]About.public.md` at the root | `/about` | About |

Use zero-padded prefixes (`[00]`, `[01]`) when a directory has more than nine items: ordering is lexical, not numeric.

## Categories

A subdirectory becomes a category only when it contains the configured category marker, `.cat` by default.

```toml
name = "guide"
display_name = "Guide"
index = "[0]Start.public.md"
# note_file_pattern = "..."
```

- `name` sets the URL segment and the fallback label.
- `display_name` overrides the label used in navigation.
- `index` supplies the content for the category page. If the index file also matches the public pattern, it remains reachable as a normal content page; the index files in this sample do not match, so they only serve the category page.
- `note_file_pattern` overrides the site pattern for that category and its descendants.

The root directory is always the site root. Its `.cat` can set `index` and `note_file_pattern`; `name` and `display_name` are ignored there.

Directory URLs end with `/` (`/guide/`). Content page URLs do not (`/guide/start`).

## Resources

A directory becomes a resource directory when it contains `.res`:

```toml
name = "images"
```

`name` is the URL prefix. Everything under the directory is served as a static file; resource directories do not appear in navigation and are not translated. This site's logo is served from `/images/logo.png`.

## Formats

| Extension | Rendering |
| --- | --- |
| `.md` | Goldmark with GFM and syntax highlighting |
| `.txt` | escaped HTML with line breaks preserved |
| `.html` | served as raw HTML |

Raw HTML in `.html` notes is not sanitized. Treat note content as trusted.

## Front matter

Markdown notes may start with YAML (`---`) or TOML (`+++`) front matter. Only `title` and `date` are understood:

```markdown
---
title: Getting started
date: 2026-11-03
---
```

`title` is prepended as an H1 and `date` as a text line. There is no description, slug, draft flag, or ordering weight in front matter; use file names and category config for those.

## URL rules

- Paths are matched case-insensitively and generated URLs are lowercase.
- Directory URLs end with `/`; content URLs do not.
- A category index note is also routed as a content page when its filename matches the public pattern.
- Hidden files and files that do not match the public pattern are not routed.
