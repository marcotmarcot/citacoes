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
	choices     map[string]string
	numPlayers  int
)

func Shuffle(vals []quote) []quote {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ret := make([]quote, len(vals))
	perm := r.Perm(len(vals))
	for i, randIndex := range perm {
		ret[i] = vals[randIndex]
	}
	return ret
}

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
	points = make(map[string]int)
	rand.Seed(time.Now().Unix())
	quotes = Shuffle(quotes)
	quoteIndex = 0
	numPlayers = 3
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
	players = make(map[string]string)
	choices = make(map[string]string)
	quoteIndex++
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
	}{name, quotes[quoteIndex].Text, numPlayers, players})
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
	answer := strings.ToLower(r.FormValue("answer"))
	if r.FormValue("numPlayers") != "" {
		tmpNum, err := strconv.Atoi(r.FormValue("numPlayers"))
		if err != nil {
			log.Printf("%s cometeu um vacilo %s", name, err.Error())
		} else {
			numPlayers = tmpNum
		}
	}
	if players[name] != written {
		players[name] = written
		submissions = append(submissions, submission{name, answer})
	}
	t.Execute(w, struct {
		Name    string
		Players map[string]string
		Answer  string
	}{name, players, answer})
}

func chooseAnswerHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	answer := r.FormValue("answer")
	if !playersReady(written) {
		url := fmt.Sprintf("/answerWritten?answer=%s&name=%s", answer, name)
		http.Redirect(w, r, url, 307)
		return
	}
	t, err := template.ParseFiles("chooseAnswer.html")
	if err != nil {
		log.Fatal(err)
	}
	answers := []string{}
	seen := map[string]bool{}
	for _, p := range rand.Perm(len(submissions)) {
		a := submissions[p].Answer
		if seen[a] || a == answer {
			continue
		}
		answers = append(answers, a)
	}
	t.Execute(w, struct {
		Name    string
		Text    string
		Answers []string
	}{name, quotes[quoteIndex].Text, answers})
}

func answerChosenHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("answerChosen.html")
	if err != nil {
		log.Fatal(err)
	}
	name := r.FormValue("name")
	answer := r.FormValue("answer")
	choices[name] = answer
	if players[name] != chosen {
		players[name] = chosen
		for _, s := range submissions {
			if answer == s.Answer {
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

	type authoredAnswer struct {
		Answer  string
		Authors []string
	}

	c := map[string]authoredAnswer{}
	for name, answer := range choices {
		var names []string
		for _, s := range submissions {
			if s.Answer == answer {
				names = append(names, s.Name)
			}
		}
		c[name] = authoredAnswer{answer, names}
	}

	t.Execute(w, struct {
		Name    string
		Quote   quote
		Choices map[string]authoredAnswer
		Points  map[string]int
	}{name, quotes[quoteIndex], c, points})
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
	Text, Truth string
}

type submission struct {
	Name, Answer string
}
