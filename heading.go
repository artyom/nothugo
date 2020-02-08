package main

import (
	"bytes"
	"io"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// firstHeading partially parses b as utf-8 encoded HTML text and returns text
// of the first <h1> element it finds. If no <h1> element is found, but HTML
// was parsed successfully, result would be an empty string and nil error.
func firstHeading(b []byte) (string, error) {
	z := html.NewTokenizer(bytes.NewReader(b))
	var textBuilder strings.Builder
	var inHeading bool // if we're inside H1
	// tokenize:
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				return "", nil
			}
			return "", z.Err()
		case html.TextToken:
			if inHeading {
				textBuilder.Write(z.Text())
			}
		case html.EndTagToken, html.StartTagToken:
			name, _ := z.TagName()
			switch atom.Lookup(name) {
			case atom.H1:
				switch tt {
				case html.StartTagToken:
					inHeading = true
				default:
					return textBuilder.String(), nil
				}
			}
		}
	}
}
