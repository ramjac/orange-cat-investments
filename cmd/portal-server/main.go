package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	dir := os.Getenv("STATIC_DIR")
	if dir == "" {
		dir = "/dist"
	}

	fs := http.FileServer(http.Dir(dir))
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	http.Handle("/", fs)

	log.Printf("Portal server listening on :%s serving %s", port, dir)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
