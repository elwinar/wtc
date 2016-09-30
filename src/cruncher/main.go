package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"logger"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
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

	Team struct {
		ID      int
		Name    string
		Country string
		Players [5]string
	}

	Player struct {
		ID     int
		Name   string
		TeamID int
		Lists  [2]string
	}

	List struct {
		ID       int
		Caster   string
		PlayerID int
	}
)

var (
	log = logger.New(os.Stderr).With(logger.M{
		"app": "cruncher",
	})
	input    = flag.String("in", "-", "input file")
	database = flag.String("db", "data.sqlite", "database file")
	silent   = flag.Bool("silent", false, "suppress output")
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

	db, err := sqlx.Connect("sqlite3", *database)
	if err != nil {
		log.Error("opening database", logger.M{
			"path": *database,
			"err":  err,
		})
		return
	}

	db.MustExec("create table team ( id integer primary key, name varchar(50), country varchar(50) )")
	db.MustExec("create table player ( id integer primary key, name varchar(50), faction varchar(50), team_id integer )")
	db.MustExec("create table list ( id integer primary key, caster varchar(50), player_id integer )")
	db.MustExec("create table match ( id integer primary key, round integer, zone integer )")
	db.MustExec("create table game ( id integer primary key, match_id integer )")
	db.MustExec("create table report ( id integer primary key, game_id integer, list_id integer, won boolean )")

	var decoder = json.NewDecoder(in)
	var matches = make(chan Match)
	go func() {
		for decoder.More() {
			var match Match
			err = decoder.Decode(&match)
			if err != nil {
				log.Error("reading match", nil)
				continue
			}

			matches <- match
		}
		close(matches)
	}()

	var teams = make(map[string]int)
	var players = make(map[string]int)
	var lists = make(map[string]map[string]int)
	for match := range matches {
		for i := 0; i <= 1; i++ {
			var team = match.Teams[i]
			if _, found := teams[team]; found {
				continue
			}

			var country, name string
			for _, c := range countries {
				if !strings.HasPrefix(team, c) {
					continue
				}

				country = c
				name = strings.TrimSpace(team[len(country):])
				break
			}

			if country == "" {
				log.Error("unable to parse team name", logger.M{
					"team": team,
				})
				continue
			}

			log.Info("inserting team", logger.M{
				"country": country,
				"name":    name,
			})
			res, err := db.Exec("insert into team (country, name) values (?, ?)", country, name)
			if err != nil {
				log.Error("inserting team", logger.M{
					"country": country,
					"name":    name,
					"err":     err,
				})
				continue
			}

			ID, _ := res.LastInsertId()
			teams[team] = int(ID)
		}

		for _, game := range match.Games {
			for i := 0; i <= 1; i++ {
				var player = game.Players[i]
				if _, found := players[player]; found {
					continue
				}

				var faction = factions[game.Lists[i]]

				log.Info("inserting player", logger.M{
					"name":    player,
					"faction": faction,
					"team_id": teams[match.Teams[i]],
				})
				res, err := db.Exec("insert into player (name, faction, team_id) values (?, ?, ?)", player, faction, teams[match.Teams[i]])
				if err != nil {
					log.Error("inserting player", logger.M{
						"name":    player,
						"faction": faction,
						"team_id": teams[match.Teams[i]],
						"err":     err,
					})
					continue
				}

				ID, _ := res.LastInsertId()
				players[player] = int(ID)
				lists[player] = map[string]int{}
			}
		}

		for _, game := range match.Games {
			for i := 0; i <= 1; i++ {
				var player = game.Players[i]
				var caster = game.Lists[i]

				if _, found := lists[player][caster]; found {
					continue
				}

				log.Info("inserting list", logger.M{
					"player": player,
					"caster": caster,
				})
				res, err := db.Exec("insert into list (caster, player_id) values (?, ?)", caster, players[player])
				if err != nil {
					log.Error("inserting list", logger.M{
						"player": player,
						"caster": caster,
						"err":    err,
					})
					continue
				}

				ID, _ := res.LastInsertId()
				lists[player][caster] = int(ID)
			}
		}

		log.Info("inserting match", logger.M{})
		res, err := db.Exec("insert into match (round, zone) values (?, ?)", match.Round, match.Zone)
		if err != nil {
			log.Error("inserting match", logger.M{
				"err": err,
			})
			continue
		}

		matchID, _ := res.LastInsertId()

		for _, game := range match.Games {
			log.Error("inserting game", logger.M{
				"match_id": matchID,
			})
			res, err := db.Exec("insert into game (match_id) values (?)", matchID)
			if err != nil {
				log.Error("inserting game", logger.M{
					"match_id": matchID,
					"err":      err,
				})
				continue
			}

			gameID, _ := res.LastInsertId()

			for i := 0; i <= 1; i++ {
				log.Info("inserting report", logger.M{
					"game_id": gameID,
					"list_id": lists[game.Players[i]][game.Lists[i]],
				})
				_, err = db.Exec("insert into report (game_id, list_id, won) values (?, ?, ?)", gameID, lists[game.Players[i]][game.Lists[i]], game.Winner == i)
				if err != nil {
					log.Error("inserting report", logger.M{
						"game_id": gameID,
						"list_id": lists[game.Players[i]][game.Lists[i]],
						"err":     err,
					})
					continue
				}
			}
		}
	}
}

var countries = []string{
	"Australia",
	"Austria",
	"Belgium",
	"Canada",
	"China",
	"Czech Republic",
	"Denmark",
	"England",
	"Finland",
	"France",
	"Germany",
	"Greece",
	"Hungary",
	"Ireland",
	"Italy",
	"Latvia",
	"Middle East",
	"Netherlands",
	"Northern Ireland",
	"Norway",
	"Poland",
	"Portugal",
	"Russia",
	"Scotland",
	"Slovenia",
	"Spain",
	"Sweden",
	"Switzerland",
	"UAE",
	"USA",
	"Wales",
}

const (
	Cygnar      = "cygnar"
	Cryx        = "cryx"
	Menoth      = "menoth"
	Khador      = "khador"
	Mercenaries = "mercenaries"
	Cyriss      = "cyriss"
	Scyrah      = "scyrah"
	Trollbloods = "trollbloods"
	Orboros     = "orboros"
	Everblight  = "everblight"
	Skorne      = "skorne"
	Minion      = "minion"
)

var factions = map[string]string{
	// Everblight
	"Absylonia 2":        Everblight,
	"Kallus 1":           Everblight,
	"Lylyth 1":           Everblight,
	"Lylyth 3":           Everblight,
	"Rhyas 1":            Everblight,
	"Saeryn 2 & Rhyas 2": Everblight,
	"Thagrosh 1":         Everblight,
	"Thagrosh 2":         Everblight,
	"Vayl 1":             Everblight,
	"Vayl 2":             Everblight,

	// Cryx
	"Agathia 1":     Cryx,
	"Asphyxious 3":  Cryx,
	"Deneghra 1":    Cryx,
	"Goreshade 1":   Cryx,
	"Goreshade 2":   Cryx,
	"Mortenebra 1":  Cryx,
	"Scaverous 1":   Cryx,
	"Skarre 1":      Cryx,
	"Skarre 2":      Cryx,
	"Terminus 1":    Cryx,
	"Venethrax 1":   Cryx,
	"Witch coven 1": Cryx,

	// Menoth
	"Amon 1":           Menoth,
	"Durst 1":          Menoth,
	"Harbinger 1":      Menoth,
	"High Reclaimer 1": Menoth,
	"High Reclaimer 2": Menoth,
	"Kreoss 1":         Menoth,
	"Kreoss 3":         Menoth,
	"Malekus 1":        Menoth,
	"Reznik 1":         Menoth,
	"Reznik 2":         Menoth,
	"Severius 1":       Menoth,
	"Severius 2":       Menoth,
	"Thyra 1":          Menoth,
	"Vindictus 1":      Menoth,

	// Minion
	"Arkadius 1":      Minion,
	"Barnabas 1":      Minion,
	"Carver 1":        Minion,
	"Maelok 1":        Minion,
	"Rask 1":          Minion,
	"Sturm & Drang 1": Minion,

	// Cyriss
	"Aurora 1":      Cyriss,
	"Axis 1":        Cyriss,
	"Directrix 1":   Cyriss,
	"Iron Mother 1": Cyriss,
	"Lucant 1":      Cyriss,

	// Orboros
	"Baldur 1":   Orboros,
	"Baldur 2":   Orboros,
	"Grayle 1":   Orboros,
	"Kaya 2":     Orboros,
	"Kromac 1":   Orboros,
	"Kromac 2":   Orboros,
	"Krueger 1":  Orboros,
	"Tanith 1":   Orboros,
	"Wurmwood 1": Orboros,

	// Trollbloods
	"Borka 1":      Trollbloods,
	"Borka 2":      Trollbloods,
	"Calandra 1":   Trollbloods,
	"Doomshaper 1": Trollbloods,
	"Doomshaper 2": Trollbloods,
	"Doomshaper 3": Trollbloods,
	"Grim 2":       Trollbloods,
	"Grissel 2":    Trollbloods,
	"Gunnbjorn 1":  Trollbloods,
	"Madrak 2":     Trollbloods,
	"Ragnor 1":     Trollbloods,
	"Skuld 1":      Trollbloods,

	// Khador
	"Butcher 1":    Khador,
	"Butcher 3":    Khador,
	"Vladimir 1":   Khador,
	"Vladimir 2":   Khador,
	"Vladimir 3":   Khador,
	"Harkevich 1":  Khador,
	"Irusk 2":      Khador,
	"Karchev 1":    Khador,
	"Sorscha 1":    Khador,
	"Strakhov 1":   Khador,
	"vHarkevich 1": Khador,

	// Cygnar
	"Caine 1":   Cygnar,
	"Caine 2":   Cygnar,
	"Darius 1":  Cygnar,
	"Haley 1":   Cygnar,
	"Haley 2":   Cygnar,
	"Haley 3":   Cygnar,
	"Maddox 1":  Cygnar,
	"Nemo 1":    Cygnar,
	"Nemo 3":    Cygnar,
	"Siege 1":   Cygnar,
	"Sloan 1":   Cygnar,
	"Stryker 1": Cygnar,
	"Stryker 2": Cygnar,

	// Mercenaries
	"Cyphon 1":   Mercenaries,
	"Damiano 1":  Mercenaries,
	"Gorten 1":   Mercenaries,
	"MacBain 1":  Mercenaries,
	"Magnus 2":   Mercenaries,
	"Montador 1": Mercenaries,
	"Thexus 1":   Mercenaries,

	// Scyrah
	"Helynna 1":  Scyrah,
	"Issyria 1":  Scyrah,
	"Kaelyssa 1": Scyrah,
	"Ossrum 1":   Scyrah,
	"Ossyan 1":   Scyrah,
	"Rahn 1":     Scyrah,
	"Ravyn 1":    Scyrah,
	"Vyros 1":    Scyrah,
	"Vyros 2":    Scyrah,

	// Skorne
	"Hexeris 2":   Skorne,
	"Makeda 2":    Skorne,
	"Mordikaar 1": Skorne,
	"Morghoul 1":  Skorne,
	"Naaresh 1":   Skorne,
	"Rasheth 1":   Skorne,
	"Xerxis 1":    Skorne,
	"Zaal 1":      Skorne,
}
