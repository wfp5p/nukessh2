package nukedb

import (
    "database/sql"
	"time"
	"log"

    _ "github.com/mattn/go-sqlite3"
)

type NukeDB struct {
 	db *sql.DB
}

type NukeRecord struct {
	IPaddr string
	Expire int64
	Blocks int
	LastUpdate time.Time
}

func New(filename string) (*NukeDB, error) {
	var ndb NukeDB
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}

	ndb.db = db

	// create our table
	_, err = db.Exec(`create table if not exists nukessh
                      (ip text primary key,
                      expire integer default 0,
                      blocks integer default 0,
                      lastupdate TIMESTAMP)`)

	if err != nil {
		return nil, err
	}

	// sqllite needs a vacuum now and then
	_, err = db.Exec(`vacuum`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`create trigger if not exists mktime_insert after insert
                      on nukessh begin update nukessh
                      set lastupdate=strftime('%s','now') where
                      ip = new.ip;end;`)

	if err != nil {
		return nil, err
	}

    _, err = db.Exec(`create trigger if not exists mktime_update after update on nukessh begin
                      update nukessh set lastupdate=strftime('%s','now') where
                      ip = new.ip;end;`)

	if err != nil {
		return nil, err
	}


	return &ndb, nil
}

func (n NukeDB) Insert(ip string, expire int64, blocks int) error {
	stmt, err := n.db.Prepare(`insert or replace into nukessh(ip, expire, blocks) values(?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(ip, expire, blocks)
	if err != nil {
		return err
	}

	return nil
}

// insert the expire, if the row is already there, increment the block
func (n NukeDB) InsertExpire(ip string, expire int64) error {
	// get row
	r, err := n.GetInfo(ip)

	if err == sql.ErrNoRows {
		r.Blocks = 1
	} else if err == nil {
		r.Blocks++
	} else {
		return err
	}

	r.IPaddr = ip
	r.Expire = expire

	return n.Insert(r.IPaddr, r.Expire, r.Blocks)
}

// given an ip, return it's record
func (n NukeDB) GetInfo(ip string) (NukeRecord, error) {
	var r NukeRecord
	err := n.db.QueryRow(`select expire, blocks, lastupdate from
            nukessh where ip=?`,ip).Scan(&r.Expire, &r.Blocks, &r.LastUpdate)
	if err != nil {
		return r, err
	}
	r.IPaddr = ip
	return r, nil
}

// Purge records where expire = 0 and lastupdate <= param
func (n NukeDB) Purge(purgetime time.Time) error {
	stmt, err := n.db.Prepare(`delete from nukessh where expire=0 and lastupdate <= ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(purgetime)
	if err != nil {
		return err
	}

	_, err = n.db.Exec(`vacuum`)
	return nil
}

// Set expire to 0 for ip
func (n NukeDB) ClearExpire(ip string) error {
	stmt, err := n.db.Prepare(`update nukessh set expire=0 where ip=?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(ip)
	return err
}

// returns slice of ips where expire > 0 and <= expire param
func (n NukeDB) GetExpires(expire int64) ([]string, error) {
	var r []string

	rows, err := n.db.Query(`select ip from nukessh where expire > 0 and expire <= ?`, expire)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ip string
	for rows.Next() {
		err := rows.Scan(&ip)
		if err != nil {
			log.Fatal(err) // probably shouldn't be a fatal
		}
		r = append(r, ip)
	}

	err = rows.Err()
	if err != nil {
			log.Fatal(err) // probably shouldn't be a fatal
	}

	return r, nil
}

// returns slice of ips where expire != 0 and expire >  expire param
func (n NukeDB) GetActive(expire int64) ([]string, error) {
	var r []string

	rows, err := n.db.Query(`select ip from nukessh where expire != 0 and expire > ?`, expire)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ip string
	for rows.Next() {
		err := rows.Scan(&ip)
		if err != nil {
			log.Fatal(err) // probably shouldn't be a fatal
		}
		r = append(r, ip)
	}

	err = rows.Err()
	if err != nil {
			log.Fatal(err) // probably shouldn't be a fatal
	}

	return r, nil
}

func (n NukeDB) Close() error {
	return n.db.Close()
}
