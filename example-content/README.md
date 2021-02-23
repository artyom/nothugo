# Layout Example

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