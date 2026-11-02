# Getting started

notesite serves a folder of Markdown notes as a website. Point it at a notes directory and a template directory, run one binary, and the folder tree becomes the site.

## Requirements

- Go 1.18 or newer
- Git, optional and only used to stamp the version

## Build

```bash
git clone https://github.com/Streamlet/notesite.git
cd notesite
VERSION=$(git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo dev)
go build -o notesite -ldflags "-X github.com/Streamlet/notesite/global.Version=$VERSION" .
```

`VERSION` is the exact tag at HEAD, or the short commit id when HEAD is not tagged. Without the linker flag the binary reports `dev`.

## Run

```bash
./notesite --config=config/site.sample.toml
```

Open <http://localhost/>. The sample config listens on port 80; change `port` in `site.toml` if that port is unavailable.

The sample config points at `note/sample` and `template/sample`, so the command works from the repository root. Copy the config and change the paths when serving your own notes.

## What the binary does

- Reads Markdown, text, and HTML notes from `note_root`.
- Builds categories from the directory tree.
- Renders pages with the three templates in `template_root`.
- Serves static assets and resource directories directly.
- Watches notes and templates and reloads them without a restart.

`./notesite --version` prints the injected version.

## A minimal site

```text
notes/
  .cat
  [0]Start.public.md
  Guide/
    .cat
    [0]index.md
    [1]Notes.public.md
templates/
  index.template.html
  category.template.html
  content.template.html
  assets/site.css
```

A subdirectory becomes a category only when it contains `.cat`. By default, note files must end in `.public.md`, `.public.txt`, or `.public.html`; the `[0]` prefix only controls ordering and is not part of the URL.

## Next steps

- [Notes and URLs](/guide/notes)
- [Templates and assets](/guide/templates)
- [Configuration](/guide/config)
- [Deployment](/guide/deploy)
