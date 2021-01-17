package main

import (
	"testing"
)

func Test_rewriteLinks(t *testing.T) {
	const body = `<p>Link: <a href="//example.com/foo.md">link1</a>, <a href="/bar.md">link2</a></p>`
	const want = `<p>Link: <a href="//example.com/foo.md">link1</a>, <a href="/bar.html">link2</a></p>`
	got, err := rewriteLinks([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
