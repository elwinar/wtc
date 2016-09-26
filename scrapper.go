package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

type Team struct {
	ID      int
	Country string
	Name    string
	Players []Player
}

type Player struct {
	ID      int
	Name    string
	Faction string
	Lists   []List
}

type List struct {
	ID     int
	Caster string
}

type Match struct {
	ID    int
	Round int
}

type MatchList struct {
	MatchID  int
	PlayerID int
	Win      bool
}

type Pairing struct {
	Round int
	Zone  string
	Team1 string
	Team2 string
	Pairs []Pair
}

type Pair struct {
	Player1 string
	List1   string
	Player2 string
	List2   string
	Winner  int
}

func main() {
	var pages = make(chan io.Reader)
	go func() {
		for _, url := range []string{
			"http://wmh-wtc.com/?round=1",
			"http://wmh-wtc.com/?round=2",
			"http://wmh-wtc.com/?round=3",
			"http://wmh-wtc.com/?round=4",
			"http://wmh-wtc.com/?round=5",
			"http://wmh-wtc.com/?round=6",
		} {
			res, err := http.Get(url)
			if err != nil {
				fmt.Println(err)
				continue
			}
			pages <- res.Body
		}
		close(pages)
	}()

	var pairings = make(chan Pairing)
	go func() {
		var round = 0
		for page := range pages {
			round++

			root, err := html.Parse(page)
			if err != nil {
				fmt.Println(err)
				return
			}

			var nodes = make(chan *html.Node)
			go func() {
				walk(root, nodes)
				close(nodes)
			}()

			for node := range nodes {
				pairing, err := extract(node)
				if err != nil {
					fmt.Println(err)
					continue
				}
				pairing.Round = round
				pairings <- pairing
			}
		}
		close(pairings)
	}()

	var teams = make(map[string]Team)
	for pairing := range pairings {
		if _, found := teams[pairing.Team1]; !found {
			team1 := strings.SplitN(pairing.Team1, " ", 3)
			if len(team1) < 3 {
				team1 = append(team1, "", "", "")
			}
			t := Team{
				ID:      NewID("team"),
				Country: team1[1],
				Name:    team1[2],
			}
			for _, pair := range pairing.Pairs {
				t.Players = append(t.Players, Player{
					ID:   NewID("player"),
					Name: pair.Player1,
				})
			}
			teams[pairing.Team1] = t
		}
		if _, found := teams[pairing.Team2]; !found {
			team2 := strings.SplitN(pairing.Team2, " ", 3)
			if len(team2) < 3 {
				team2 = append(team2, "", "", "")
			}
			t := Team{
				ID:      NewID("team"),
				Country: team2[1],
				Name:    team2[2],
			}
			for _, pair := range pairing.Pairs {
				t.Players = append(t.Players, Player{
					ID:   NewID("player"),
					Name: pair.Player2,
				})
			}
			teams[pairing.Team2] = t
		}
	}
}

var counters = map[string]int{}

func NewID(t string) int {
	counters[t]++
	return counters[t]
}

func walk(node *html.Node, pairings chan *html.Node) {
	var isPairing bool
	for _, attr := range node.Attr {
		if attr.Key == "class" && strings.Contains(attr.Val, "pairing-row") {
			isPairing = true
			break
		}
	}

	if isPairing {
		pairings <- node
		return
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walk(child, pairings)
	}
}

func extract(node *html.Node) (Pairing, error) {
	zone := node.FirstChild
	team1 := zone.NextSibling
	team2 := team1.NextSibling.NextSibling
	pairs := team2.NextSibling

	var pairing Pairing
	if zone == nil || team1 == nil || team2 == nil || pairs == nil {
		return pairing, errors.New("malformed node")
	}

	pairing.Zone = zone.LastChild.FirstChild.Data
	pairing.Team1 = team1.FirstChild.LastChild.FirstChild.Data
	pairing.Team2 = team2.FirstChild.LastChild.FirstChild.Data

	for pairNode := pairs.FirstChild; pairNode != nil; pairNode = pairNode.NextSibling {
		var pair = Pair{
			Player1: pairNode.FirstChild.FirstChild.Data,
			List1:   pairNode.FirstChild.LastChild.FirstChild.Data,
			Player2: pairNode.LastChild.FirstChild.Data,
			List2:   pairNode.LastChild.LastChild.FirstChild.Data,
		}

		for _, attr := range pairNode.FirstChild.Attr {
			if attr.Key != "class" {
				continue
			}

			if strings.Contains(attr.Val, "winner") {
				pair.Winner = 1
			} else {
				pair.Winner = 2
			}
			break
		}

		pairing.Pairs = append(pairing.Pairs, pair)
	}
	return pairing, nil
}
