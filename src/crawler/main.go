package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"logger"

	"golang.org/x/net/html"
)

const (
	URLTemplate = "http://wmh-wtc.com/?round=%d"
	Rounds      = 6
)

var (
	log = logger.New(os.Stderr).With(logger.M{
		"app": "crawler",
	})
	output = flag.String("out", "-", "output file")
	silent = flag.Bool("silent", false, "suppress output")
)

type (
	Page struct {
		Round int
		Body  io.Reader
	}

	MatchNode struct {
		Round int
		Root  *html.Node
	}

	Match struct {
		Round int
		Zone  string
		Teams [2]string
		Games [5]Game
	}

	Game struct {
		Players [2]string
		Lists   [2]string
		Winner  int
	}
)

func main() {
	flag.Parse()

	if *silent {
		log.SetOutput(ioutil.Discard)
	}

	var out io.Writer = os.Stdout
	if *output != "-" {
		file, err := os.Create(*output)
		if err != nil {
			log.Error("creating output file", logger.M{
				"path": *output,
				"err":  err,
			})
			return
		}

		out = file
	}

	var pages = make(chan Page)
	go func() {
		var URL string
		for i := 1; i <= Rounds; i++ {
			URL = fmt.Sprintf(URLTemplate, i)
			log.Info("retrieving page", logger.M{
				"round": i,
				"url":   URL,
			})
			res, err := http.Get(URL)
			if err != nil {
				log.Error("retrieving page", logger.M{
					"round": i,
					"err":   err,
				})
				continue
			}
			pages <- Page{
				Round: i,
				Body:  res.Body,
			}
		}
		close(pages)
	}()

	var nodes = make(chan MatchNode)
	go func() {
		for page := range pages {
			log.Info("parsing page", logger.M{
				"round": page.Round,
			})
			root, err := html.Parse(page.Body)
			if err != nil {
				log.Error("parsing page", logger.M{
					"round": page.Round,
					"err":   err,
				})
				continue
			}

			for _, node := range walk(root, nil) {
				nodes <- MatchNode{
					Round: page.Round,
					Root:  node,
				}
			}
		}
		close(nodes)
	}()

	var matches = make(chan Match)
	go func() {
		for node := range nodes {
			var match = Match{
				Round: node.Round,
				Zone:  strings.TrimSpace(node.Root.FirstChild.LastChild.FirstChild.Data),
			}

			log.Info("extracting match", logger.M{
				"round": node.Round,
				"zone":  match.Zone,
			})

			for i, teamNode := range []*html.Node{
				node.Root.FirstChild.NextSibling,
				node.Root.FirstChild.NextSibling.NextSibling.NextSibling,
			} {
				match.Teams[i] = strings.TrimSpace(teamNode.FirstChild.LastChild.FirstChild.Data[len("Team"):])
			}

			var g int
			for gameNode := node.Root.LastChild.FirstChild; gameNode != nil; gameNode = gameNode.NextSibling {
				var game = Game{
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

			matches <- match
		}
		close(matches)
	}()

	var encoder = json.NewEncoder(out)
	for match := range matches {
		log.Info("writing match", logger.M{
			"round": match.Round,
			"zone":  match.Zone,
		})
		err := encoder.Encode(match)
		if err != nil {
			log.Error("writing retrieved match", logger.M{
				"match": match,
				"err":   err,
			})
			continue
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
