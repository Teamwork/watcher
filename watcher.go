package watcher

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/shirou/gopsutil/process"

	// Auto loads .env files into current environment
	_ "github.com/joho/godotenv/autoload"
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
				if event.Op == fsnotify.Chmod {
					continue
				}
				fname := strings.TrimPrefix(event.Name, "/")
				if len(opt.Exclude) > 0 && excludeRe.MatchString(fname) {
					break
				}
				if matchRe.MatchString(fname) {
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

		cmd = exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Start()

		return cmd
	}
}

// Kill the process and all children
func kill(proc *os.Process) error {
	if proc == nil {
		return errors.New("nil process")
	}
	var kfn func(p *process.Process) error
	kfn = func(p *process.Process) error {
		// this uses pgrep :/
		children, err := p.Children()
		if err != process.ErrorNoChildren && err != nil {
			return err
		}
		for _, c := range children {
			if err := kfn(c); err != nil {
				return err
			}
		}
		return p.Kill()
	}

	p, err := process.NewProcess(int32(proc.Pid))
	if err != nil {
		return err
	}
	return kfn(p)
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
