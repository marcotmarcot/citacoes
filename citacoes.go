// Bug: O voto em sua própria resposta não vale.
// Bug: Linha depois do texto na escolha.
// Bug: Quem votou em qual e qual é a certa.
// Bug: Novo jogo está zerando para todo mundo.
package main

import (
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"time"
)

var (
	quotes = []quote{
		{"O que a vida quer da gente é __________. João Guimarães Rosa.", "coragem"},
		{"Não é que eu tenha medo de morrer. É que __________ na hora que isso acontecer. Woody Allen.", "eu não quero estar lá"},
		{"O __________ não é tudo. Não se esqueça também do ouro, dos diamantes, da platina e das propriedades. Tom Jobim.", "dinheiro"},
		{"Era um menino tão mau que só se tornou __________ para ver a caveira dos outros. Jô Soares.", "radiologista"},
		{"Era tão azarado que, se quisesse __________, era só sentar-se nele. Jô Soares.", "achar uma agulha no palheiro"},
		{"O dinheiro não nos traz necessariamente a felicidade. Uma pessoa que tem dez milhões de dólares não é mais feliz do que __________. H. Brown.", "a que tem só nove milhões"},
		{"O amor é cego, mas o __________ devolve a visão.", "casamento"},
		{"Sou __________, pois não tenho onde cair morto.", "imortal"},
		{"Se é para morrer de batida... Que seja de __________. Valéria Alves de Lima.", "maracujá"},
		{"- Dói, né? - O quê? - Deitar no sofá e lembrar que __________.", "esqueceu o controle"},
		{"Meu pai, quando eu era criança, fez uma bicicleta inteira usando só __________.", "talo de mamoeiro"},
		{"Se você não consegue explicar algo __________, você não entendeu suficientemente bem.", "Albert Einstein"},
		{"A reputação de um médico se faz pelo número de pessoas famosas que __________.", "morrem sob seus cuidados"},
		{"Se ferradura desse sorte __________.", "burro não puxava carroça"},
		{"Marquei um encontro pela internet, quando cheguei lá a pessoa era __________.", "meu irmão"},
		{"Por que o youtuber foi ao dentista? Porque ele queria __________.", "fazer um canal"},
		{"Pareço normal, mas já fui no dicionário procurar o que era __________.", "dicionário"},
		{"Por que vocs preocupais com __________? Olhai como crescem os lírios do campo! Jesus de Nazaré.", "vestuário"},
		{"Eu jamais iria para a fogueira por uma opinião minha, afinal, __________. Friedrich Wilhelm Nietsche.", "não tenho certeza alguma"},
		{"Se querer conhecer a uma pessoa, não lhe perguntes o que pensa mas sim __________. Santo Agostinho.", "o que ela ama"},
		{"As armas são para dizer que lutamos e as rosas para dizer que __________. Axl Rose.", "vencemos"},
		{"O ignorante afirma, o sábio duvida, o sensato __________. Aristóteles.", "reflete"},
		{"Quem pensa pouco, __________ muito. Leonardo Da Vinci.", "erra"},
		{"Toda a música que não __________ é apenas um ruído. D'Alembert.", "pinta nada"},
		{"A harmonia se obtém pela __________. Platão", "virtude"},
		{"Antes de fazer alguma coisa, pense. Quando achar que já pode fazê-la, __________. Pitágoras.", "pense novamente"},
		{"Minhas refeição e minhas __________ gastronômicas. Título de blog.", "reflexão"},
		{"Deus ama ao que __________ com alegria. Coríntios 9:7.", "dá"},
		{"Feliz aquele que pegar em teus filhos e der com eles __________. Salmos 137:9.", "nas pedras"},
		{"Eis que reprovarei a vossa semente, e espalharei __________ sobre os vossos rostos. Malaquia 2:3.", "esterco"},
		{"Samaria virá a ser deserta, porque se rebelou contra o seu Deus; cairão à espada, seus filhos serão despedaçados, e as suas __________ serão fendidas pelo meio. Oseias 13:16.", "grávidas"},
		{"E __________ servirá de comida a todas as aves dos céus, e aos animais da terra; e ninguém os espantará. Deuteronômio 28:26.", "o teu cadáver"},
		{}}
	quoteIndex  int
	submissions []submission
	points      map[string]int
)

func main() {
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
			points[name]++
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
