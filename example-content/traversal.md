# Filesystem Traversal Details

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

If you prefer to have resulting rendered files to have .html suffix instead of
.md, then run render subcommand with -html flag. In this case the above example
will have the following layout after rendering:

    index.html
    README.html
    traversal.html
    Templating/
    index.html
        about.html
    media/
        logo.png