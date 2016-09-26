package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"golang.org/x/net/html"
)

type Match struct {
	ID    int
	Round int
	Zone  string
	Teams [2]string
	Games [5]Game
}

type Game struct {
	ID      int
	Players [2]string
	Lists   [2]string
	Winner  int
}

type Team struct {
	ID      int
	Name    string
	Players [5]string
}

type Player struct {
	ID     int
	Name   string
	TeamID int
	Lists  [2]string
}

type List struct {
	ID       int
	Caster   string
	PlayerID int
}

func main() {
	var matches []Match
	for round := 1; round < 7; round++ {
		res, err := http.Get(fmt.Sprintf("http://wmh-wtc.com/?round=%d", round))
		if err != nil {
			log.Println("fetching round %d: %s", round, err)
			continue
		}

		root, err := html.Parse(res.Body)
		if err != nil {
			log.Println("parsing round %d: %s", round, err)
			continue
		}

		nodes := walk(root, nil)
		for _, matchNode := range nodes {
			var match = Match{
				ID:    ID("match"),
				Round: round,
				Zone:  strings.TrimSpace(matchNode.FirstChild.LastChild.FirstChild.Data),
			}

			for i, teamNode := range []*html.Node{
				matchNode.FirstChild.NextSibling,
				matchNode.FirstChild.NextSibling.NextSibling.NextSibling,
			} {
				match.Teams[i] = strings.TrimSpace(teamNode.FirstChild.LastChild.FirstChild.Data)
			}

			var g int
			for gameNode := matchNode.LastChild.FirstChild; gameNode != nil; gameNode = gameNode.NextSibling {
				var game = Game{
					ID: ID("game"),
					Players: [2]string{
						strings.TrimSpace(gameNode.FirstChild.FirstChild.Data),
						strings.TrimSpace(gameNode.LastChild.FirstChild.Data),
					},
					Lists: [2]string{
						strings.TrimSpace(gameNode.FirstChild.LastChild.FirstChild.Data),
						strings.TrimSpace(gameNode.LastChild.LastChild.FirstChild.Data),
					},
				}

				for _, attr := range gameNode.FirstChild.Attr {
					if attr.Key != "class" {
						continue
					}

					if strings.Contains(attr.Val, "winner") {
						game.Winner = 0
					} else {
						game.Winner = 1
					}
					break
				}

				match.Games[g] = game
				g++
			}

			matches = append(matches, match)
		}
	}

	var teams = map[string]Team{}
	var players = map[string]Player{}
	var lists = map[string]map[string]List{}

	for _, match := range matches {
		for t := 0; t <= 1; t++ {
			if _, found := teams[match.Teams[t]]; !found {
				team := Team{
					ID:   ID("team"),
					Name: match.Teams[t],
				}
				for g, game := range match.Games {
					players[game.Players[t]] = Player{
						ID:     ID("player"),
						Name:   game.Players[t],
						TeamID: team.ID,
					}
					team.Players[g] = game.Players[t]
				}
				teams[team.Name] = team
			}

			for _, game := range match.Games {
				for p := 0; p <= 1; p++ {
					if _, found := lists[game.Players[p]]; !found {
						lists[game.Players[p]] = map[string]List{}
					}

					if _, found := lists[game.Players[p]][game.Lists[p]]; !found {
						lists[game.Players[p]][game.Lists[p]] = List{
							ID:       ID("list"),
							Caster:   game.Lists[p],
							PlayerID: players[game.Players[p]].ID,
						}
					}
				}
			}
		}
	}

	db, err := sqlx.Connect("sqlite3", "data.sqlite")
	if err != nil {
		log.Println(err)
		return
	}

	db.MustExec("create table team ( id integer, name varchar(50), country varchar(50) )")
	db.MustExec("create table player ( id integer, name varchar(50), team_id integer )")
	db.MustExec("create table list ( id integer, caster varchar(50), player_id integer )")

	for _, team := range teams {
		chunks := append(strings.SplitN(team.Name, " ", 3), "")
		db.MustExec("insert into team values (?, ?, ?)", team.ID, chunks[2], chunks[1])

		for _, p := range team.Players {
			player := players[p]
			db.MustExec("insert into player values (?, ?, ?)", player.ID, player.Name, player.TeamID)

			for _, list := range lists[p] {
				db.MustExec("insert into list values (?, ?, ?)", list.ID, list.Caster, list.PlayerID)
			}
		}
	}

	db.MustExec("create table match ( id integer, round integer )")
	db.MustExec("create table game ( id integer, match_id integer )")
	db.MustExec("create table report ( id integer, game_id integer, list_id integer, won boolean )")

	for _, match := range matches {
		db.MustExec("insert into match values (?, ?)", match.ID, match.Round)

		for _, game := range match.Games {
			db.MustExec("insert into game values (?, ?)", game.ID, match.ID)
			db.MustExec("insert into report values (?, ?, ?, ?)", ID("report"), game.ID, lists[game.Players[0]][game.Lists[0]].ID, game.Winner == 0)
			db.MustExec("insert into report values (?, ?, ?, ?)", ID("report"), game.ID, lists[game.Players[1]][game.Lists[1]].ID, game.Winner == 1)
		}
	}
}

func walk(node *html.Node, nodes []*html.Node) []*html.Node {
	var isPairing bool
	for _, attr := range node.Attr {
		if attr.Key == "class" && strings.Contains(attr.Val, "pairing-row") {
			isPairing = true
			break
		}
	}
	if isPairing {
		nodes = append(nodes, node)
		return nodes
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		nodes = walk(child, nodes)
	}

	return nodes
}

var counters = map[string]int{}

func ID(t string) int {
	counters[t]++
	return counters[t]
}
