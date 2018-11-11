// Bug: Quem votou em qual e qual é a certa.
// Bug: Voltar à tela inicial pode reiniciar antes de alguém ver resultados
package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	written string = "escreveu"
	chosen         = "escolheu"
)

var (
	quotes      []quote
	quoteIndex  int
	submissions []submission
	points      map[string]int
	players     map[string]string
	numPlayers  int
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
	players = make(map[string]string)
	quoteIndex = rand.Int() % len(quotes)
}

func writeAnswerHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("writeAnswer.html")
	if err != nil {
		log.Fatal(err)
	}
	name := r.FormValue("name")
	_, ok := players[name]
	if r.FormValue("clear") == "1" && ok {
		clear()
	}
	t.Execute(w, struct {
		Name, Quote string
		NumPlayers  int
		Players     map[string]string
	}{name, quotes[quoteIndex].text, numPlayers, players})
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
	if r.FormValue("numPlayers") != "" {
		tmpNum, err := strconv.Atoi(r.FormValue("numPlayers"))
		if err != nil {
			log.Printf("%s cometeu um vacilo %s", name, err.Error())
		} else {
			numPlayers = tmpNum
		}
	}
	s := submission{}
	if players[name] != written {
		players[name] = written
		s = submission{name, strings.ToLower(r.FormValue("answer"))}
		submissions = append(submissions, s)
	}
	t.Execute(w, struct {
		Name    string
		Players map[string]string
		Answer  string
	}{name, players, s.Answer})
}

func chooseAnswerHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if !playersReady(written) {
		url := fmt.Sprintf("/answerWritten?name=%s", name)
		http.Redirect(w, r, url, 307)
		return
	}
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
	t.Execute(w, struct {
		Name    string
		Text    string
		Answers []string
	}{name, quotes[quoteIndex].text, answers})
}

func answerChosenHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("answerChosen.html")
	if err != nil {
		log.Fatal(err)
	}
	name := r.FormValue("name")
	if players[name] != chosen {
		players[name] = chosen
		chosen := r.FormValue("answer")
		if chosen == quotes[quoteIndex].truth {
			points[name]++
		}
		for _, s := range submissions {
			if chosen == s.Answer {
				points[s.Name]++
			}
		}
	}

	t.Execute(w, struct {
		Name    string
		Players map[string]string
	}{name, players})
}

func resultsHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if !playersReady(chosen) {
		url := fmt.Sprintf("/answerChosen?name=%s", name)
		http.Redirect(w, r, url, 307)
		return
	}
	t, err := template.ParseFiles("results.html")
	if err != nil {
		log.Fatal(err)
	}

	t.Execute(w, struct {
		Name   string
		Points map[string]int
	}{name, points})
}

func playersReady(state string) bool {
	intRepr := func(s string) int {
		num := -1
		switch s {
		case written:
			num = 0
		case chosen:
			num = 1
		}
		return num
	}
	total := 0
	for _, player := range players {
		if intRepr(player) >= intRepr(state) {
			total++
		}
	}
	if total < numPlayers {
		return false
	}
	return true
}

type quote struct {
	text, truth string
}

type submission struct {
	Name, Answer string
}
