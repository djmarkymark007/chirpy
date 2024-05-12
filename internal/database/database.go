package database

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

type Chirp struct {
	Id       int    `json:"id"`
	Body     string `json:"body"`
	AuthorId int    `json:"author_id"`
}

type User struct {
	Email string `json:"email"`
	Id    int    `json:"id"`
}

type UserDatabase struct {
	Email          string    `json:"email"`
	Id             int       `json:"id"`
	PasswordHash   []byte    `json:"password_hash"`
	RefreshToken   string    `json:"refresh_token"`
	TokenExpiresAt time.Time `json:"token_expires_at"`
}

type Database struct {
	path string
	mu   *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]Chirp        `json:"chirps"`
	Users  map[int]UserDatabase `json:"users"`
}

// NOTE(Mark): not sure if this is need
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

func (db *Database) CreateChirp(chirp Chirp) (Chirp, error) {
	data, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}
	chirp.Id = len(data.Chirps) + 1
	data.Chirps[len(data.Chirps)] = chirp
	err = db.writeDB(data)
	if err != nil {
		return Chirp{}, err
	}

	return chirp, nil
}

func (db *Database) UpdateUser(userChange UserDatabase) error {
	data, err := db.loadDB()
	if err != nil {
		return err
	}

	data.Users[userChange.Id-1] = userChange

	err = db.writeDB(data)
	if err != nil {
		return err
	}
	return nil
}

func (db *Database) DeleteChirp(chripId int) error {
	data, err := db.loadDB()
	if err != nil {
		return err
	}

	delete(data.Chirps, chripId-1)

	for index, chirp := range data.Chirps {
		if chirp.Id < chripId {
			chripId--
			data.Chirps[index] = chirp
		}
	}

	err = db.writeDB(data)
	if err != nil {
		return err
	}

	return nil
}

func (db *Database) GetChirpById(id int) (Chirp, error) {
	foundChirp := Chirp{}
	chirps, err := db.GetChirps()
	if err != nil {
		return foundChirp, err
	}

	for _, chirp := range chirps {
		if chirp.Id == id {
			foundChirp = chirp
			break
		}
	}

	return foundChirp, nil
}

func (db *Database) CreateUser(email string, passwordHash []byte) (User, error) {
	data, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	newUser := UserDatabase{Id: len(data.Users) + 1, Email: email, PasswordHash: passwordHash}
	data.Users[len(data.Users)] = newUser
	err = db.writeDB(data)
	if err != nil {
		return User{}, err
	}

	//NOTE(Mark): i don' t like having to struct on for the database and on for the return
	return User{Id: newUser.Id, Email: newUser.Email}, nil
}

func (db *Database) UserExist(email string) (bool, error) {
	user, err := db.GetUser(email)
	if err != nil {
		return false, err
	}

	if user.Email != email {
		return false, nil
	}

	return true, nil
}

func (db *Database) GetUser(email string) (UserDatabase, error) {
	users, err := db.GetUsers()
	if err != nil {
		return UserDatabase{}, fmt.Errorf("failed to get users to check if a user exist. %s", err)
	}

	for _, value := range users {
		if email == value.Email {
			return value, nil
		}
	}

	return UserDatabase{}, nil
}

func (db *Database) GetUserById(id int) (UserDatabase, error) {
	users, err := db.GetUsers()
	if err != nil {
		return UserDatabase{}, fmt.Errorf("failed to get users to check if a user exist. %s", err)
	}

	for _, value := range users {
		if id == value.Id {
			return value, nil
		}
	}

	return UserDatabase{}, nil
}

func (db *Database) GetUsers() ([]UserDatabase, error) {
	data, err := db.loadDB()
	if err != nil {
		return []UserDatabase{}, err
	}
	var result []UserDatabase

	for _, value := range data.Users {
		result = append(result, value)
	}

	return result, nil
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

	result := DBStructure{Chirps: make(map[int]Chirp), Users: make(map[int]UserDatabase)}

	db.ensureDB()

	data, err := os.ReadFile(db.path)
	if err != nil {
		return DBStructure{}, fmt.Errorf("error loading database. path: %s, error: %s", db.path, err)
	}

	if len(data) == 0 {
		return result, nil
	}

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
