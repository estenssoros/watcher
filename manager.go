package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

func id() string {
	d, _ := os.Getwd()
	return fmt.Sprintf("%x", md5.Sum([]byte(d)))
}

type manager struct {
	*configuration
	Restart    chan bool
	gil        *sync.Once
	ID         string
	context    context.Context
	cancelFunc context.CancelFunc
}

func newManager(ctx context.Context, c *configuration) *manager {
	ctx, cancelFunc := context.WithCancel(ctx)
	return &manager{
		configuration: c,
		Restart:       make(chan bool),
		gil:           &sync.Once{},
		ID:            id(),
		context:       ctx,
		cancelFunc:    cancelFunc,
	}
}

// runs build and reports on error
func (m *manager) buildTransaction(fn func() error) {
	if err := fn(); err != nil {
		logrus.Error(err)
	}
}

// runAndListen runs a command and reports the output to console
func (m *manager) runAndListen(cmd *exec.Cmd) error {
	var stderr bytes.Buffer
	mw := io.MultiWriter(&stderr, os.Stderr)
	cmd.Stderr = mw
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	err := cmd.Start()
	if err != nil {
		return fmt.Errorf("%s\n%s", err, stderr.String())
	}

	logrus.Infof("running: %s (PID: %d)", strings.Join(cmd.Args, " "), cmd.Process.Pid)
	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("%s\n%s", err, stderr.String())
	}
	return nil
}

// build builds a go binary to run
func (m *manager) build(event fsnotify.Event) {
	m.gil.Do(func() {

		defer func() {
			m.gil = &sync.Once{}
		}()

		m.buildTransaction(func() error {
			time.Sleep(m.BuildDelay * time.Millisecond)

			now := time.Now()
			logrus.Infof("Rebuild on: %s", event.Name)

			args := []string{"build", "-v"}
			args = append(args, "-o", m.FullBuildPath(), m.BuildTargetPath)
			cmd := exec.Command("go", args...)

			if err := m.runAndListen(cmd); err != nil {
				if strings.Contains(err.Error(), "no buildable Go source files") {
					m.cancelFunc()
					log.Fatal(err)
				}
				return err
			}

			tt := time.Since(now)
			logrus.Infof("Building Completed (PID: %d) (Time: %s)", cmd.Process.Pid, tt)
			m.Restart <- true
			return nil
		})
	})
}

// runner kills an old command and starts a new one
func (m *manager) runner() {
	var cmd *exec.Cmd
	for {
		<-m.Restart
		if cmd != nil {
			pid := cmd.Process.Pid
			logrus.Infof("Stopping: PID %d", pid)
			cmd.Process.Kill()
		}
		go func() {
			err := m.runAndListen(cmd)
			if err != nil {
				logrus.Error(err)
			}
		}()
	}
}

// start creates a watcher for fs events, builds, and monitors
func (m *manager) start() error {
	w := newWatcher(m)
	w.start()
	go m.build(fsnotify.Event{Name: ":start:"})
	// watch files
	go func() {
		log.Println("watching files...")
		for {
			select {
			case event := <-w.Events:
				if event.Op != fsnotify.Chmod {
					go m.build(event)
				}
				w.Remove(event.Name)
				w.Add(event.Name)
			case <-m.context.Done():
				break
			}
		}
	}()

	go func() {
		for {
			select {
			case err := <-w.Errors:
				logrus.Error(err)
			case <-m.context.Done():
				break
			}
		}
	}()
	m.runner()
	return nil
}
