package main

import (
	"encoding/csv"
	"flag"
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

var (
	g  *game
	ip = flag.Int("port", 8080, "port to run the game")
)

func main() {
	flag.Parse()
	rand.Seed(time.Now().Unix())
	g = newGame()
	http.HandleFunc("/", writeAnswerHandler)
	http.HandleFunc("/answerWritten", answerWrittenHandler)
	http.HandleFunc("/chooseAnswer", chooseAnswerHandler)
	http.HandleFunc("/answerChosen", answerChosenHandler)
	http.HandleFunc("/results", resultsHandler)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*ip), nil))
}

func writeAnswerHandler(w http.ResponseWriter, r *http.Request) {
	player := r.FormValue("player")
	clear := r.FormValue("clear") == "1"

	if g.newRound(player, clear) {
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
		NumPlayers    int
		Players       []string
	}{player, g.currentQuote().Text, g.numPlayers, g.playersReady(0)})
}

func answerWrittenHandler(w http.ResponseWriter, r *http.Request) {
	player := r.FormValue("player")
	answer := strings.ToLower(r.FormValue("answer"))
	numPlayers := parseInt(r.FormValue("numPlayers"), 3)

	g.numPlayers = numPlayers
	if g.newAnswer(player, answer) {
		url := fmt.Sprintf("/chooseAnswer?player=%s&answer=%s", player, answer)
		http.Redirect(w, r, url, 307)
		return
	}
	ready := g.playersReady(answeredStatus)

	t, err := template.ParseFiles("answerWritten.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, struct {
		Player       string
		Missing      int
		PlayersReady []string
		Answer       string
	}{player, g.numPlayers - len(ready), ready, answer})
}

func chooseAnswerHandler(w http.ResponseWriter, r *http.Request) {
	player := r.FormValue("player")
	answer := r.FormValue("answer")

	if g.gameStatus() < answeredStatus {
		url := fmt.Sprintf("/answerWritten?player=%s&answer=", player, answer)
		http.Redirect(w, r, url, 307)
		return
	}

	if choice, noChoice := g.noChoice(player, answer); noChoice {
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
	}{player, g.currentQuote().Text, g.choices(player, answer)})
}

func answerChosenHandler(w http.ResponseWriter, r *http.Request) {
	player := r.FormValue("player")
	choice := r.FormValue("choice")

	if g.answerChosen(player, choice) {
		url := fmt.Sprintf("/results?player=%s", player)
		http.Redirect(w, r, url, 307)
		return
	}
	ready := g.playersReady(chosenStatus)

	t, err := template.ParseFiles("answerChosen.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, struct {
		Player       string
		Missing      int
		PlayersReady []string
	}{player, g.numPlayers - len(ready), ready})
}

func resultsHandler(w http.ResponseWriter, r *http.Request) {
	player := r.FormValue("player")

	if g.gameStatus() < chosenStatus {
		url := fmt.Sprintf("/answerChosen?player=%s", player)
		http.Redirect(w, r, url, 307)
		return
	}

	votedAnswers := g.votedAnswers(player)
	ready := g.playersReady(seenResultStatus)

	t, err := template.ParseFiles("results.html")
	if err != nil {
		log.Fatal(err)
	}
	t.Execute(w, struct {
		Player       string
		Quote        quote
		Answers      []votedAnswer
		Points       map[string]int
		Missing      int
		PlayersReady []string
	}{player, g.currentQuote(), votedAnswers, g.points, g.numPlayers - len(ready), ready})
}

type game struct {
	quotes       []quote
	quoteIndex   int
	points       map[string]int
	submissions  []submission
	playerStatus map[string]status
	voters       map[string][]string
	numPlayers   int
}

func newGame() *game {
	g := &game{}
	quotesFile, err := os.Open("citacoes.csv")
	if err != nil {
		log.Fatal(err)
	}
	quotesFields, err := csv.NewReader(quotesFile).ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	for _, q := range quotesFields {
		g.quotes = append(g.quotes, quote{q[0], q[1]})
	}
	g.quotes = Shuffle(g.quotes)
	g.points = make(map[string]int)
	g.quoteIndex = 0
	g.numPlayers = 3
	g.restart()
	return g
}

// Returns if the round is ready to start.
func (g *game) newRound(player string, clear bool) bool {
	_, ok := g.playerStatus[player]
	if clear && ok {
		g.restart()
	}
	if g.gameStatus() < seenResultStatus {
		return false
	}
	return true
}

// Returns if all the answers were already collected.
func (g *game) newAnswer(player, answer string) bool {
	// Register new player.
	if _, ok := g.points[player]; !ok {
		g.points[player] = 0
	}
	// Register submission if this player has not yet registered their
	// submission.
	if g.playerStatus[player] < answeredStatus {
		g.playerStatus[player] = answeredStatus
		g.submissions = append(g.submissions, submission{player, answer})
	}
	return g.gameStatus() >= answeredStatus
}

// Returns the only option this player has if they have only one option.
func (g *game) noChoice(player, answer string) (string, bool) {
	choices := g.choices(player, answer)
	if len(choices) == 1 {
		return choices[0], true
	}
	return "", len(choices) == 0
}

// Returns the options that this player has.
func (g *game) choices(player, answer string) []string {
	answers := []string{g.currentQuote().Truth}
	seen := map[string]bool{g.currentQuote().Truth: true}
	for _, p := range rand.Perm(len(g.submissions)) {
		if g.submissions[p].Player == player {
			continue
		}
		a := g.submissions[p].Answer
		if seen[a] || a == answer {
			continue
		}
		seen[a] = true
		answers = append(answers, a)
	}
	return answers
}

// Returns if all the answers were already chosen.
func (g *game) answerChosen(player, choice string) bool {
	if g.playerStatus[player] < chosenStatus {
		g.playerStatus[player] = chosenStatus
		g.voters[choice] = append(g.voters[choice], player)
		if choice == g.currentQuote().Truth {
			g.points[player]++
		}
		for _, s := range g.submissions {
			if choice == s.Answer {
				g.points[s.Player]++
			}
			if s.Player == player && s.Answer == g.currentQuote().Truth {
				g.points[player]++
			}
		}
	}

	return g.gameStatus() >= chosenStatus
}

// Returns the answers with their votes and update the status of the player.
func (g *game) votedAnswers(player string) []votedAnswer {
	g.playerStatus[player] = seenResultStatus

	var answers []votedAnswer
	for _, s := range g.submissions {
		answers = append(answers, votedAnswer{s.Player, s.Answer, g.voters[s.Answer]})
	}
	return answers
}

func (g *game) restart() {
	g.submissions = nil
	g.playerStatus = make(map[string]status)
	g.voters = make(map[string][]string)
	g.quoteIndex++
}

func (g *game) currentQuote() quote {
	return g.quotes[g.quoteIndex]
}

func (g *game) playersReady(s status) []string {
	players := []string{}
	for player, playerStatus := range g.playerStatus {
		if playerStatus >= s {
			players = append(players, player)
		}
	}
	return players
}

// Returns the status of the game, which is the status of the player in the
// earliest status.
func (g *game) gameStatus() status {
	if len(g.playerStatus) < g.numPlayers {
		return notAnsweredStatus
	}
	s := seenResultStatus
	for _, playerStatus := range g.playerStatus {
		if playerStatus < s {
			s = playerStatus
		}
	}
	return s
}

type quote struct {
	Text, Truth string
}

type submission struct {
	Player, Answer string
}

type status int

const (
	notAnsweredStatus status = iota
	answeredStatus
	chosenStatus
	seenResultStatus
)

type votedAnswer struct {
	Player string
	Answer string
	Voters []string
}

func Shuffle(vals []quote) []quote {
	ret := make([]quote, len(vals))
	for i, randIndex := range rand.Perm(len(vals)) {
		ret[i] = vals[randIndex]
	}
	return ret
}

func parseInt(s string, def int) int {
	if i, err := strconv.Atoi(s); err != nil {
		log.Printf("Could not parse int %v: %v", s, err)
		return 3
	} else {
		return i
	}
}
