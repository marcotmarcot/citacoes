// Bug: O voto em sua própria resposta não vale.
// Bug: Linha depois do texto na escolha.
// Bug: Quem votou em qual e qual é a certa.
// Bug: Novo jogo está zerando para todo mundo.
// Bug: Não dar ponto de quem voto no seu mesmo
// Bug: Não exibir respostas duplicadas
// Bug: Não exibir as próprias respostas
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
	quotes = []quote{
		{}}
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
	quoteIndex = rand.Int()%len(quotes)
}

func writeAnswerHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("writeAnswer.html")
	if err != nil {
		log.Fatal(err)
	}
	if r.FormValue("clear") == "1" {
		clear()
	}
	t.Execute(w, writeAnswerInput{quotes[quoteIndex].text, r.FormValue("name")})
}

type writeAnswerInput struct {
	Quote, Name string
}

func answerWrittenHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("answerWritten.html")
	if err != nil {
		log.Fatal(err)
	}
	name := r.FormValue("name")
	submissions = append(submissions, submission{name, r.FormValue("answer")})
	t.Execute(w, r.FormValue("name"))
}

func chooseAnswerHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("chooseAnswer.html")
	if err != nil {
		log.Fatal(err)
	}
	answers := []string{quotes[quoteIndex].truth}
	for _, p := range rand.Perm(len(submissions)) {
		answers = append(answers, submissions[p].answer)
	}
	t.Execute(w, chooseAnswerInput{r.FormValue("name"), quotes[quoteIndex].text, answers})
}

type chooseAnswerInput struct {
	Name    string
	Text string
	Answers []string
}

func answerChosenHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("answerChosen.html")
	if err != nil {
		log.Fatal(err)
	}
	name := r.FormValue("name")
	chosen := r.FormValue("answer")
	if chosen == quotes[quoteIndex].truth {
		points[name]++
	}
	for _, s := range submissions {
		if chosen == s.answer {
			points[s.name]++
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
	name, answer string
}
