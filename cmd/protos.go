package main

import (
	"os"
	"path/filepath"
	"runtime"
)

func projectRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filename))
}

func genRoot() string {
	return filepath.Join(projectRoot(), "nexusapi")
}

func protoRoot() string {
	return filepath.Join(projectRoot(), "proto")
}

func findProtos() ([]string, error) {
	dir := filepath.Join(protoRoot(), "nexus")
	files := []string{}
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if filepath.Ext(path) == ".proto" {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}
