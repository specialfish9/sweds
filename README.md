# Simple WEb Dav Server (SWEDS)

A minimal WebDAV server in Go, backed by `golang.org/x/net/webdav`.

## Configuration

SWEDS is configured through a YAML file. The file path is read from
the `CONFIG_PATH` environment variable and defaults to `config.yaml` in the
current directory.

All keys are optional and fall back to the defaults below:

| Key      | Default    | Description                                       |
| -------- | ---------- | ------------------------------------------------- |
| `addr`   | `:8080`    | Listen address                                    |
| `dir`    | `./data`   | Directory to serve (confined; `..` cannot escape) |
| `prefix` | `""`       | URL prefix to mount under, e.g. `/dav`            |
| `user`   | `""`       | Basic auth username (empty disables auth)         |
| `pass`   | `""`       | Basic auth password                               |
| `cert`   | `""`       | TLS certificate file (enables HTTPS with `key`)   |
| `key`    | `""`       | TLS key file                                      |

Example `config.yaml`:

```yaml
addr: ":8080"
dir: "./data"
prefix: ""
user: "alice"
pass: "secret"
# cert: "cert.pem"
# key: "key.pem"
```

## Run

```sh
CONFIG_PATH=./config.yaml go run .
```

If `CONFIG_PATH` is unset, `config.yaml` in the working directory is used.

## Test

```sh
curl -u alice:secret -X MKCOL http://localhost:8080/docs/
curl -u alice:secret -T hello.txt http://localhost:8080/docs/hello.txt
curl -u alice:secret -X PROPFIND -H "Depth: 1" http://localhost:8080/docs/
```

## Mount

- **Linux (davfs2):** `sudo mount -t davfs http://localhost:8080/ /mnt/dav`
- **macOS Finder:** Go > Connect to Server > `http://localhost:8080/`
- **Windows:** Map network drive (needs HTTPS, or a registry tweak for Basic over HTTP)

## Docker

The image is a multi-stage build producing a static binary on a
[distroless](https://github.com/GoogleContainerTools/distroless) base
(`gcr.io/distroless/static-debian12:nonroot`). A sample `config.yaml` is baked
in at `/config.yaml` (`CONFIG_PATH` points there by default).

Prebuilt images are published to Docker Hub as
[`specialfish9/sweds`](https://hub.docker.com/r/specialfish9/sweds):

```sh
docker pull specialfish9/sweds:latest
```

Or build it yourself:

```sh
docker build -t specialfish9/sweds:latest .
```

Run with a persisted data volume (served dir defaults to `./data`, i.e.
`/home/nonroot/data` inside the container):

```sh
docker run -p 8080:8080 -v sweds-data:/home/nonroot/data specialfish9/sweds:latest
```

Override the baked config with your own:

```sh
docker run -p 8080:8080 \
  -v $PWD/config.yaml:/config.yaml \
  -v sweds-data:/home/nonroot/data \
  specialfish9/sweds:latest
```

### Compose

```sh
docker compose up --build
```

The bundled `compose.yaml` builds the image, persists data in the `sweds-data`
volume, and bind-mounts `./config.yaml` over the baked config.

