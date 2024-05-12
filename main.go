package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"github.com/djmarkymark007/chirpy/internal/authorize"
	"github.com/djmarkymark007/chirpy/internal/database"
	"github.com/djmarkymark007/chirpy/internal/validate"
)

var db *database.Database
var config apiConfig

const InternalErrorMsg = "Something went wrong"

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

func updateUser(w http.ResponseWriter, r *http.Request) {
	log.Print("--- updateUser ---")
	params := User{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	token := getTokenFromHeader(r)
	id, err := authorize.GetIdFromJwt(token, config.jwtSecret)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	user, err := db.GetUserById(id)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	user.Email = params.Email
	user.PasswordHash = passwordHash

	err = db.UpdateUser(user)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	respondWithJson(w, 200, database.User{Id: id, Email: params.Email})
}

func getTokenFromHeader(r *http.Request) string {
	authorization := r.Header.Get("Authorization")
	authorizationParts := strings.Split(authorization, " ")
	token := authorizationParts[len(authorizationParts)-1]
	return token
}

func postLogin(w http.ResponseWriter, r *http.Request) {
	log.Print("--- postLogin ---")
	type parameters struct {
		Password         string `json:"password"`
		Email            string `json:"email"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	user, err := db.GetUser(params.Email)
	if err != nil {
		log.Printf("postLogin: %s\n", err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	if bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(params.Password)) != nil {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	jwtToken, err := authorize.CreateJwt(user.Id, params.ExpiresInSeconds, config.jwtSecret)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	refreshToken, err := authorize.CreateRefreshToken()
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	user.RefreshToken = refreshToken
	user.TokenExpiresAt = time.Now().UTC().Add(60 * 24 * time.Hour)
	err = db.UpdateUser(user)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	type UserWithjwt struct {
		Id           int    `json:"id"`
		Email        string `json:"email"`
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	respondWithJson(w, 200, UserWithjwt{Id: user.Id, Email: user.Email, Token: jwtToken, RefreshToken: refreshToken})
}

func refreshJWT(w http.ResponseWriter, r *http.Request) {
	log.Print("--- refreshJWT ---")
	token := getTokenFromHeader(r)

	validToken, currentUser, err := authorize.ValidateRefreshToken(token, db)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	jwtToken := ""
	if validToken {
		jwtToken, err = authorize.CreateJwt(currentUser.Id, 0, config.jwtSecret)
		if err != nil {
			log.Print(err)
			respondWithError(w, 500, InternalErrorMsg)
			return
		}
	} else {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	type TokenType struct {
		JwtToken string `json:"token"`
	}

	respondWithJson(w, 200, TokenType{JwtToken: jwtToken})
}

func revokeToken(w http.ResponseWriter, r *http.Request) {
	log.Print("--- revokeToken ---")

	token := getTokenFromHeader(r)
	users, err := db.GetUsers()
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	valid := false
	currentUser := database.UserDatabase{}
	for _, user := range users {
		if user.RefreshToken == token {
			currentUser = user
			valid = true
			break
		}
	}

	if !valid {
		respondWithError(w, 401, "Unauthorized")
		return
	}

	currentUser.RefreshToken = ""
	currentUser.TokenExpiresAt = time.Time{}
	err = db.UpdateUser(currentUser)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	respondWithJson(w, 204, "")
}

// TODO(Mark): Not sure if i like this
type User struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

func postUsers(w http.ResponseWriter, r *http.Request) {
	log.Print("--- postUsers ---")
	params := User{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	alreadyExist, err := db.UserExist(params.Email)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	if alreadyExist {
		respondWithError(w, 401, "user email already used")
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}
	email, err := db.CreateUser(params.Email, passwordHash)
	if err != nil {
		log.Println(err)
		respondWithError(w, 500, InternalErrorMsg)
	}

	respondWithJson(w, 201, email)
}

func postChirps(w http.ResponseWriter, r *http.Request) {
	log.Print("--- postChirps ---")

	token := getTokenFromHeader(r)
	if token == "" {
		respondWithError(w, 401, "Unauthorized")
	}

	userId, err := authorize.GetIdFromJwt(token, config.jwtSecret)
	if err != nil {
		log.Print(err)
		respondWithError(w, 401, "Unauthorized")
	}

	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s\n", err)
		respondWithError(w, 400, "Invalid JSON data")
		return
	}
	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is to long")
		return
	}

	chirp := database.Chirp{Id: 0, Body: validate.ProfaneFilter(params.Body), AuthorId: userId}
	chirp, err = db.CreateChirp(chirp)
	if err != nil {
		log.Println(err)
		respondWithError(w, 500, InternalErrorMsg)
	}

	respondWithJson(w, 201, chirp)
}

func getChirps(w http.ResponseWriter, r *http.Request) {
	log.Print("--- getChirps ---")
	chirps, err := db.GetChirps()
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
	}

	respondWithJson(w, 200, chirps)
}

func deleteChirp(w http.ResponseWriter, r *http.Request) {
	token := getTokenFromHeader(r)
	userId, err := authorize.GetIdFromJwt(token, config.jwtSecret)
	if err != nil {
		log.Print(err)
		respondWithError(w, 403, "Unauthorized")
		return
	}

	path := r.PathValue("chirpID")
	chirpId, err := strconv.Atoi(path)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	chirp, err := db.GetChirpById(chirpId)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	if chirp.AuthorId != userId {
		log.Printf("chrip author id: %v, user id: %v", chirp.AuthorId, userId)
		respondWithError(w, 403, "Unauthorized")
		return
	}

	err = db.DeleteChirp(chirp.Id)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	respondWithJson(w, 204, "")
}

func getChirp(w http.ResponseWriter, r *http.Request) {
	log.Print("--- getChirp ---")

	path := r.PathValue("chirpID")
	value, err := strconv.Atoi(path)
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
		return
	}

	chirps, err := db.GetChirps()
	if err != nil {
		log.Print(err)
		respondWithError(w, 500, InternalErrorMsg)
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
	jwtSecret      string
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
	var err error

	err = godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %s", err)
	}

	config = apiConfig{fileserverHits: 0, jwtSecret: os.Getenv("JWT_SECRET")}

	dbg := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()
	if *dbg {
		os.Remove(path)
	}

	db, err = database.NewDB(path)
	if err != nil {
		log.Fatal(err)
	}

	serverHandler := http.NewServeMux()
	serverHandler.Handle("/app/*", http.StripPrefix("/app", middlewareLog(config.middlewareMetricsInc(http.FileServer(http.Dir("."))))))
	serverHandler.HandleFunc("GET /admin/metrics", config.metrics)
	serverHandler.HandleFunc("GET /api/reset", config.reset)
	serverHandler.HandleFunc("GET /api/healthz", status)
	serverHandler.HandleFunc("GET /api/chirps", getChirps)
	serverHandler.HandleFunc("POST /api/chirps", postChirps)
	serverHandler.HandleFunc("GET /api/chirps/{chirpID}", getChirp)
	serverHandler.HandleFunc("POST /api/users", postUsers)
	serverHandler.HandleFunc("POST /api/login", postLogin)
	serverHandler.HandleFunc("PUT /api/users", updateUser)
	serverHandler.HandleFunc("POST /api/refresh", refreshJWT)
	serverHandler.HandleFunc("POST /api/revoke", revokeToken)
	serverHandler.HandleFunc("DELETE /api/chirps/{chirpID}", deleteChirp)

	server := http.Server{Handler: serverHandler, Addr: ":" + port}

	fmt.Print("starting server\n")
	fmt.Printf("servering files from: %s. on port: %s\n", filepathRoot, port)
	server.ListenAndServe()
}
