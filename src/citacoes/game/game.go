package game

type Game interface {
	NextQuote()
	Quote() Quote
	NumPlayers() int
}

type Quote struct {
	Text, Truth string
}
