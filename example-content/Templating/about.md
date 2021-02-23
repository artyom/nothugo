# Template rendering

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