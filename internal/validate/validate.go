package validate

import "strings"

var Profane = [3]string{
	"kerfuffle",
	"sharbert",
	"fornax",
}

func ProfaneFilter(msg string) string {
	words := strings.Split(msg, " ")
	for index, word := range words {
		for _, banWord := range Profane {
			if strings.ToLower(word) == banWord {
				words[index] = "****"
			}
		}
	}
	return strings.Join(words, " ")
}
