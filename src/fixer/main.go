package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"logger"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var (
	log = logger.New(os.Stderr).With(logger.M{
		"app": "fixer",
	})
	input  = flag.String("in", "-", "input file")
	output = flag.String("out", "-", "output file")
	silent = flag.Bool("silent", false, "suppress output")
)

type (
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

	var in io.Reader = os.Stdin
	if *input != "-" {
		file, err := os.Open(*input)
		if err != nil {
			log.Error("opening input file", logger.M{
				"path": *input,
				"err":  err,
			})
			return
		}
		in = file
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

	var decoder = json.NewDecoder(in)
	var matches = make(chan Match)
	go func() {
		for decoder.More() {
			var match Match
			err := decoder.Decode(&match)
			if err != nil {
				log.Error("reading match", nil)
				continue
			}

			matches <- match
		}
		close(matches)
	}()

	var fixedMatches = make(chan Match)
	go func() {
		for match := range matches {
			for g, game := range match.Games {
				for l := 0; l <= 1; l++ {
					if game.Lists[l] == "vHarkevich 1" {
						match.Games[g].Lists[l] = "Harkevich 1"
					}
				}
			}
			fixedMatches <- match
		}
		close(fixedMatches)
	}()

	var encoder = json.NewEncoder(out)
	for match := range fixedMatches {
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
