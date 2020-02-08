package main

import (
	"bytes"
	"fmt"
	"strings"

	anchor "github.com/shurcooL/sanitized_anchor_name"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// createAnchors takes utf-8 HTML data, parses it as a content of an <article>
// element, walks over resulting tree and sets slugified unique id attribute
// for each h1..h6 element it finds. It then renders such HTML subtree and
// returns result. If reuse is true, input slice b is reused for rendering.
func createAnchors(b []byte, reuse bool) ([]byte, error) {
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
	seen := map[string]struct{}{}
	var walkFn func(*html.Node)
	walkFn = func(n *html.Node) {
		var isHeader bool
		if n.Type == html.ElementNode {
			switch n.DataAtom {
			case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
				isHeader = true
			}
		}
		if !isHeader {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walkFn(c)
			}
			return
		}
		// header node: it's ok to not recursively traverse children here, as
		// headers cannot be nested
		title := nodeText(n)

		slug := anchor.Create(title)
		if _, ok := seen[slug]; !ok {
			seen[slug] = struct{}{}
		} else {
			for i := 1; i < 100; i++ {
				s := fmt.Sprintf("%s-%d", slug, i)
				if _, ok := seen[s]; !ok {
					slug = s
					seen[slug] = struct{}{}
					break
				}
			}
		}
		for i, attr := range n.Attr {
			if attr.Key == "id" {
				n.Attr[i].Val = slug
				return
			}
		}
		n.Attr = append(n.Attr, html.Attribute{Key: "id", Val: slug})
	}
	walkFn(root)

	var out *bytes.Buffer
	if reuse {
		out = bytes.NewBuffer(b[:0])
	} else {
		out = new(bytes.Buffer)
	}
	for c := root.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(out, c); err != nil {
			return nil, err
		}
	}
	return out.Bytes(), nil
}

// nodeText returns text extracted from note and all its descendants
func nodeText(n *html.Node) string {
	var b strings.Builder
	var fn func(*html.Node)
	fn = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			fn(c)
		}
	}
	fn(n)
	return b.String()
}
