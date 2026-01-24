package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	port := "3000"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	fs := http.FileServer(http.Dir(dir))
	http.Handle("/", fs)

	log.Printf("Serving docs at http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
