package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/djmarkymark007/chirpy/internal/database"
	"github.com/djmarkymark007/chirpy/internal/validate"
)

var db *database.Database

// TODO(Mark): custom 404 page
func status(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

type returnError struct {
	Error string `json:"error"`
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	ret := returnError{Error: msg}
	data, err := json.Marshal(ret)
	if err != nil {
		log.Printf("Error marshalling JSON: %s\n", err)
		w.WriteHeader(500)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %d\n", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(data)
}

func postUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "failed to decode JSON")
		return
	}

	email, err := db.CreateUser(params.Email)
	if err != nil {
		log.Println(err)
		respondWithError(w, 500, "something went wrong")
	}

	respondWithJson(w, 201, email)
}

func postChirps(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s\n", err)
		respondWithError(w, 400, "Invalid JSON data")
		return
	}
	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is to long")
		return
	}

	chirp, err := db.CreateChirp(validate.ProfaneFilter(params.Body))
	if err != nil {
		log.Println(err)
		respondWithError(w, 500, "something went wrong")
	}

	respondWithJson(w, 201, chirp)
}

func getChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := db.GetChirps()
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, "something went wrong")
	}

	respondWithJson(w, 200, chirps)
}

func getChirp(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("chirpID")
	fmt.Println(path)
	value, err := strconv.Atoi(path)
	if err != nil {
		respondWithError(w, 500, "could not convert to int")
		return
	}

	chirps, err := db.GetChirps()
	if err != nil {
		respondWithError(w, 500, "Failed to load chirps from database")
		return
	}
	if value > len(chirps) {
		respondWithError(w, 404, "chirp doesn't exist")
		return
	}

	var loc int = -1
	for index, chirp := range chirps {
		if chirp.Id == value {
			loc = index
		}
	}

	respondWithJson(w, 200, chirps[loc])
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

const metricsMsg string = `<html>

<body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
</body>

</html>
`

func (cfg *apiConfig) metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	msg := fmt.Sprintf(metricsMsg, cfg.fileserverHits)
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
	const path = "database.json"

	dbg := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()
	if *dbg {
		os.Remove(path)
	}

	var err error
	db, err = database.NewDB(path)
	if err != nil {
		log.Fatal(err)
	}

	config := apiConfig{fileserverHits: 0}

	serverHandler := http.NewServeMux()
	serverHandler.Handle("/app/*", http.StripPrefix("/app", middlewareLog(config.middlewareMetricsInc(http.FileServer(http.Dir("."))))))
	serverHandler.HandleFunc("GET /admin/metrics", config.metrics)
	serverHandler.HandleFunc("GET /api/reset", config.reset)
	serverHandler.HandleFunc("GET /api/healthz", status)
	serverHandler.HandleFunc("GET /api/chirps", getChirps)
	serverHandler.HandleFunc("POST /api/chirps", postChirps)
	serverHandler.HandleFunc("GET /api/chirps/{chirpID}", getChirp)
	serverHandler.HandleFunc("POST /api/users", postUsers)

	server := http.Server{Handler: serverHandler, Addr: ":" + port}

	fmt.Print("starting server\n")
	fmt.Printf("servering files from: %s. on port: %s\n", filepathRoot, port)
	server.ListenAndServe()
}
