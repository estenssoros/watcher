package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

type watcher struct {
	*fsnotify.Watcher
	*manager
	context context.Context
}

func newWatcher(r *manager) *watcher {
	w, _ := fsnotify.NewWatcher()
	return &watcher{
		Watcher: w,
		manager: r,
		context: r.context,
	}
}

// start scans directory and adds files to watched list
func (w *watcher) start() {
	go func() {
		for {
			var count int
			err := filepath.Walk(w.AppRoot, func(path string, info os.FileInfo, err error) error {
				if info == nil {
					w.cancelFunc()
					return errors.New("nil directory")
				}

				if info.IsDir() {
					if strings.HasPrefix(filepath.Base(path), "_") {
						return filepath.SkipDir
					}
					if len(path) > 1 && strings.HasPrefix(filepath.Base(path), ".") || w.isIgnoredFolder(path) {
						logrus.Infof("skipping %s", path)
						return filepath.SkipDir
					}
				}
				if strings.HasSuffix(path, "test.go") {
					return filepath.SkipDir
				}

				if w.isWatchedFile(path) {
					count++
					w.Add(path)
				}

				return nil
			})
			logrus.Infof("watching %d files", count)
			if err != nil {
				w.context.Done()
				break
			}
			// sweep for new files every 1 second
			time.Sleep(1 * time.Second)
		}
	}()
}

func (w watcher) isIgnoredFolder(path string) bool {
	paths := strings.Split(path, "/")
	if len(paths) <= 0 {
		return false
	}

	for _, e := range w.IgnoredFolders {
		if strings.TrimSpace(e) == paths[0] {
			return true
		}
	}
	return false
}

func (w watcher) isWatchedFile(path string) bool {
	ext := filepath.Ext(path)

	for _, e := range w.IncludedExtensions {
		if strings.TrimSpace(e) == ext {
			return true
		}
	}

	return false
}
