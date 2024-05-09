package validate_test

import (
	"testing"

	"github.com/djmarkymark007/chirpy/internal/validate"
)

func TestProfaneFilter(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		msg := "This is a kerfuffle opinion I need to share with the world"
		want := "This is a **** opinion I need to share with the world"
		got := validate.ProfaneFilter(msg)
		if got != want {
			t.Errorf("got %s want %s\n", got, want)
		}
	})
	t.Run("Capitalize", func(t *testing.T) {
		msg := "This is a Kerfuffle opinion I need to share with the world"
		want := "This is a **** opinion I need to share with the world"
		got := validate.ProfaneFilter(msg)
		if got != want {
			t.Errorf("got %s want %s\n", got, want)
		}
	})
	t.Run("punctuation", func(t *testing.T) {
		msg := "This is a Kerfuffle. opinion I need to share with the world"
		want := "This is a Kerfuffle. opinion I need to share with the world"
		got := validate.ProfaneFilter(msg)
		if got != want {
			t.Errorf("got %s want %s\n", got, want)
		}

	})

	t.Run("All words", func(t *testing.T) {
		want := "****"
		for _, bannedWord := range validate.Profane {
			got := validate.ProfaneFilter(bannedWord)
			if got != want {
				t.Errorf("got %s want %s\n", got, want)
			}
		}
	})
}
