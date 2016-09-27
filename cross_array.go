package main

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	type result struct {
		caster   string
		opponent string
		won      int
		played   int
	}

	db, err := sqlx.Connect("sqlite3", "wtc2016.sqlite")
	if err != nil {
		log.Println(err)
	}

	var raw []result
	err := db.Select(&raw, `
		select 
			l.caster,
			l2.caster, 
			count(case when r.won then 1 end) as won, 
			count(*) as played
		from 
			list as l 
			join report as r on l.id = r.list_id 
			join report as r2 on r.game_id = r2.game_id 
			join list as l2 on r2.list_id = l2.id 
		group by l.caster, l2.caster 
		order by played, won;
	`)
	if err != nil {
		log.Println(err)
	}

	var res map[string][string]result
}
