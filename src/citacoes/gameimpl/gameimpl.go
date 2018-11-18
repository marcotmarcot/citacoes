package gameimpl

import (
	"citacoes/game"
	"citacoes/round"
	"encoding/csv"
	"log"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"
)

type GameImpl struct {
	Round      *round.Round
	points     map[string]int
	quotes     []game.Quote
	quoteIndex int
	lastSeen   map[string]time.Time
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
	g.points = map[string]int{}
	g.quoteIndex = 0
	g.lastSeen = map[string]time.Time{}
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
	n := 0
	for _, timeSeen := range g.lastSeen {
		if time.Since(timeSeen).Seconds() < 30 {
			n++
		}
	}
	if n < 3 {
		n = 3
	}
	return n
}

// Returns if the round is ready to start.
func (g *GameImpl) NewRound(player string, clear bool) bool {
	if clear && g.Round.IsPlaying(player) {
		if len(g.Round.PlayersReady(round.SeenResultStatus)) < g.NumPlayers() {
			return false
		}
		g.Round = round.NewRound(g)
	}
	if player != "" {
		g.lastSeen[player] = time.Now()
	}
	return true
}

// Returns if all the answers were already collected.
func (g *GameImpl) NewAnswer(player, answer string) bool {
	g.lastSeen[player] = time.Now()
	// Register new player. This is important so that all the players are shown
	// on the results, even those that don't have any points.
	if _, ok := g.points[player]; !ok {
		g.points[player] = 0
	}
	return g.Round.NewAnswer(player, answer)
}

// Returns if all the players have already chosen and answer.
func (g *GameImpl) AnswerChosen(player, choice string) bool {
	g.lastSeen[player] = time.Now()
	pointed, complete := g.Round.AnswerChosen(player, choice)
	for _, player := range pointed {
		g.points[player]++
	}
	return complete
}

func (g *GameImpl) Points() []RankedPlayer {
	r := []RankedPlayer{}
	for player, points := range g.points {
		r = append(r, RankedPlayer{player, points})
	}
	sort.Sort(byPoints(r))
	return r
}

type RankedPlayer struct {
	Player string
	Points int
}

type byPoints []RankedPlayer

func (b byPoints) Len() int {
	return len(b)
}

func (b byPoints) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byPoints) Less(i, j int) bool {
	return b[i].Points > b[j].Points
}

func shuffleQuotes(vals []game.Quote) []game.Quote {
	ret := make([]game.Quote, len(vals))
	for i, randIndex := range rand.Perm(len(vals)) {
		ret[i] = vals[randIndex]
	}
	return ret
}
