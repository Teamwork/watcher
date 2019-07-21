# watcher

A watcher that doesn't care about GOPATH, or anything really, just watches
some paths and run

## Installation

```
go install github.com/teamwork/watcher/cmd/watcher
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
