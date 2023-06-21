[![Go Report](https://goreportcard.com/badge/github.com/topi314/goapps)](https://goreportcard.com/report/github.com/topi314/goapps)
[![Go Version](https://img.shields.io/github/go-mod/go-version/topi314/goapps)](https://golang.org/doc/devel/release.html)
[![goapps License](https://img.shields.io/github/license/topi314/goapps)](LICENSE)
[![goapps Version](https://img.shields.io/github/v/tag/topi314/goapps?label=release)](https://github.com/topi314/goapps/releases/latest)
[![Docker](https://github.com/topi314/goapps/actions/workflows/docker.yml/badge.svg)](https://github.com/topi314/goapps/actions/workflows/docker.yml)
[![Discord](https://discordapp.com/api/guilds/608506410803658753/embed.png?style=shield)](https://discord.gg/sD3ABd5)

# goapps

A simple service dashboard with custom icons, names and links.
Services can be hidden depending on OIDC groups & users.

<img src=".github/preview.png" alt="preview">

<details>
<summary>Table of Contents</summary>

- [Features](#features)
- [Installation](#installation)
  - [Docker](#docker)
    - [Docker Compose](#docker-compose)
  - [Manual](#manual)
    - [Requirements](#requirements)
    - [Build](#build)
    - [Run](#run)
- [Configuration](#configuration)
- [License](#license)
- [Contributing](#contributing)
- [Contact](#contact)
</details>

## Features

- OIDC Authentication
- Customizable icons, names, descriptions and links
- Hide services depending on OIDC groups & users
- Dark & Light mode
- Responsive design
- Docker support
- Easy to use
- No database required

## Installation

### Docker

The easiest way to deploy goapps is using docker with [Docker Compose](https://docs.docker.com/compose/). You can find the docker image on [Packages](https://github.com/topi314/goapps/pkgs/container/goapps).

#### Docker Compose

Create a new `docker-compose.yml` file with the following content:

```yaml
version: "3.8"

services:
  goapps:
    image: ghcr.io/topi314/goapps:latest
    container_name: goapps
    restart: unless-stopped
    volumes:
      - ./goapps.yml:/var/lib/goapps/goapps.yml:ro
      - ./icons/:/var/lib/goapps/icons/:ro
    ports:
      - 80:80
```

For `goapps.yml` see [Configuration](#configuration).

```bash
docker-compose up -d
```

---

### Manual


#### Requirements

- Go 1.20 or higher

#### Build

```bash
git clone https://github.com/topi314/goapps.git
cd goapps
go build -o goapps
```

or

```bash
go install github.com/topi314/goapps@latest
```

#### Run

```bash
goapps --config=goapps.yml
```

---

## Configuration

Create a new `goapps.yml` file with the following content:


```yml
  log:
    # log level, either "debug", "info", "warn" or "error"
    level: info
    # log format, either "json" or "text"
    format: text
    # whether to add the source file and line to the log output
    add_source: false

  # enable or disable hot reload of templates and assets
  dev_mode: false
  # enable or disable debug profiler endpoint
  debug: false

  server:
    # on which address & port to listen
    listen_addr: 0.0.0.0:80
    # the title of the page
    title: goapps
    # the icon of the page
    icon: icon/goapps.png
    # where to find the custom icons
    icons_dir: ./icon

  # auth configuration for OIDC
  auth:
    # if the site is http or https
    secure: true
    # the OIDC issuer URL
    issuer_url: https://auth.example.com
    # the client ID
    client_id: goapps
    # the client secret
    client_secret: secret
    # the redirect URL for the OIDC callback
    redirect_url: https://example.com/callback

  # the services to display on the dashboard
  services:
      # the name of the service
    - name: example
      # the description of the service (optional)
      description: example service
      # the icon of the service (can also be a url to an external site) (optional)
      icon: icon/example.png
      # the link to the service
      link: https://example.com
      # the groups that can see the service (optional)
      groups: [ group1 ]
      # the users that can see the service (optional)
      users: [ user1 ]
}
```

---

## License

goapps is licensed under the [Apache License 2.0](/LICENSE).

---

## Contributing

Contributions are always welcome! Just open a pull request or discussion and I will take a look at it.

---

## Contact

- [Discord](https://discord.gg/sD3ABd5)
- [Twitter](https://twitter.com/topi314)
- [Email](mailto:git@topi.wtf)
