package main

import (
	"bytes"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// rewriteLinks takes utf-8 HTML data, parses it as a content of an <article>
// element, and rewrites any non-absolute links with .md suffix to have .html
// suffix, returning resulting html back.
func rewriteLinks(b []byte) ([]byte, error) {
	if !bytes.Contains(b, []byte(mdSuffix)) {
		return b, nil
	}
	root := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Article,
		Data:     atom.Article.String(),
	}
	nodes, err := html.ParseFragment(bytes.NewReader(b), root)
	if err != nil {
		return nil, err
	}
	for _, node := range nodes {
		root.AppendChild(node)
	}
	var walkFn func(*html.Node)
	walkFn = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.A {
			for i, attr := range n.Attr {
				if attr.Key == "href" {
					u, err := url.Parse(attr.Val)
					if err != nil || u.Scheme != "" || u.Host != "" || !strings.HasSuffix(u.Path, mdSuffix) {
						break
					}
					u.Path = strings.TrimSuffix(u.Path, mdSuffix) + htmlSuffix
					n.Attr[i].Val = u.String()
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkFn(c)
		}
	}
	walkFn(root)
	out := new(bytes.Buffer)
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(out, c); err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}
