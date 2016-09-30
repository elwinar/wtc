package main

import (
	"flag"
	"fmt"
	"logger"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var (
	log = logger.New(os.Stderr).With(logger.M{
		"app": "checker",
	})
	database = flag.String("db", "data.sqlite", "database file")
)

func main() {
	flag.Parse()

	db, err := sqlx.Connect("sqlite3", *database)
	if err != nil {
		log.Error("opening database", logger.M{
			"path": *database,
			"err":  err,
		})
		return
	}

	// Find casters whose name in wrong
	var casters = make([]string, 0, len(CastersFactions))
	for caster, _ := range CastersFactions {
		casters = append(casters, caster)
	}
	query, args, _ := sqlx.In(`
		select distinct
			name,
			round,
			zone,
			caster
		from list
		join player on player.id = list.player_id
		join report on report.list_id = list.id
		join game on game.id = report.game_id
		join match on match.id = game.match_id
		where caster not in (?)
		order by caster
	`, casters)

	var typos []struct {
		Name   string
		Round  int
		Zone   int
		Caster string
	}
	err = db.Select(&typos, query, args...)
	if err != nil {
		log.Error("unable to get typoed casters", logger.M{
			"err": err,
		})
		return
	}

	for _, typo := range typos {
		fmt.Println(typo)
	}
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

var CastersFactions = map[string]string{
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
	"Butcher 1":   Khador,
	"Butcher 3":   Khador,
	"Vladimir 1":  Khador,
	"Vladimir 2":  Khador,
	"Vladimir 3":  Khador,
	"Harkevich 1": Khador,
	"Irusk 2":     Khador,
	"Karchev 1":   Khador,
	"Sorscha 1":   Khador,
	"Strakhov 1":  Khador,

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
