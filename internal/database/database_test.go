package database_test

import (
	"os"
	"reflect"
	"testing"

	"github.com/djmarkymark007/chirpy/internal/database"
)

func TestDatabase(t *testing.T) {
	const path = "./testDatabase.json"
	os.Remove(path)

	chirp1 := database.Chirp{Body: "this is a chirp", Id: 0, AuthorId: 1}
	chirp2 := database.Chirp{Body: "this is another chirp", Id: 0, AuthorId: 2}

	db, err := database.NewDB(path)
	if err != nil {
		t.Fatal(err)
	}

	db.CreateChirp(chirp1)
	db.CreateChirp(chirp2)
	got, err := db.GetChirps()
	if err != nil {
		t.Fatal(err)
	}

	want := []database.Chirp{
		{Id: 1, Body: "this is a chirp", AuthorId: 1},
		{Id: 2, Body: "this is another chirp", AuthorId: 2},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("value mis match: got %v, want %v\n", got, want)
	}

}
