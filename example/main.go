package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"

	"github.com/pthm/hxcmp"
	"github.com/pthm/hxcmp/example/components"
)

//go:embed static
var staticFiles embed.FS

func main() {
	// Create store
	store := NewStore()

	// Create registry with encryption key (in production, use a real secret)
	key := []byte("example-key-must-be-32-bytes!!")
	reg := hxcmp.NewRegistry(key)
	hxcmp.SetDefault(reg)

	// Initialize all components
	components.Init(store, reg)

	// Create router
	mux := http.NewServeMux()

	// Component routes
	mux.Handle("/_c/", reg.Handler())

	// Page routes
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/task/{id}", handleTaskDetail)

	// Static files
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Start server
	addr := ":8080"
	fmt.Printf("Starting server at http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	// Read filter state from URL - this is the source of truth
	status := r.URL.Query().Get("status")
	hxcmp.Render(w, r, Layout(status))
}

func handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	hxcmp.Render(w, r, TaskDetailLayout(id))
}
