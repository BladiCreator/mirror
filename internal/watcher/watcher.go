package watcher

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watch runs onChange when any watched file is modified.
func Watch(paths []string, onChange func() error, stopCh <-chan struct{}) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if err := watcher.Add(abs); err != nil {
			return fmt.Errorf("watch add %s: %w", abs, err)
		}
	}

	debounce := time.NewTimer(300 * time.Millisecond)
	debounce.Stop()
	requeued := false

	trigger := func() {
		if !debounce.Stop() {
			select {
			case <-debounce.C:
			default:
			}
		}
		debounce.Reset(300 * time.Millisecond)
	}

	for {
		select {
		case <-stopCh:
			return nil
		case ev := <-watcher.Events:
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				requeued = true
				trigger()
			}
		case err := <-watcher.Errors:
			return err
		case <-debounce.C:
			if requeued {
				requeued = false
				if err := onChange(); err != nil {
					fmt.Println("watch error:", err)
				}
			}
		}
	}
}
