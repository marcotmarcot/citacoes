package round

import (
	"citacoes/game"
	"math/rand"
)

type Round struct {
	g            game.Game
	submissions  []Submission
	playerStatus map[string]Status
	// A mapping from each answer to the list of players that voted on it.
	voters map[string][]string
}

func NewRound(g game.Game) *Round {
	r := &Round{}
	r.g = g
	r.g.NextQuote()
	r.submissions = nil
	r.playerStatus = map[string]Status{}
	r.voters = map[string][]string{}
	return r
}

func (r *Round) IsPlaying(player string) bool {
	_, ok := r.playerStatus[player]
	return ok
}

func (r *Round) PlayersReady(status Status) []string {
	players := []string{}
	for player, playerStatus := range r.playerStatus {
		if playerStatus >= status {
			players = append(players, player)
		}
	}
	return players
}

func (r *Round) NewAnswer(player, answer string) bool {
	// Register submission if this player has not yet registered their
	// submission.
	if r.playerStatus[player] < AnsweredStatus {
		r.playerStatus[player] = AnsweredStatus
		r.submissions = append(r.submissions, Submission{player, answer})
	}
	return len(r.PlayersReady(AnsweredStatus)) >= r.g.NumPlayers()

}

// Returns the only option this player has if they have only one option.
func (r *Round) NoChoice(player, answer string) (string, bool) {
	choices := r.Choices(player, answer)
	if len(choices) == 1 {
		return choices[0], true
	}
	return "", len(choices) == 0
}

// Returns the options that this player has.
func (r *Round) Choices(player, answer string) []string {
	answers := []string{}
	seen := map[string]bool{}
	truth := r.g.Quote().Truth
	if truth != answer {
		answers = append(answers, truth)
		seen[truth] = true
	}
	for _, s := range r.submissions {
		if s.Player == player {
			continue
		}
		a := s.Answer
		if seen[a] || a == answer {
			continue
		}
		seen[a] = true
		answers = append(answers, a)
	}
	return shuffleStrings(answers)
}

// Returns the list of players that deserve to receive a point and if all the
// players have already chosen an answer.
func (r *Round) AnswerChosen(player, choice string) (pointed []string, complete bool) {
	if r.playerStatus[player] < ChosenStatus {
		r.playerStatus[player] = ChosenStatus
		r.voters[choice] = append(r.voters[choice], player)
		truth := r.g.Quote().Truth
		if choice == truth {
			pointed = append(pointed, player)
		}
		for _, s := range r.submissions {
			if choice == s.Answer {
				pointed = append(pointed, s.Player)
			}
			if s.Player == player && s.Answer == truth {
				pointed = append(pointed, player)
			}
		}
	}

	return pointed, len(r.PlayersReady(ChosenStatus)) >= r.g.NumPlayers()
}

// Returns the answers with their votes and update the status of the player.
func (r *Round) VotedAnswers(player string) []VotedAnswer {
	r.playerStatus[player] = SeenResultStatus

	var answers []VotedAnswer
	for _, s := range r.submissions {
		answers = append(answers, VotedAnswer{s.Player, s.Answer, r.voters[s.Answer]})
	}
	return answers
}

// Returns the players that voted in the Truth.
func (r *Round) TruthVoters() []string {
	return r.voters[r.g.Quote().Truth]
}

type Status int

const (
	NotAnsweredStatus Status = iota
	AnsweredStatus
	ChosenStatus
	SeenResultStatus
)

type Submission struct {
	Player, Answer string
}

func shuffleStrings(vals []string) []string {
	ret := make([]string, len(vals))
	for i, randIndex := range rand.Perm(len(vals)) {
		ret[i] = vals[randIndex]
	}
	return ret
}

type VotedAnswer struct {
	Player string
	Answer string
	Voters []string
}
