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

	chirp1 := "this is a chirp"
	chirp2 := "this is another chirp"

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
		{Id: 0, Body: "this is a chirp"},
		{Id: 1, Body: "this is another chirp"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("value mis match: got %v, want %v\n", got, want)
	}

}
