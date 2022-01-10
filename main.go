package main

import (
	"embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// 100 megabytes
const MAX_UPLOAD_SIZE = 100 << 20

//go:embed templates
var templatesFolder embed.FS

var supportedFiles = map[string]string{
	"audio/basic":                   "audio",
	"audio/mpeg":                    "audio",
	"audio/mp3":                     "audio",
	"audio/ogg":                     "audio",
	"audio/wav":                     "audio",
	"audio/wave":                    "audio",
	"audio/avi":                     "audio",
	"audio/midi":                    "audio",
	"image/jpeg":                    "pictures",
	"image/png":                     "pictures",
	"image/gif":                     "pictures",
	"image/bmp":                     "pictures",
	"image/tiff":                    "pictures",
	"image/webp":                    "pictures",
	"image/svg":                     "pictures",
	"video/mp4":                     "video",
	"video/ogg":                     "video",
	"video/webm":                    "video",
	"video/avi":                     "video",
	"video/mpeg":                    "video",
	"video/quicktime":               "video",
	"application/octet-stream":      "documents",
	"application/pdf":               "documents",
	"application/zip":               "documents",
	"text/plain; charset=utf-8":     "documents",
	"text/html; charset=utf-8":      "documents",
	"text/css":                      "documents",
	"text/javascript":               "documents",
	"application/json":              "documents",
	"application/msword":            "documents",
	"application/vnd.ms-excel":      "documents",
	"application/vnd.ms-powerpoint": "documents",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "documents",
}

func index(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	tmpl, err := template.ParseFS(templatesFolder, "templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	files := r.MultipartForm.File["file"]
	for _, fileHeader := range files {
		if fileHeader.Size > MAX_UPLOAD_SIZE {
			http.Error(w, fmt.Sprintf("The uploaded file is too big: %s. Please use an fle less than 100MB in size", fileHeader.Filename), http.StatusBadRequest)
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		buf := make([]byte, 512)
		_, err = file.Read(buf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		filetype := http.DetectContentType(buf)
		ft, ok := supportedFiles[filetype]
		if !ok {
			http.Error(w, "The provided file format is not allowed.", http.StatusBadRequest)
			return
		}

		_, err = file.Seek(0, io.SeekStart)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(fileHeader.Filename))
		createFile := fmt.Sprintf("./uploads/%s/%s", ft, filename)

		f, err := os.Create(createFile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer f.Close()
	}

	fmt.Fprintf(w, "Upload successful")
}

func main() {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Add("Pragma", "no-cache")
			w.Header().Add("Expires", "0")
			next.ServeHTTP(w, r)
		})
	})

	r.Get("/", index)
	r.Post("/upload", upload)
	r.Get("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads/"))).ServeHTTP)

	log.Println("Listening on :3030")
	http.ListenAndServe(":3030", r)
}
