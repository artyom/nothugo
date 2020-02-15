// Command nothugo is a basic static site generator, taking *.md files as its
// input. Its main focus is simplicity and not getting in a way of existing
// file hierarchies.
//
// Usage: nothugo [flags] [mode]
//
// Modes are:
//
// render:
//
// In this mode program recursively walks input directory (-src), renders *.md
// files to HTML, writin output to the output directory (-dst), keeping the
// same file tree structure. Files with names that don't match *.md pattern are
// either hard-linked (if possible), or copied to the destination directory.
// Non-regular files, or files/directories with names starting with "." (unix
// hidden) are skipped.
//
// serve:
//
// In this mode program starts basic HTTP server (-addr) serving static files
// from the output directory (-dst). It is not the only way to serve generated
// content, this can be done with any web server. Most useful for local
// previews.
//
// example:
//
// In this mode program generates example content in the input (-src) and
// templates (-templates) directories. It is best used to create scaffolding
// for a new project or get a sense on how this tool works. As a precaution it
// refuses to overwrite existing files.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

func main() {
	args := runArgs{
		InputDir:     ".",
		OutputDir:    "output",
		TemplatesDir: "templates",
		Addr:         "localhost:8080",
	}
	flag.StringVar(&args.InputDir, "src", args.InputDir, "source directory with .md files")
	flag.StringVar(&args.OutputDir, "dst", args.OutputDir, "destination directory to write rendered files")
	flag.StringVar(&args.TemplatesDir, "templates", args.TemplatesDir, "directory with .html templates")
	flag.StringVar(&args.Addr, "addr", args.Addr, "host:port to listen when run in serve mode")
	flag.Parse()
	log.SetFlags(0)
	var err error
	switch flag.Arg(0) {
	case "serve":
		err = serve(args.Addr, args.OutputDir)
	case "example":
		err = generateExampleContent(args.InputDir, args.TemplatesDir)
	case "render":
		err = run(args)
	default:
		fmt.Fprint(flag.CommandLine.Output(), shortUsage)
		os.Exit(2)
	}
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}

type runArgs struct {
	InputDir     string
	OutputDir    string
	TemplatesDir string
	Addr         string // only for serve
}

func (args *runArgs) validate() error {
	var err error
	if args.InputDir, err = filepath.Abs(args.InputDir); err != nil {
		return err
	}
	if args.OutputDir, err = filepath.Abs(args.OutputDir); err != nil {
		return err
	}
	if args.TemplatesDir, err = filepath.Abs(args.TemplatesDir); err != nil {
		return err
	}
	if args.InputDir == args.OutputDir {
		return errors.New("source and destination directories cannot be the same")
	}
	if args.InputDir == args.TemplatesDir {
		return errors.New("source and templates directories cannot be the same")
	}
	return nil
}

func run(args runArgs) error {
	if err := args.validate(); err != nil {
		return err
	}
	pat := filepath.Join(args.TemplatesDir, "*.html")
	tpl, err := template.ParseGlob(pat)
	if err != nil {
		return fmt.Errorf("parsing templates from %q: %w", args.TemplatesDir, err)
	}
	mtime, err := latestMtime(pat)
	if err != nil {
		return err
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
	convert := func(w io.Writer, r io.Reader) error {
		src, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		return md.Convert(src, w)
	}
	if _, err := exec.LookPath(gfmBinary); err == nil {
		convert = cmarkConvert
	}

	// used to build index.html files. Key is a *destination* directory.
	dirsIndex := make(map[string]struct {
		pages      []pageMeta // pages in this directory
		categories []pageMeta // subdirectories that contain .md files
	})
	// directories that already have "index.html" file in them, to avoid
	// overwriting them with automatically generated index. Key is a
	// *destination* directory.
	skipIndex := make(map[string]struct{})

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		base := filepath.Base(path)
		if info.IsDir() && len(base) > 1 && strings.HasPrefix(base, ".") {
			// skip hidden directories
			return filepath.SkipDir
		}
		if info.IsDir() && (path == args.TemplatesDir || path == args.OutputDir) {
			return filepath.SkipDir
		}
		if !info.Mode().IsRegular() || strings.HasPrefix(base, ".") {
			// skip non-regular or hidden files
			return nil
		}
		rel, err := filepath.Rel(args.InputDir, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(args.OutputDir, rel)

		// in a non-root directory that has some renderable content, mark this
		// directory as a subcategory of its parent
		if dstDir := filepath.Dir(dst); dstDir != args.OutputDir && strings.HasSuffix(path, mdSuffix) {
			key := filepath.Dir(dstDir)
			res := dirsIndex[key]
			dir := filepath.Base(filepath.Dir(dst))
			// to avoid duplicates it's enough to check if the last element
			// matches the one we're about to add because of the way
			// directories are traversed
			if len(res.categories) == 0 || res.categories[len(res.categories)-1].Dst != dir {
				res.categories = append(res.categories, pageMeta{
					Title: fileNameToTitle(dir),
					Dst:   dir,
				})
				dirsIndex[key] = res
			}
		}

		key := filepath.Dir(dst)
		if !strings.HasSuffix(path, mdSuffix) {
			if base == "index.html" {
				skipIndex[key] = struct{}{}
			}
			return copyFile(dst, path)
		}

		title, err := renderFile(tpl, convert, mtime, dst, path)
		if err != nil {
			return err
		}
		res := dirsIndex[key]
		res.pages = append(res.pages, pageMeta{Title: title, Dst: base, src: path})
		dirsIndex[key] = res
		return nil
	}
	if err := filepath.Walk(args.InputDir, walkFunc); err != nil {
		return nil
	}

	for dir, res := range dirsIndex {
		if _, ok := skipIndex[dir]; ok {
			continue
		}
		if err := renderIndex(tpl, convert, dir, res.pages, res.categories); err != nil {
			return err
		}
	}
	return nil
}

// pageMeta is an element of a directory index
type pageMeta struct {
	Title string // page title
	Dst   string // destination file name
	src   string // source file name
}

// copyFile unlinks dst to ensure it does not exist, then tries to create hard
// link from src to dst, if that doesn't succeed, it copies src to dst.
func copyFile(dst, src string) error {
	if src == dst {
		return errors.New("source and destination cannot be the same")
	}
	if fi1, err := os.Stat(src); err == nil {
		if fi2, err := os.Stat(dst); err == nil {
			if os.SameFile(fi1, fi2) {
				return nil
			}
		}
	}
	_ = os.Remove(dst)
	if err := os.MkdirAll(filepath.Dir(dst), 0777); err != nil {
		return err
	}
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.OpenFile(dst, os.O_EXCL|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}
	if err := dstFile.Close(); err != nil {
		return err
	}
	if fi, err := srcFile.Stat(); err == nil {
		mtime := fi.ModTime()
		_ = os.Chtimes(dst, mtime, mtime)
	}
	return nil
}

// renderFile converts Markdown file src into HTML using convert function, then
// renders it to dst file using template tpl. It returns title of rendered page
// and error, if any. After writing to dst, function sets modification time of
// dst either to mtime argument or modification time of src, whichever is most
// recent.
func renderFile(tpl *template.Template, convert convertFunc, mtime time.Time, dst, src string) (string, error) {
	if src == dst {
		return "", errors.New("source and destination cannot be the same")
	}
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer f.Close()
	out := new(bytes.Buffer)
	if err := convert(out, f); err != nil {
		return "", err
	}
	title := fileNameToTitle(filepath.Base(dst))
	if s, err := firstHeading(out.Bytes()); err == nil && s != "" {
		title = s
	}
	page := &Page{
		Title:   title,
		Content: template.HTML(out.Bytes()),
	}
	out.Reset()
	if err := tpl.Execute(out, page); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0777); err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(dst, out.Bytes(), 0666); err != nil {
		return "", err
	}
	if fi, err := f.Stat(); err == nil {
		if m := fi.ModTime(); m.After(mtime) {
			mtime = m
		}
		_ = os.Chtimes(dst, mtime, mtime)
	}
	return title, nil
}

// renderIndex writes index.html file to directory dir. If an element of pages
// describes "README.md" file, this file is rendered using convert function to
// HTML format. This HTML, and every other element from pages is then used to
// render template tpl.
func renderIndex(tpl *template.Template, convert convertFunc, dir string, pages, categories []pageMeta) error {
	var readme template.HTML
	out := new(bytes.Buffer)
	nonReadmePages := make([]pageMeta, 0, len(pages))
	for _, meta := range pages {
		if meta.Dst != "README.md" || readme != "" {
			nonReadmePages = append(nonReadmePages, meta)
			continue
		}
		b, err := ioutil.ReadFile(meta.src)
		if err != nil {
			return err
		}
		if err := convert(out, bytes.NewReader(b)); err != nil {
			return err
		}
		readme = template.HTML(out.Bytes())
	}
	title := fmt.Sprintf("%s index", filepath.Base(dir))
	if readme != "" {
		if s, err := firstHeading([]byte(readme)); err == nil && s != "" {
			title = s
		}
	}
	page := &Page{
		Title:      title,
		Content:    readme,
		Pages:      nonReadmePages,
		Categories: categories,
	}
	out.Reset()
	if err := tpl.Execute(out, page); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(dir, "index.html"), out.Bytes(), 0666)
}

// serve runs HTTP server listening on addr that serves static files from dir
// as a site root.
func serve(addr, dir string) error {
	if addr == "" {
		addr = "localhost:0"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	fileServer := http.FileServer(http.Dir(dir))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; preload")
		}
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "same-origin")
		fileServer.ServeHTTP(w, r)
	})
	log.Printf("serving on http://%s/", ln.Addr())
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return srv.Serve(ln)
}

type Page struct {
	Title      string
	Content    template.HTML
	Pages      []pageMeta // non-empty only for index pages
	Categories []pageMeta // non-empty only for index pages
}

// convertFunc converts Markdown source src to HTML and writes it to dst. HTML
// produced is not a full page code, but a content of block element,
// appropriate to be put into <div>, or <article> element.
type convertFunc func(dst io.Writer, src io.Reader) error

// cmarkConvert is a convertFunc that does text to HTML conversion with an
// external cmark-gfm binary
func cmarkConvert(dst io.Writer, src io.Reader) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, gfmBinary,
		"--validate-utf8",
		"--smart",
		"--github-pre-lang",
		"--strikethrough-double-tilde",
		"-e", "footnotes",
		"-e", "table",
		"-e", "strikethrough",
		"-e", "autolink",
		"-e", "tasklist")
	cmd.Stdin = src
	b, err := cmd.Output()
	if err != nil {
		return err
	}
	if b, err = createAnchors(b, true); err != nil {
		return fmt.Errorf("create anchors on header elements: %w", err)
	}
	_, err = dst.Write(b)
	return err
}

// latestMtime stats each file matching pattern pat and returns the latest
// mtime of them all.
func latestMtime(pat string) (time.Time, error) {
	names, err := filepath.Glob(pat)
	if err != nil {
		return time.Time{}, err
	}
	var mtime time.Time
	for _, name := range names {
		fi, err := os.Stat(name)
		if err != nil {
			return time.Time{}, err
		}
		if m := fi.ModTime(); m.After(mtime) {
			mtime = m
		}
	}
	return mtime, nil
}

func fileNameToTitle(name string) string {
	if strings.ContainsAny(name, " ") {
		return strings.TrimSuffix(name, mdSuffix)
	}
	return repl.Replace(strings.TrimSuffix(name, mdSuffix))
}

var repl = strings.NewReplacer("-", " ")

const mdSuffix = ".md"
const gfmBinary = "cmark-gfm" // https://github.com/github/cmark-gfm binary

const shortUsage = `Usage: nothugo [flags] [mode]
Modes are:

	render  — generate static site
	serve   — start HTTP server for pregenerated site
	example — generate example content

Run with -h flag to see full help text.
`

func init() {
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), usage)
		fmt.Fprintf(flag.CommandLine.Output(), "Flags are:\n\n")
		flag.PrintDefaults()
	}
}

//go:generate usagegen
