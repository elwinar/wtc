create table team (
	id integer,
	country varchar(20),
	name varchar(20)
);

create table player (
	id integer,
	name varchar(50),
	team_id integer,
	faction varchar(15)
);

create table list (
	id integer,
	caster varchar(15),
	player_id integer
);

create table match (
	id integer,
	round_number integer
);

create table match_list (
	match_id integer,
	list_id integer,
	win boolean
);
