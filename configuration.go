package main

import (
	"path"
	"runtime"
	"strings"
	"time"
)

type configuration struct {
	AppRoot            string        `yaml:"app_root" json:"app_root"`
	IgnoredFolders     []string      `yaml:"ignored_folders" json:"ignored_folders"`
	IncludedExtensions []string      `yaml:"included_extensions" json:"included_extensions"`
	BuildTargetPath    string        `yaml:"build_target_path" json:"build_target_path"`
	BuildPath          string        `yaml:"build_path" json:"build_path"`
	BuildDelay         time.Duration `yaml:"build_delay" json:"build_delay"`
	BinaryName         string        `yaml:"binary_name" json:"binary_name"`
}

func (c *configuration) FullBuildPath() string {
	buildPath := path.Join(c.BuildPath, c.BinaryName)
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(strings.ToLower(buildPath), ".exe") {
			buildPath += ".exe"
		}
	}
	return buildPath
}
