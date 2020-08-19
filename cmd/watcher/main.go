package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/teamwork/watcher"
)

var (
	defaultMatch   = `(?:go\.mod|\.(?:go|tmpl))$`
	defaultExclude = `^vendor/`
)

func init() {
	flag.Usage = usage
}

func main() {
	l := log.New(os.Stderr, "(watcher) ", 0)

	var path arrFlag
	var match string
	var exclude string

	flag.Var(&path, "p", "paths to watch (default .)")
	flag.StringVar(&match, "m", defaultMatch, "match files (regexp)")
	flag.StringVar(&exclude, "e", defaultExclude, "exclude files (regexp)")
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		return
	}
	if path == nil {
		path = arrFlag{"."}
	}
	args := flag.Args()

	runCmd := watcher.Command(args...)
	updateFn := func(changes map[string]int) {
		for k, v := range changes {
			l.Printf("updates: %s (%d)\n", k, v)
		}

		l.Printf("run triggered...\n")
		err := runCmd(true).Wait()
		l.Printf("process interrupted: %v\n", err)
	}

	go updateFn(nil) // start the command before any changes

	opt := watcher.Options{
		Match:         match,
		Exclude:       exclude,
		Paths:         []string(path),
		HandleSignals: func(os.Signal) bool { return true },
	}
	if err := watcher.Watch(opt, updateFn); err != nil {
		l.Print("err:", err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr,
		`
Usage: 

  %s <flags> [cmd] <cmd args...>

Flags:

`, os.Args[0])

	flag.PrintDefaults()

	fmt.Fprintf(os.Stderr,
		`
Examples:

  %[1]s go run ./cmd/app)
	listen for current path and execute the go run
  %[1]s -p . -p /tmp/somedir go run ./cmd/app
	listen for changes on current and on a specific path 
  %[1]s -m '(\.go|\.txt)$' go run ./cmd/app
	listen for changes on .go and .txt files only
  %[1]s -e 'specific.go$' go run ./cmd/app
	listen for changes on except for specified match

`, os.Args[0])
}

type arrFlag []string

func (s *arrFlag) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(*s, " ")
}

func (s *arrFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}
