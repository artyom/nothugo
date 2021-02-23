package main

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func Test_generateExampleContent(t *testing.T) {
	dir := t.TempDir()
	err := generateExampleContent(filepath.Join(dir, "root"), filepath.Join(dir, "tpl"))
	if err != nil {
		t.Fatal(err)
	}
	fn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		t.Log(strings.TrimPrefix(path, dir))
		return nil
	}
	if err := filepath.WalkDir(dir, fn); err != nil {
		t.Fatal(err)
	}
}
