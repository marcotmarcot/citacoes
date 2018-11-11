// Bug: Quem votou em qual e qual é a certa.
// Bug: Fazer tolower das respostas
package main

import (
	"encoding/csv"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

var (
	quotes      []quote
	quoteIndex  int
	submissions []submission
	points      map[string]int
)

func main() {
	quotesFile, err := os.Open("citacoes.csv")
	if err != nil {
		log.Fatal(err)
	}
	quotesFields, err := csv.NewReader(quotesFile).ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	for _, q := range quotesFields {
		quotes = append(quotes, quote{q[0], q[1]})
	}
	rand.Seed(time.Now().Unix())
	clear()
	http.HandleFunc("/", writeAnswerHandler)
	http.HandleFunc("/answerWritten", answerWrittenHandler)
	http.HandleFunc("/chooseAnswer", chooseAnswerHandler)
	http.HandleFunc("/answerChosen", answerChosenHandler)
	http.HandleFunc("/results", resultsHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func clear() {
	submissions = nil
	points = make(map[string]int)
	quoteIndex = rand.Int() % len(quotes)
}

func writeAnswerHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("writeAnswer.html")
	if err != nil {
		log.Fatal(err)
	}
	if r.FormValue("clear") == "1" {
		clear()
	}
	t.Execute(w, writeAnswerInput{r.FormValue("name"), quotes[quoteIndex].text})
}

type writeAnswerInput struct {
	Name, Quote string
}

func answerWrittenHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("answerWritten.html")
	if err != nil {
		log.Fatal(err)
	}
	name := r.FormValue("name")
	if _, ok := points[name]; !ok {
		points[name] = 0
	}
	submission := submission{name, r.FormValue("answer")}
	submissions = append(submissions, submission)
	t.Execute(w, submission)
}

func chooseAnswerHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("chooseAnswer.html")
	if err != nil {
		log.Fatal(err)
	}
	quote := quotes[quoteIndex]
	truth := quote.truth
	answers := []string{truth}
	seen := map[string]bool{truth: true}
	for _, p := range rand.Perm(len(submissions)) {
		answer := submissions[p].Answer
		if seen[answer] || answer == r.FormValue("answer") {
			continue
		}
		answers = append(answers, answer)
	}
	t.Execute(w, chooseAnswerInput{r.FormValue("name"), quotes[quoteIndex].text, answers})
}

type chooseAnswerInput struct {
	Name    string
	Text    string
	Answers []string
}

func answerChosenHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("answerChosen.html")
	if err != nil {
		log.Fatal(err)
	}
	name := r.FormValue("name")
	chosen := r.FormValue("answer")
	for _, s := range submissions {
		if chosen == s.Answer {
			points[s.Name]++
		}
	}

	t.Execute(w, name)
}

func resultsHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("results.html")
	if err != nil {
		log.Fatal(err)
	}

	t.Execute(w, resultsInput{r.FormValue("name"), points})
}

type resultsInput struct {
	Name   string
	Points map[string]int
}

type quote struct {
	text, truth string
}

type submission struct {
	Name, Answer string
}
