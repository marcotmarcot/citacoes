package main

import (
	"citacoes/game"
	"citacoes/gameimpl"
	"citacoes/round"
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	g  *gameimpl.GameImpl
	ip = flag.Int("port", 8080, "port to run the game")
)

func main() {
	flag.Parse()
	rand.Seed(time.Now().Unix())
	g = gameimpl.NewGame()
	http.HandleFunc("/", checkInHandler)
	http.HandleFunc("/writeAnswer", writeAnswerHandler)
	http.HandleFunc("/answerWritten", answerWrittenHandler)
	http.HandleFunc("/chooseAnswer", chooseAnswerHandler)
	http.HandleFunc("/answerChosen", answerChosenHandler)
	http.HandleFunc("/results", resultsHandler)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*ip), nil))
}

func checkInHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("checkIn.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, nil)
}

func writeAnswerHandler(w http.ResponseWriter, r *http.Request) {
	player := r.FormValue("player")
	clear := r.FormValue("clear") == "1"

	if g.NewRound(player, clear) {
		url := fmt.Sprintf("/results?player=%s", player)
		http.Redirect(w, r, url, 307)
		return
	}

	t, err := template.ParseFiles("writeAnswer.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, struct {
		Player, Quote string
		Players       []string
	}{player, g.Quote().Text, g.Round.PlayersReady(round.NotAnsweredStatus)})
}

func answerWrittenHandler(w http.ResponseWriter, r *http.Request) {
	player := r.FormValue("player")
	answer := strings.ToLower(r.FormValue("answer"))

	if g.NewAnswer(player, answer) {
		url := fmt.Sprintf("/chooseAnswer?player=%s&answer=%s", player, answer)
		http.Redirect(w, r, url, 307)
		return
	}
	ready := g.Round.PlayersReady(round.AnsweredStatus)

	t, err := template.ParseFiles("answerWritten.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, struct {
		Player       string
		Missing      int
		PlayersReady []string
		Answer       string
	}{player, g.NumPlayers() - len(ready), ready, answer})
}

func chooseAnswerHandler(w http.ResponseWriter, r *http.Request) {
	player := r.FormValue("player")
	answer := r.FormValue("answer")

	if g.Round.Status() < round.AnsweredStatus {
		url := fmt.Sprintf("/answerWritten?player=%s&answer=", player, answer)
		http.Redirect(w, r, url, 307)
		return
	}

	if choice, noChoice := g.Round.NoChoice(player, answer); noChoice {
		url := fmt.Sprintf("/answerChosen?player=%s&choice=%s", player, choice)
		http.Redirect(w, r, url, 307)
		return
	}

	t, err := template.ParseFiles("chooseAnswer.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, struct {
		Player  string
		Text    string
		Choices []string
	}{player, g.Quote().Text, g.Round.Choices(player, answer)})
}

func answerChosenHandler(w http.ResponseWriter, r *http.Request) {
	player := r.FormValue("player")
	choice := r.FormValue("choice")

	if g.AnswerChosen(player, choice) {
		url := fmt.Sprintf("/results?player=%s", player)
		http.Redirect(w, r, url, 307)
		return
	}
	ready := g.Round.PlayersReady(round.ChosenStatus)

	t, err := template.ParseFiles("answerChosen.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, struct {
		Player       string
		Missing      int
		PlayersReady []string
	}{player, g.NumPlayers() - len(ready), ready})
}

func resultsHandler(w http.ResponseWriter, r *http.Request) {
	player := r.FormValue("player")

	if g.Round.Status() < round.ChosenStatus {
		url := fmt.Sprintf("/answerChosen?player=%s", player)
		http.Redirect(w, r, url, 307)
		return
	}

	votedAnswers := g.Round.VotedAnswers(player)
	ready := g.Round.PlayersReady(round.SeenResultStatus)

	t, err := template.ParseFiles("results.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, struct {
		Player       string
		Quote        game.Quote
		TruthVoters  []string
		Answers      []round.VotedAnswer
		Points       map[string]int
		Missing      int
		PlayersReady []string
	}{player, g.Quote(), g.Round.TruthVoters(), votedAnswers, g.Points, g.NumPlayers() - len(ready), ready})
}

func parseInt(s string, def int) int {
	if i, err := strconv.Atoi(s); err != nil {
		log.Printf("Could not parse int %v: %v", s, err)
		return 3
	} else {
		return i
	}
}
