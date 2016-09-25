package main

import (
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"strings"
)

func main() {
	res, err := http.Get("http://wmh-wtc.com/?round=1")
	if err != nil {
		fmt.Println(err)
		return
	}

	root, err := html.Parse(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	var pairings = make(chan *html.Node)
	go func() {
		walk(root, pairings)
		close(pairings)
	}()

	for pairing := range pairings {
		fmt.Println(pairing)
	}
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
