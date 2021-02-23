package main

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func generateExampleContent(rootDir, templatesDir string) error {
	if rootDir == templatesDir {
		return errors.New("source and templates directories cannot be the same")
	}
	fn := func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := exampleContent.ReadFile(p)
		if err != nil {
			return err
		}
		dst := filepath.Join(rootDir, filepath.FromSlash(strings.TrimPrefix(p, examplesRoot)))
		return writeIfNotExists(dst, b)
	}
	if err := fs.WalkDir(exampleContent, ".", fn); err != nil {
		return err
	}
	return writeIfNotExists(filepath.Join(templatesDir, "default.html"), exampleTemplate)
}

//go:embed example-content
var exampleContent embed.FS

const examplesRoot = "example-content"

//go:embed example-template.html
var exampleTemplate []byte

// writeIfNotExists creates file at dst and writes b as its content. It fails
// if file already exists. Parent directories created as necessary.
func writeIfNotExists(dst string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0777); err != nil {
		return err
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(b); err != nil {
		return err
	}
	return f.Close()
}
