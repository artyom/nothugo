package main

import "testing"

func Test_createAnchors(t *testing.T) {
	const body = `<h1 class="foo">Some <span>header</span></h1><p>Text</p><h2>some header</h2>`
	got, err := createAnchors([]byte(body), true)
	if err != nil {
		t.Fatal(err)
	}
	const want = `<h1 class="foo" id="some-header">Some <span>header</span></h1><p>Text</p><h2 id="some-header-1">some header</h2>`
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
