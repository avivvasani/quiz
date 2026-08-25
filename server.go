// quizodex is a tiny static file server for the quiz web-page.
// It behaves like `python3 -m http.server 8000`: it serves everything in the
// current directory (new_quiz.html, questions.json, media, etc.) with proper
// MIME types, directory listings, and HTTP range/HEAD support.
//
// Usage:
//
//	go run server.go                 # serves . on :8000
//	go run server.go -port 8080      # another port
//	go run server.go -root ../quiz   # another directory
//
// Build a standalone binary:
//
//	go build -o quizserver server.go
//	./quizserver -port 8000
package main

import (
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var mimeTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".htm":   "text/html; charset=utf-8",
	".json":  "application/json; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".mjs":   "text/javascript; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".txt":   "text/plain; charset=utf-8",
	".xml":   "text/xml; charset=utf-8",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".svg":   "image/svg+xml",
	".webp":  "image/webp",
	".avif":  "image/avif",
	".ico":   "image/x-icon",
	".mp4":   "video/mp4",
	".webm":  "video/webm",
	".ogg":   "video/ogg",
	".ogv":   "video/ogg",
	".mp3":   "audio/mpeg",
	".wav":   "audio/wav",
	".m4a":   "audio/mp4",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".otf":   "font/otf",
	".pdf":   "application/pdf",
}

// indexFiles are tried, in order, when a directory is requested.
var indexFiles = []string{"index.html", "new_quiz.html"}

func contentTypeFor(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ct, ok := mimeTypes[ext]; ok {
		return ct
	}
	return ""
}

// serveFile hands the file to net/http (which supports HEAD and Range
// requests), using our MIME mapping for correct types.
func serveFile(w http.ResponseWriter, r *http.Request, full string) {
	if ct := contentTypeFor(full); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	http.ServeFile(w, r, full)
}

// renderListing shows a basic directory listing like python's http.server.
func renderListing(w http.ResponseWriter, r *http.Request, dir, urlPath string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, "cannot read directory: "+err.Error(), http.StatusInternalServerError)
		return
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html>\n<head><meta charset=\"utf-8\">")
	b.WriteString("<title>Directory listing for " + html.EscapeString(urlPath) + "</title></head>\n<body>\n")
	b.WriteString("<h1>Directory listing for " + html.EscapeString(urlPath) + "</h1>\n<hr>\n<ul>\n")

	parent := strings.TrimSuffix(urlPath, "/")
	if parent != "" && parent != "/" {
		if i := strings.LastIndex(parent, "/"); i >= 0 {
			b.WriteString("<li><a href=\"" + html.EscapeString(parent[:i+1]) + "\">../</a></li>\n")
		}
	}
	for _, e := range entries {
		name := e.Name()
		href := urlPath + "/" + name
		if !strings.HasSuffix(href, "/") {
			href += "/"
		}
		if e.IsDir() {
			b.WriteString("<li><a href=\"" + html.EscapeString(href) + "\">" + html.EscapeString(name) + "/</a></li>\n")
		} else {
			b.WriteString("<li><a href=\"" + html.EscapeString(href) + "\">" + html.EscapeString(name) + "</a></li>\n")
		}
	}
	b.WriteString("</ul>\n</body>\n</html>\n")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, b.String())
}

func handler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean("/" + r.URL.Path)
		full := filepath.Join(root, clean)

		// Guard against path traversal outside the root.
		rel, err := filepath.Rel(root, full)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			http.NotFound(w, r)
			return
		}

		info, err := os.Stat(full)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if info.IsDir() {
			for _, idx := range indexFiles {
				candidate := filepath.Join(full, idx)
				if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
					serveFile(w, r, candidate)
					return
				}
			}
			renderListing(w, r, full, clean)
			return
		}

		serveFile(w, r, full)
	}
}

func main() {
	port := flag.String("port", "8000", "port to listen on")
	root := flag.String("root", ".", "directory to serve")
	flag.Parse()

	abs, err := filepath.Abs(*root)
	if err != nil {
		log.Fatalf("invalid root: %v", err)
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		log.Fatalf("root is not a directory: %s", abs)
	}

	addr := ":" + *port
	fmt.Printf("Serving %s on http://localhost:%s\n", abs, addr)
	fmt.Println("Open http://localhost:" + *port + "/new_quiz.html  (Ctrl+C to stop)")

	http.Handle("/", handler(abs))
	log.Fatal(http.ListenAndServe(addr, nil))
}
