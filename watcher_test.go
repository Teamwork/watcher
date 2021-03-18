package watcher

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDebounce(t *testing.T) {
	count := int32(0)
	fn := debounce(time.Millisecond*200, func() {
		atomic.AddInt32(&count, 1)
	})
	for i := 0; i < 5; i++ {
		fn()
	}
	time.Sleep(time.Second)
	if atomic.LoadInt32(&count) != 1 {
		t.Fatal("should only have one call only")
	}
}

func TestCommand(t *testing.T) {
	fn := Command("go", "run", "./test/test.go")

	checked := make(chan error)
	cmd := fn(true)
	go func() {
		checked <- cmd.Wait()
	}()

	fn(false)

	err := <-checked
	if err == nil || !strings.Contains(err.Error(), "killed") {
		t.Fatal("thing", err)
	}
}

func TestWatch(t *testing.T) {
	testFile := "./test/test.txt"
	opt := Options{
		Match: `(?:\.go|\.txt)$`,
		Paths: []string{"./test"},
		Ready: make(chan struct{}),
	}

	checked := make(chan error)
	// nolint: errcheck
	go Watch(opt, func(changes map[string]int) {
		if _, ok := changes["test/test.txt"]; !ok {
			checked <- errors.New("not found")
			return
		}
		checked <- nil
	})
	<-opt.Ready

	if err := ioutil.WriteFile(testFile, []byte{1}, os.FileMode(0644)); err != nil {
		t.Fatal("creating test file", err)
	}
	defer os.Remove(testFile) // nolint: errcheck

	select {
	case err := <-checked:
		if err != nil {
			t.Fatal("target file not found")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout expecting any changes")
	}
}
