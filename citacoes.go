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
	answered int = iota + 1
	chosen
	resulted
)

var (
	quotes      []quote
	quoteIndex  int
	submissions []submission
	points      map[string]int
	players     map[string]int
	choices     map[string]string
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
	players = make(map[string]int)
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
		if len(getPlayersReady(resulted)) < numPlayers {
			url := fmt.Sprintf("/results?name=%s", name)
			http.Redirect(w, r, url, 307)
			return
		}
		clear()
	}
	t.Execute(w, struct {
		Name, Quote string
		NumPlayers  int
		Players     []string
	}{name, quotes[quoteIndex].Text, numPlayers, getPlayersReady(0)})
}

func answerWrittenHandler(w http.ResponseWriter, r *http.Request) {
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
	if players[name] < answered {
		players[name] = answered
		submissions = append(submissions, submission{name, answer})
	}
	ready := getPlayersReady(answered)
	if len(ready) >= numPlayers {
		url := fmt.Sprintf("/chooseAnswer?name=%s&answer=%s", name, answer)
		http.Redirect(w, r, url, 307)
		return
	}
	t, err := template.ParseFiles("answerWritten.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, struct {
		Name         string
		Missing      int
		PlayersReady []string
		Answer       string
	}{name, numPlayers - len(ready), ready, answer})
}

func chooseAnswerHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	answer := r.FormValue("answer")
	if len(getPlayersReady(answered)) < numPlayers {
		url := fmt.Sprintf("/answerWritten?name=%s&answer=%s", name, answer)
		http.Redirect(w, r, url, 307)
		return
	}
	answers := []string{}
	seen := map[string]bool{}
	for _, p := range rand.Perm(len(submissions)) {
		a := submissions[p].Answer
		if seen[a] || a == answer {
			continue
		}
		seen[a] = true
		answers = append(answers, a)
	}
	if len(answers) < 2 {
		chosen := answer
		if len(answers) == 1 {
			chosen = answers[0]
		}
		url := fmt.Sprintf("/answerChosen?name=%s&answer=%s", name, chosen)
		http.Redirect(w, r, url, 307)
		return
	}
	t, err := template.ParseFiles("chooseAnswer.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, struct {
		Name    string
		Text    string
		Answers []string
	}{name, quotes[quoteIndex].Text, answers})
}

func answerChosenHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	answer := r.FormValue("answer")
	if players[name] < chosen {
		choices[name] = answer
		players[name] = chosen
		for _, s := range submissions {
			if answer == s.Answer {
				points[s.Name]++
			}
		}
	}

	ready := getPlayersReady(chosen)
	if len(ready) >= numPlayers {
		url := fmt.Sprintf("/results?name=%s", name)
		http.Redirect(w, r, url, 307)
		return
	}
	t, err := template.ParseFiles("answerChosen.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, struct {
		Name         string
		Missing      int
		PlayersReady []string
	}{name, numPlayers - len(ready), ready})
}

func resultsHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if len(getPlayersReady(chosen)) < numPlayers {
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
	players[name] = resulted

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

	ready := getPlayersReady(resulted)
	t.Execute(w, struct {
		Name         string
		Quote        quote
		Choices      map[string]authoredAnswer
		Points       map[string]int
		Missing      int
		PlayersReady []string
	}{name, quotes[quoteIndex], c, points, numPlayers - len(ready), ready})
}

func getPlayersReady(state int) []string {
	pls := []string{}
	for name, status := range players {
		if status >= state {
			pls = append(pls, name)
		}
	}
	return pls
}

type quote struct {
	Text, Truth string
}

type submission struct {
	Name, Answer string
}

func Shuffle(vals []quote) []quote {
	ret := make([]quote, len(vals))
	for i, randIndex := range rand.Perm(len(vals)) {
		ret[i] = vals[randIndex]
	}
	return ret
}
