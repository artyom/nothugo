package main

import (
	"errors"
	"os"
	"path/filepath"
)

func generateExampleContent(rootDir, templatesDir string) error {
	if rootDir == templatesDir {
		return errors.New("source and templates directories cannot be the same")
	}
	exampleFiles := []struct {
		path string
		body []byte
	}{
		{filepath.Join(rootDir, "README.md"), []byte(rootReadme)},
		{filepath.Join(rootDir, "traversal.md"), []byte(traversalHelp)},
		{filepath.Join(rootDir, "Templating", "about.md"), []byte(templatingHelp)},
		{filepath.Join(templatesDir, "default.html"), []byte(exampleTemplate)},
	}
	for _, rec := range exampleFiles {
		if err := writeIfNotExists(rec.path, rec.body); err != nil {
			return err
		}
	}
	return nil
}

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

// README.md
const rootReadme = `# Layout Example

This file is the top-level "index" file.

When a directory contains any *.md files and does not hold an index.html file,
nothugo automatically generates an index.html file. It creates such an index by
listing all *.md files under this directory and all subdirectories having at
least one *.md file. If a directory contains a "README.md" file, such file is
used as an index page content and can be rendered by template along with the
index of child resources. If the README.md file exists, it won't be present in
the list of child resources.

If the program finds a [cmark-gfm](https://github.com/github/cmark-gfm)
executable in the PATH environment, it uses it to render Markdown files;
otherwise, it uses a [built-in
library](https://pkg.go.dev/github.com/yuin/goldmark?tab=overview).
`

// traversal.md
const traversalHelp = `# Filesystem Traversal Details

When nothugo traverses the source directory (-src flag), it only processes
regular files and directories, skipping everything else and paths with names
starting with "." (Unix hidden). It renders all *.md files to HTML. For other
files, it either creates hard links for them at the destination or, if
hard-linking fails, copies them.

The assumption is that that destination directory (-dst flag) can be removed
and generated from scratch. You should avoid editing files there directly — it
may even be dangerous if you change a file hard-linked from its origin.

Consider that the source directory (-src) represents the following filesystem
tree:

    README.md
    traversal.md
    .gitignore
    .git/
        ...
    Templating/
        about.md
    media/
        logo.png

After rendering such directory, the destination will look like this:

    index.html
    README.md
    traversal.md
    Templating/
        index.html
        about.md
    media/
        logo.png

In this directory, the top-level "index.html" file contains rendered content of
the "README.md" file, and links to two child resources: "Templating"
subcategory, and "traversal.md" page. File "Templating/index.html" contains the
single link to its "about.md" child page.

Notice that all files at the destination keep their original names, even though
files with .md suffix are now actually HTML documents. It enables having
relative cross-document links in existing Markdown document corpus and is
compatible with how Github renders a preview of .md files.
`

// Template/about.md
const templatingHelp = `# Template rendering

For templating Go's "html/template" package is used. See
<https://golang.org/pkg/html/template/> and
<https://golang.org/pkg/text/template/> for more details on how templates work.

Templates are read from templates directory (-templates flag) as one or more
*.html files with
[template.ParseGlob](https://golang.org/pkg/text/template/#Template.ParseGlob)
function.

Template is rendered with the Page object:

	type Page struct {
		Title      string        // page title, as: <title>{{.Title}}</title>
		Content    template.HTML // page content, rendered as HTML
		Pages      []pageMeta    // non-empty only for index pages
		Categories []pageMeta    // non-empty only for index pages
	}

	// pageMeta is an immediate child of the section. It either points to a
	// *.md file, or a subdirectory that contains at least one *.md file.
	type pageMeta struct {
		Title string // page title, as: <a ...>{{.Title}}</a>
		Dst   string // destination file/directory name, as: <a href="{{.Dst}}">
	}

When rendering pages or creating an automated index, the document name is
discovered from the first \<h1\> element, or derived from the file name.	
`

// templates/default.html
const exampleTemplate = `<!DOCTYPE html><head><meta charset="utf-8"><title>{{ .Title }}</title>
<style>
body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
    font-size: 1rem;
    line-height: 170%;
    max-width: 45em;
    margin: auto;
    padding-right: 1em;
    padding-left: 1em;
}
</style>
</head>
<body>{{ .Content }}

{{/*
For templating Go's "html/template" package is used. See
https://golang.org/pkg/html/template/ and https://golang.org/pkg/text/template/
for more details on how templates work.

Templates are read from this directory with template.ParseGlob [1] function,
pattern is "*.html".

Template is rendered with the Page object:

	type Page struct {
		Title      string        // page title, as: <title>{{.Title}}</title>
		Content    template.HTML // page content, rendered as HTML
		Pages      []pageMeta    // non-empty only for index pages
		Categories []pageMeta    // non-empty only for index pages
	}

	// pageMeta is an immediate child of the section. It either points to a
	// *.md file, or a subdirectory that contains at least one *.md file.
	type pageMeta struct {
		Title string // page title, as: <a ...>{{.Title}}</a>
		Dst   string // destination file/directory name, as: <a href="{{.Dst}}">
	}

[1]: https://golang.org/pkg/text/template/#Template.ParseGlob
*/}}

{{if .Content}}{{if or .Pages .Categories}}<hr>{{end}}{{end}}
{{if .Pages }}<p>Pages in this category</p><ul>{{ range .Pages }}
    <li><a href="{{.Dst}}">{{.Title}}</a></li>{{end}}</ul>
{{end}}
{{if .Categories}}<p>Subcategories:</p><ul>{{range .Categories}}
    <li><a href="{{.Dst}}">{{.Title}}</a></li>{{end}}</ul>
{{end}}
</body>
`
