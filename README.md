![Logo](http://svg.wiersma.co.za/glasslabs/looking-glass)

[![Go Report Card](https://goreportcard.com/badge/github.com/glasslabs/looking-glass)](https://goreportcard.com/report/github.com/glasslabs/looking-glass)
[![Build Status](https://github.com/glasslabs/looking-glass/actions/workflows/test.yml/badge.svg)](https://github.com/glasslabs/looking-glass/actions)
[![Coverage Status](https://coveralls.io/repos/github/glasslabs/looking-glass/badge.svg?branch=main)](https://coveralls.io/github/glasslabs/looking-glass?branch=main)
[![GitHub release](https://img.shields.io/github/release/glasslabs/looking-glass.svg)](https://github.com/glasslabs/looking-glass/releases)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://raw.githubusercontent.com/glasslabs/looking-glass/main/LICENSE)

Smart mirror platform written in Go leveraging WASM.

## Table of Contents
* [Requirements](#requirements)
* [Usage](#usage)
    * [Run](#run) ([Options](#run-options))
* [Configuration](#configuration)
    * [Configuration Options](#configuration-options)
    * [Configuration Variables](#configuration-variables)
* [Modules](#modules)
    * [Package Naming](#package-naming)
    * [Development](#development)

## Requirements

**Chrome**

Chrome or Chromium must be installed. The version must be greater or equal to 70. If looking glass cannot find 
Chrome, use the `LORCACHROME` environment variable to force the location of your installation.

## Install

On a fresh install of Raspberry Pi OS Lite, run the following command:

```shell
bash -c "$(curl -fsSL https://git.io/looking_glass)"
``` 

## Usage

### Run

Runs looking glass using the specified configuration and modules path.

```bash
glass run -c /path/to/config.yaml -m /path/to/modules
```

#### Run Options

**--secrets** FILE, **-s** FILE, **$SECRETS** *(Optional)*

The path to the YAML secrets file to hold sensitive configuration values. Secrets can be accessed in the 
configuration using [Go template syntax](https://golang.org/pkg/text/template/) using the ".Secrets" prefix.

**--config** FILE, **-c** FILE, **$CONFIG** *(Required)*

The path to the YAML configuration file for `looking-glass` which includes module configuration. 
This file will be parsed using [Go template syntax](https://golang.org/pkg/text/template/).

**--assets** PATH, **-a** PATH, **$ASSETS** *(Required)*

The path to the assets.

**--modules** PATH, **-m** PATH, **$MODULES** *(Required)*

The path to the module cache. 

**--log.format** FORMAT, **$LOG_FORMAT** *(Default: "logfmt")*

Specify the format of logs. Supported formats: 'logfmt', 'json', 'console'.

**--log.level** LEVEL, **$LOG_LEVEL** *(Default: "info")*

Specify the log level. Supported levels: 'debug', 'info', 'warn', 'error', 'crit'.

## Configuration

```yaml
ui:
  width:  640
  height: 480
  fullscreen: true
  customCss:
    - path/to/custom.css
modules:
  - name: simple-clock
    url: https://github.com/glasslabs/clock/releases/download/v1.0.0/clock.wasm
    position: top:right
  - name: simple-weather
    uri: https://github.com/glasslabs/weather/releases/download/v1.0.0/weather.wasm
    position: top:left
    config:
      locationId: 996506
      appId: {{ .Secrets.weather.appId }}
      units: metric
```

The module configuration can contain secrets from the secrets YAML prefixed with `.Secrets`
as shown in the example above.

### Configuration Options

**ui.width**

The width of the chrome window.

**ui.height**

The height of the chrome window.

**ui.fullscreen**

If the chrome window should start fullscreen.

**ui.customCSS**

A list of custom css files to load. These can be used to customise the layout of looking glass.

**modules.[].name**

The name of the module. This name must be unique. This is used as the ID of the module HTML wrapper.

**modules.[].url**

The module URL or path of the module under the modules path.

**modules.[].position**

The position of the module.

**modules.[].config**

The configuration that will be passed to the module.

### Configuration Variables

The configuration file will be parsed using [Go template syntax](https://golang.org/pkg/text/template/). The available variables are:

**Secrets**

The secrets in the case they appear in the secrets file.

**Env**

The environment variables available when running looking-glass.

## Modules

You can discover modules on GitHub using [GitHub Search](https://github.com/search?q=topic%3Alooking-glass+topic%3Amodule+language%3AGo&ref=simplesearch).

### Development

To make your module discoverable on GitHub, add the topics `looking-glass` and `module`.

## TODO

This is very much a work in progress and under active development. The immediate list of
things to do is below:

* Localisation
* Testing Framework for Modules
