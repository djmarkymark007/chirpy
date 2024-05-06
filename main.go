package main

import (
	"fmt"
	"net/http"
)

// TODO(Mark): custom 404 page
func status(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

type apiConfig struct {
	fileserverHits int
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func middlewareLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%s %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	msg := fmt.Sprintf("Hits: %d", cfg.fileserverHits)
	w.Write([]byte(msg))
}

func (cfg *apiConfig) reset(w http.ResponseWriter, _ *http.Request) {
	cfg.fileserverHits = 0
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}

func main() {
	const port = "8080"
	const filepathRoot = "."

	config := apiConfig{fileserverHits: 0}

	serverHandler := http.NewServeMux()
	serverHandler.Handle("/app/*", http.StripPrefix("/app", middlewareLog(config.middlewareMetricsInc(http.FileServer(http.Dir("."))))))
	serverHandler.HandleFunc("GET /metrics", config.metrics)
	serverHandler.HandleFunc("/reset", config.reset)
	serverHandler.HandleFunc("GET /healthz", status)

	server := http.Server{Handler: serverHandler, Addr: ":" + port}

	fmt.Print("starting server\n")
	fmt.Printf("servering files from: %s. on port: %s\n", filepathRoot, port)
	server.ListenAndServe()
}
