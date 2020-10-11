![Logo](http://svg.wiersma.co.za/glasslabs/looking-glass)

[![Go Report Card](https://goreportcard.com/badge/github.com/glasslabs/looking-glass)](https://goreportcard.com/report/github.com/glasslabs/looking-glass)
[![Build Status](https://travis-ci.com/glasslabs/looking-glass.svg?branch=master)](https://travis-ci.com/glasslabs/looking-glass)
[![Coverage Status](https://coveralls.io/repos/github/glasslabs/looking-glass/badge.svg?branch=master)](https://coveralls.io/github/glasslabs/looking-glass?branch=master)
[![GitHub release](https://img.shields.io/github/release/glasslabs/looking-glass.svg)](https://github.com/glasslabs/looking-glass/releases)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/glasslabs/looking-glass/master/LICENSE)

Smart mirror platform written in Go leveraging Yaegi.

## Table of Contents
* [Usage](#usage)
* [Configuration](#configuration)
* [Modules](#modules)

## Usage

### Run

Runs looking glass using the specified config and modules path.

```bash
glass run -c /path/to/config.yaml -m /path/to/modules
```

#### Options

**--secrets** FILE, **-s** FILE, **$SECRETS** *(Optional)*

The path to the YAML secrets file. Secrets can be accessed in the 
configuration using [Go template syntax](https://golang.org/pkg/text/template/) using the ".Secrets" prefix.

**--config** FILE, **-c** FILE, **$CONFIG** *(Required)*

The path to the YAML configuration file. This file will be parsed
using [Go template syntax](https://golang.org/pkg/text/template/). 

**--modules** PATH, **-m** PATH, **$MODULES** *(Required)*

The path to the modules. Module must be located under a `src` folder in the modules path.
The modules path should be writable to `looking-glass`. 

**--log.format** FORMAT, **$LOG_FORMAT** *(Default: "logfmt")*

Specify the format of logs. Supported formats: 'logfmt', 'json'.

**--log.level** LEVEL, **$LOG_LEVEL** *(Default: "info")*

Specify the log level. e.g. 'debug', 'warning'.

## Configuration

```yaml
ui:
  width:  640           # The width of the chrome window
  height: 480           # The height of the chrome window
  fullscreen: true      # If the chrome window should start fullscreen
modules:
  - name: simple-clock      # The name of the module (must be unique)
    path: clock             # The path to the module
    position: top:right     # The position of the module
  - name: simple-weather
    path: weather
    position: top:left
    config:
      locationId: 996506
      appId: {{ .Secrets.weather.appId }}
      units: metric
```

The module configuration can contain secrets from the secrets YAML prefixed with `.Secrets`
as shown in the example above. 

## Modules

You can discover modules on Github using [Github Search](https://github.com/search?q=topic%3Alooking-glass+topic%3Amodule+language%3AGo&ref=simplesearch).

### Package Naming

If your module uses a hyphen, which is not supported by Go, it will be assumed that the package name is the
last path of the hyphenated name (e.g. `looking-glass` would result in a package name `glass`). If this is not
the case for your module, the `Package` should be set in the module configuration to the correct package name and
should be documented in your module.

### Development

To make your module discoverable on Github, add the topics `looking-glass` and `module`.

Modules are parsed in [yaegi](http://github.com/traefik/yaegi) and must expose two functions to be loaded:

#### NewConfig

`NewConfig` exposes your configuration structure to looking glass. The function must return
a single structure with default values set. The yaml configuration will be decoded into
the returned structure, so it should contain `yaml` tags for the configuration to be decoded
properly.  

```go
func NewConfig() *Config 
```

#### New

`New` creates an instance of your module. It must return an `io.Closer` and an `error`.
The function takes a `context.Context`, the configuration structure returned by `NewConfig`,
`Info` and `UI` objects. `Info` and `UI` are located in `github.com/glasslabs/looking-glass/module/types`.

```go
func New(ctx context.Context, cfg *Config, info types.Info, ui types.UI) (io.Closer, error)
```

#### Dependencies

All dependencies must vendored except for `github.com/glasslabs/looking-glass/module/types`. 
If you still wish to use Go Modules for dependency management, you should run `go mod vendor` to 
vendor your dependencies and commit your `vendor` folder to git.

## TODO

This is very much a work in progress and under active development. The immediate list of
things to do is below:

* Better documentation
* Module download from config
