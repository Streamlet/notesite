# Deployment

## Build a release binary

```bash
VERSION=$(git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo dev)
CGO_ENABLED=0 go build -trimpath -o notesite \
  -ldflags "-s -w -X github.com/Streamlet/notesite/global.Version=$VERSION" .
```

`-ldflags -X` injects the version into `global.Version` at link time; without it the binary reports `latest`. `-s -w` strips debug data; the injected version is still available through `--version` and the template `.Engine`.

## Docker

The included Dockerfile builds the binary, copies the sample templates and notes, and generates a `site.toml` from the sample config.

```bash
docker build -t notesite .
docker run --rm -p 8080:80 notesite
```

Open <http://localhost:8080/>. To serve content from the host:

```bash
docker run --rm -p 8080:80 \
  -v "$PWD/note/sample:/usr/local/notesite/note:ro" \
  -v "$PWD/template/sample:/usr/local/notesite/template:ro" \
  notesite
```

The container listens on port 80 because its generated `site.toml` comes from the sample config.

## Socket

```toml
[server]
port = 0
sock = "/var/run/notesite.sock"
```

Ensure the process can create the socket and the reverse proxy can read it.

## Reverse proxy

Let nginx or another proxy terminate TLS and handle compression and caching. A socket upstream:

```nginx
location / {
    proxy_pass http://unix:/var/run/notesite.sock:/;
}
```

## Security

notesite does not terminate TLS or provide authentication. Put it behind a reverse proxy when it is reachable from outside localhost.

Everything that matches the public note pattern, and every file inside a resource directory, is public. Do not store secrets in `note_root`. Treat `.html` notes and templates as trusted code; they are served or executed without sanitization.

## Versioning

`global.Version` defaults to `latest`. A release build overrides it with the linker flag shown above, using the exact tag at HEAD or the short commit id when HEAD is not tagged. The CI workflow applies the same rule and packages the `notesite` binary. `--version` prints the value; templates receive it as `.Engine`, for example `notesite/v0.7`.

## Updates

- Note content is read per request.
- Template and include changes reload automatically.
- Category and resource config changes rebuild the route tree.
- `site.toml` changes require a restart.
