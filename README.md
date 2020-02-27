# watcher

A watcher that doesn't care about GOPATH, or anything really, just watches
some paths and run

## Installation

```
go get github.com/teamwork/watcher/cmd/watcher
```

## Usage

```bash
watcher go run ./cmd/someservice
```

The example above will watch any file changes with the default matcher on
current working dir

## Defaults

**Match**: `(?:go\.mod|\.(?:go|tmpl))$`  
**Exclude**: `^vendor/`  
**Paths**: _current working dir_

## New

Support for setting ENV vars from .env files

```
$ echo "HELLO=world" > .env
$ watcher sh -c 'echo $HELLO'

(watcher) run triggered...
world
(watcher) process interrupted: <nil>
```
