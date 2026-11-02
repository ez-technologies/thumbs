// Command upload is an HTTP server that accepts an image upload and stores the original next to its thumbnail:
// "<name>" and "<name>.thumb.jpg".
//
//	go run ./examples/upload -dir ./uploads -addr :8080
//	curl -F file=@photo.jpg http://localhost:8080/upload
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ez-technologies/thumbs"
)

// maxUpload is the largest upload accepted, in bytes.
const maxUpload = 20 << 20

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	dir := flag.String("dir", "uploads", "directory to store originals and thumbnails in")
	flag.Parse()
	if err := os.MkdirAll(*dir, 0o755); err != nil {
		log.Fatal(err)
	}
	http.Handle("POST /upload", uploadHandler(*dir))
	log.Printf("listening on %s, storing in %s", *addr, *dir)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

// uploadHandler stores the multipart field "file" in dir, with its thumbnail beside it, and answers with both names.
func uploadHandler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "expected a multipart field named file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "upload too large or cut short", http.StatusRequestEntityTooLarge)
			return
		}

		// Make the thumbnail first: it also checks that the upload is a picture.
		thumb, err := thumbs.Make(bytes.NewReader(data))
		if err != nil {
			http.Error(w, "not a JPEG, PNG or WebP image", http.StatusUnsupportedMediaType)
			return
		}

		name := filepath.Base(header.Filename) // never trust a path from the client
		if name == "." || name == string(filepath.Separator) {
			http.Error(w, "missing file name", http.StatusBadRequest)
			return
		}
		original := filepath.Join(dir, name)
		if err := os.WriteFile(original, data, 0o644); err != nil {
			http.Error(w, "could not store the original", http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(original+thumbs.Suffix, thumb, 0o644); err != nil {
			http.Error(w, "could not store the thumbnail", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"original":        name,
			"original_bytes":  len(data),
			"thumbnail":       name + thumbs.Suffix,
			"thumbnail_bytes": len(thumb),
		})
	})
}
