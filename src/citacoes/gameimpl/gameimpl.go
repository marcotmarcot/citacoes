package gameimpl

import (
	"citacoes/game"
	"citacoes/round"
	"encoding/csv"
	"log"
	"math/rand"
	"os"
	"strings"
)

type GameImpl struct {
	Round      *round.Round
	Points     map[string]int
	quotes     []game.Quote
	quoteIndex int
	numPlayers int
}

func NewGame() *GameImpl {
	g := &GameImpl{}
	quotesFile, err := os.Open("citacoes.csv")
	if err != nil {
		log.Fatal(err)
	}
	quotesFields, err := csv.NewReader(quotesFile).ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	for _, q := range quotesFields {
		g.quotes = append(g.quotes, game.Quote{q[0], strings.ToLower(q[1])})
	}
	g.quotes = shuffleQuotes(g.quotes)
	g.Points = make(map[string]int)
	g.quoteIndex = 0
	g.numPlayers = 3
	g.Round = round.NewRound(g)
	return g
}

func (g *GameImpl) NextQuote() {
	g.quoteIndex++
}

func (g *GameImpl) Quote() game.Quote {
	return g.quotes[g.quoteIndex]
}

func (g *GameImpl) NumPlayers() int {
	return g.numPlayers
}

// Returns if the round is ready to start.
func (g *GameImpl) NewRound(player string, clear bool) bool {
	if clear && g.Round.IsPlaying(player) {
		g.Round = round.NewRound(g)
	}
	if g.Round.Status() < round.SeenResultStatus {
		return false
	}
	return true
}

func (g *GameImpl) SetNumPlayers(numPlayers int) {
	g.numPlayers = numPlayers
}

// Returns if all the answers were already collected.
func (g *GameImpl) NewAnswer(player, answer string) bool {
	// Register new player. This is important so that all the players are shown
	// on the results, even those that don't have any points.
	if _, ok := g.Points[player]; !ok {
		g.Points[player] = 0
	}
	return g.Round.NewAnswer(player, answer)
}

// Returns if all the players have already chosen and answer.
func (g *GameImpl) AnswerChosen(player, choice string) bool {
	pointed, complete := g.Round.AnswerChosen(player, choice)
	for _, player := range pointed {
		g.Points[player]++
	}
	return complete
}

func shuffleQuotes(vals []game.Quote) []game.Quote {
	ret := make([]game.Quote, len(vals))
	for i, randIndex := range rand.Perm(len(vals)) {
		ret[i] = vals[randIndex]
	}
	return ret
}
