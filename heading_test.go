package main

import "testing"

func Test_firstHeading(t *testing.T) {
	const src = `<body><p>Text</p><h1>Header <span>text</span></h1>`

	got, err := firstHeading([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if want := "Header text"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
