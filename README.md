![autobuild](https://github.com/Streamlet/notesite/workflows/autobuild/badge.svg)

# notesite
a dynamic website system generated from notes.

## Introduction

## Build & Run
```bash
VERSION=$(git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo dev)
go build -o notesite -ldflags "-X github.com/Streamlet/notesite/global.Version=$VERSION"
./notesite --config=config/site.sample.toml
```

Then use your browser to visit <http://localhost/> .

## Demo

<https://notesite.streamsoft.net/>
