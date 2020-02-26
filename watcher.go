package watcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	// Auto loads .env files into current environment
	"github.com/joho/godotenv"
)

// UpdateFunc type of the func that will be called after changes
type UpdateFunc func(map[string]int)

// Options for the watcher struct
type Options struct {
	// Match if the regexp matches the run command will be restarted
	Match string
	// Exclude if any file changed matches exclude it will ignore
	Exclude string
	// Path to watch
	Paths []string

	// this channel will be closed when the watcher is ready, useful in case
	// watcher is started within a go routine and we want to wait until ready
	Ready chan struct{}
}

// Watch filed in background
func Watch(opt Options, run UpdateFunc) error {

	matchRe, err := regexp.Compile(opt.Match)
	if err != nil {
		return err
	}
	excludeRe, err := regexp.Compile(opt.Exclude)
	if err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// This will track file changes
	mu := sync.Mutex{}
	changes := map[string]int{}
	changesFn := func() {
		mu.Lock()
		defer mu.Unlock()
		go run(changes)
		changes = map[string]int{}
	}
	update := debounce(time.Millisecond*500, changesFn)

	done := make(chan error)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op == fsnotify.Chmod || event.Name == "" {
					break
				}
				fname := filepath.Clean(strings.TrimPrefix(event.Name, "/"))
				if fname != ".env" && len(opt.Exclude) > 0 && excludeRe.MatchString(fname) {
					break
				}

				switch event.Op {
				case fsnotify.Remove:
					watcher.Remove(fname) // in case it is watched a dir
				case fsnotify.Create:
					// Ignoring error here as it doesn't affect the flow
					finfo, _ := os.Stat(fname) // nolint: errcheck
					if finfo != nil && finfo.IsDir() {
						watcher.Add(fname)
					}
				}
				if matchRe.MatchString(fname) || fname == ".env" {
					mu.Lock()
					changes[fname]++
					mu.Unlock()
					update()
				}
			case err := <-watcher.Errors:
				done <- err
				return
			}
		}
	}()
	for _, path := range opt.Paths {
		err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				return nil
			}
			if len(opt.Exclude) > 0 && excludeRe.MatchString(p) {
				return nil
			}
			// Hardcoded exclude of .git folder
			if strings.HasPrefix(p, ".git") {
				return nil
			}
			return watcher.Add(p)
		})
		if err != nil {
			return err
		}
	}

	if opt.Ready != nil {
		close(opt.Ready)
	}
	return <-done
}

// CommandFunc for the Command
type CommandFunc func(bool) *exec.Cmd

// Command returns a CommandFunc that when called will always kill the
// previously command and will start a new one if the start flag is true
func Command(args ...string) CommandFunc {
	if len(args) < 1 {
		panic("invalid number of args")
	}
	var mu sync.Mutex
	var cmd *exec.Cmd
	return func(start bool) *exec.Cmd {
		mu.Lock()
		defer mu.Unlock()

		if cmd != nil {
			kill(cmd.Process)
		}
		if !start {
			return nil
		}

		osEnv := os.Environ()
		// ignoring error as doesn't affect the execution
		if env, _ := godotenv.Read(); env != nil { // nolint: errcheck
			for k, v := range env {
				osEnv = append(osEnv, fmt.Sprintf("%s=%s", k, v))
			}
		}

		cmd = startProcess(osEnv, args...)
		return cmd
	}
}

// debounce delays the execution of fn to avoid multiple fast calls it will
// call the funcs in a routine
func debounce(d time.Duration, fn func()) func() {
	if fn == nil {
		panic("fn must be set")
	}
	controlChan := make(chan struct{})
	go func() {
		t := time.NewTimer(d)
		t.Stop()
		for {
			select {
			case <-controlChan:
				t.Reset(d)
			case <-t.C:
				go fn()
			}
		}
	}()
	return func() {
		controlChan <- struct{}{}
	}
}
