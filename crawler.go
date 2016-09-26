package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"golang.org/x/net/html"
)

type Team struct {
	ID   int
	Name string
}

func main() {
	pages := fetch()
	nodes := parse(pages)
	teams := extract(nodes)

	for team := range teams {
		fmt.Println(team.Name)
	}
}

func fetch() chan io.Reader {
	var out = make(chan io.Reader)

	go func() {
		for i := 1; i < 7; i++ {
			res, err := http.Get(fmt.Sprintf("http://wmh-wtc.com/?round=%d", i))
			if err != nil {
				log.Println(err)
				continue
			}

			out <- res.Body
		}
		close(out)
	}()

	return out
}

func parse(pages chan io.Reader) chan *html.Node {
	var out = make(chan *html.Node)

	var walk func(node *html.Node, out chan *html.Node)
	walk = func(node *html.Node, out chan *html.Node) {
		var isPairing bool
		for _, attr := range node.Attr {
			if attr.Key == "class" && strings.Contains(attr.Val, "pairing-row") {
				isPairing = true
				break
			}
		}
		if isPairing {
			out <- node
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child, out)
		}
	}

	go func() {
		for page := range pages {
			root, err := html.Parse(page)
			if err != nil {
				log.Println(err)
				continue
			}
			walk(root, out)
		}
		close(out)
	}()

	return out
}

var teams = map[string]Team{}

func extract(nodes chan *html.Node) chan Team {
	var out = make(chan Team)

	go func() {
		for node := range nodes {
			for _, teamNode := range []*html.Node{
				node.FirstChild.NextSibling,
				node.FirstChild.NextSibling.NextSibling.NextSibling,
			} {
				name := teamNode.FirstChild.LastChild.FirstChild.Data
				if _, found := teams[name]; !found {
					team := Team{
						ID:   ID("team"),
						Name: name,
					}

					teams[name] = team
					out <- team
				}
			}
		}
		close(out)
	}()

	return out
}

var counters = map[string]int{}

func ID(t string) int {
	counters[t]++
	return counters[t]
}
