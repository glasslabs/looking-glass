![Logo](http://svg.wiersma.co.za/glasslabs/looking-glass)

[![Go Report Card](https://goreportcard.com/badge/github.com/glasslabs/looking-glass)](https://goreportcard.com/report/github.com/glasslabs/looking-glass)
[![Build Status](https://github.com/glasslabs/looking-glass/actions/workflows/test.yml/badge.svg)](https://github.com/glasslabs/looking-glass/actions)
[![Coverage Status](https://coveralls.io/repos/github/glasslabs/looking-glass/badge.svg?branch=main)](https://coveralls.io/github/glasslabs/looking-glass?branch=main)
[![GitHub release](https://img.shields.io/github/release/glasslabs/looking-glass.svg)](https://github.com/glasslabs/looking-glass/releases)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/glasslabs/looking-glass/main/LICENSE)

Smart mirror platform written in Go. Modules are compiled to WebAssembly and loaded at runtime,
making it easy to add new functionality without modifying the core application.

## Table of Contents

- [Requirements](#requirements)
- [Installation](#installation)
- [Usage](#usage)
  - [Run](#run)
  - [Run Options](#run-options)
- [Configuration](#configuration)
  - [Configuration Options](#configuration-options)
  - [Template Variables](#template-variables)
- [Module Positions](#module-positions)
- [Modules](#modules)
  - [Development](#development)

## Requirements

A platform supported by [Gio](https://gioui.org): Linux, macOS, Windows, or Raspberry Pi OS.

## Installation

Pre-built binaries for all supported platforms are available on the
[releases page](https://github.com/glasslabs/looking-glass/releases). Download the
archive for your platform, extract it, and place the `glass` binary somewhere on your
`PATH`.

```shell
# Example for Linux amd64
curl -sSL https://github.com/glasslabs/looking-glass/releases/latest/download/glass_linux_amd64.tar.gz \
  | tar -xz -C /usr/local/bin glass
```

### Building from Source

If you prefer to build from source, Go 1.22 or later is required. Download the
embedded Roboto fonts and then build the binary:

```shell
make fonts
make build
```

## Usage

### Run

Run looking-glass with a configuration file, an assets directory, and a module cache directory.

```shell
glass run --config /path/to/config.yaml --assets /path/to/assets --modules /path/to/modules
```

### Run Options

**`--secrets` FILE, `-s` FILE, `$SECRETS`** *(optional)*

Path to a YAML file containing sensitive values. Secrets are available in the configuration
file as `.Secrets.<key>` using Go template syntax.

**`--config` FILE, `-c` FILE, `$CONFIG`** *(required)*

Path to the YAML configuration file. The file is rendered as a
[Go template](https://pkg.go.dev/text/template) before parsing.

**`--assets` PATH, `-a` PATH, `$ASSETS`** *(required)*

Path to the assets directory served to modules.

**`--modules` PATH, `-m` PATH, `$MODULES`** *(required)*

Path to the module cache directory. Downloaded WASM modules are stored here.

**`--log.format` FORMAT, `$LOG_FORMAT`** *(default: `logfmt`)*

Log output format. Supported values: `logfmt`, `json`, `console`.

**`--log.level` LEVEL, `$LOG_LEVEL`** *(default: `info`)*

Minimum log level. Supported values: `debug`, `info`, `warn`, `error`, `crit`.

## Configuration

```yaml
ui:
  width: 1024
  height: 760
  fullscreen: false
modules:
  - name: simple-clock
    uri: https://github.com/glasslabs/clock/releases/download/v1.0.0/clock.wasm
    position: top:right
  - name: simple-weather
    uri: https://github.com/glasslabs/weather/releases/download/v1.0.0/weather.wasm
    position: top:left
    config:
      locationId: 996506
      appId: {{ .Secrets.weather.appId }}
      units: metric
  - name: simple-calendar
    uri: https://github.com/glasslabs/calendar/releases/download/v1.0.0/calendar.wasm
    position: top:right
    config:
      timezone: Africa/Johannesburg
      maxDays: 5
      calendars:
        - url: {{ .Secrets.calendar.myCalendar }}
```

### Configuration Options

**`ui.width`**

Width of the window in pixels.

**`ui.height`**

Height of the window in pixels.

**`ui.fullscreen`**

Whether the window starts in fullscreen mode.

**`modules[].name`**

Unique name for the module. Used to identify the module within the layout.

**`modules[].uri`**

HTTP(S) URL or local file path to the module WASM file. The file is downloaded and
cached in the modules directory on first use.

**`modules[].position`**

Position of the module in the layout grid. See [Module Positions](#module-positions).

**`modules[].config`**

Arbitrary YAML configuration passed to the module at startup.

### Template Variables

The configuration file is rendered as a [Go template](https://pkg.go.dev/text/template)
before parsing. The following variables are available:

**`.Secrets`**

Values from the secrets file, accessible by key path (e.g. `.Secrets.weather.appId`).

**`.Env`**

Environment variables available at startup, accessible by name (e.g. `.Env.HOME`).

**`.ConfigPath`**

The directory containing the configuration file. Useful for constructing paths to
assets that live alongside the config.

## Module Positions

Modules are placed on a 3×3 grid. The position is specified as `<vertical>:<horizontal>`.

| | `left` | `center` | `right` |
|---|---|---|---|
| **`top`** | `top:left` | `top:center` | `top:right` |
| **`middle`** | `middle:left` | `middle:center` | `middle:right` |
| **`bottom`** | `bottom:left` | `bottom:center` | `bottom:right` |

Multiple modules can share the same position; they are stacked vertically within that cell.

## Modules

Discover community modules on GitHub using the
[`looking-glass` + `module` topics](https://github.com/search?q=topic%3Alooking-glass+topic%3Amodule+language%3AGo&ref=simplesearch).

### Development

Modules are Go programs compiled to `GOOS=wasip1 GOARCH=wasm`. The
[`glasslabs/client-go`](https://github.com/glasslabs/client-go) package provides the
host API for rendering widgets, logging, and making HTTP requests.

```shell
GOOS=wasip1 GOARCH=wasm go build -o my-module.wasm .
```

To make a module discoverable on GitHub, add the topics `looking-glass` and `module`
to the repository.
