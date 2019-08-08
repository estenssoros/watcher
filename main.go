package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var projectName string

func init() {
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	logrus.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	wd, _ := os.Getwd()
	projectName = filepath.Base(wd)
}

var rootCmd = &cobra.Command{
	Use:   "watcher",
	Short: "watches for files and reloads app",
	RunE: func(cmd *cobra.Command, args []string) error {
		defer func() {
			if r := recover(); r != nil {
				logrus.Error(r)
			}
		}()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		return errors.Wrap(startApp(ctx), "start dev server")
	},
}

func startApp(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("ERROR: ", r)
		}
	}()
	c := &configuration{
		AppRoot:            ".",
		IgnoredFolders:     []string{"vendor"},
		IncludedExtensions: []string{".go"},
		BuildPath:          "tmp",
		BuildDelay:         time.Duration(200),
		BinaryName:         projectName,
	}
	return newManager(ctx, c).start()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}
