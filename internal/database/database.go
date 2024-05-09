package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

type User struct {
	Email string `json:"email"`
	Id    int    `json:"id"`
}

type Database struct {
	path string
	mu   *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp `json:"chirps"`
	Users  map[int]User  `json:"users"`
}

func (dbs *DBStructure) len() int {
	return len(dbs.Chirps) + len(dbs.Users)
}

func NewDB(path string) (*Database, error) {
	db := Database{path: path, mu: &sync.RWMutex{}}
	err := db.ensureDB()
	if err != nil {
		return &Database{}, err
	}

	return &db, nil
}

func (db *Database) ensureDB() error {
	f, err := os.Open(db.path)
	if err != nil {
		if os.IsNotExist(err) {
			f, err := os.Create(db.path)
			if err != nil {
				return err
			}
			f.Close()
			return nil
		} else {
			return err
		}
	}
	f.Close()
	return nil
}

func (db *Database) CreateChirp(body string) (Chirp, error) {
	data, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}
	newChirp := Chirp{Id: data.len() + 1, Body: body}
	data.Chirps[data.len()] = newChirp
	err = db.writeDB(data)
	if err != nil {
		return Chirp{}, err
	}

	return newChirp, nil
}

func (db *Database) CreateUser(email string) (User, error) {
	data, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	newUser := User{Id: data.len() + 1, Email: email}
	data.Users[data.len()] = newUser
	err = db.writeDB(data)
	if err != nil {
		return User{}, err
	}

	return newUser, nil
}

func (db *Database) GetChirps() ([]Chirp, error) {
	data, err := db.loadDB()
	if err != nil {
		return []Chirp{}, err
	}
	var result []Chirp

	for _, value := range data.Chirps {
		result = append(result, value)
	}

	return result, nil
}

func (db *Database) loadDB() (DBStructure, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.ensureDB()

	data, err := os.ReadFile(db.path)
	if err != nil {
		return DBStructure{}, fmt.Errorf("error loading database. path: %s, error: %s", db.path, err)
	}

	result := DBStructure{Chirps: make(map[int]Chirp), Users: make(map[int]User)}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return DBStructure{}, errors.New("failed to decode json data")
	}

	return result, nil
}

func (db *Database) writeDB(dbstructure DBStructure) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.ensureDB()

	data, err := json.Marshal(dbstructure)
	if err != nil {
		return err
	}
	os.WriteFile(db.path, data, 0666)
	return nil
}
