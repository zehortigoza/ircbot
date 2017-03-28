// Copyright (c) 2017 ircbot authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/sorcix/irc"
)

type Tell struct {
	db *sql.DB
}

func NewTell(db *sql.DB) (*Tell, error) {
	sqlStmt := `
CREATE TABLE IF NOT EXISTS tell (id INTEGER NOT NULL PRIMARY KEY, ch TEXT, src TEXT, dst TEXT, msg TEXT, dt DATE, read BOOLEAN);
`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("couldn't create tell database: %s", err)
	}

	return &Tell{db: db}, nil
}

func (t *Tell) HandleMessage(conn *Conn, m *irc.Message) {
	// TODO: Should nick change count as activity?

	join := AcceptJoin(m)
	if join != nil && join.channel != "" {
		count := tellCount(t.db, join.channel, join.user)
		if count > 0 {
			say(conn, join.channel, fmt.Sprintf("%s: tenho %d mensagens deste canal para voce, quando voce falar algo aqui eu conto\n", join.user, count))
		}
	}

	msg := AcceptPRIVMSG(m)
	if msg == nil || msg.channel == "" {
		return
	}

	if strings.HasPrefix(m.Trailing, "%tell ") {
		text := strings.SplitN(m.Trailing, " ", 3)
		if len(text) < 3 {
			say(conn, msg.channel, "use %tell NICK MSG...")
		} else {
			err := tellInsert(t.db, msg.channel, msg.src, text[1], text[2])
			if err != nil {
				log.Println(err)
			}
			say(conn, msg.channel, fmt.Sprintf("%s: Done!\n", msg.src))
		}
	}
	tellCheck(conn, t.db, msg.channel, msg.src)
}

func tellCheck(conn *Conn, db *sql.DB, ch, name string) error {
	records, err := tellRead(db, ch, name)
	if err != nil {
		log.Fatal(err)
	}
	for _, r := range records {
		say(conn, ch, fmt.Sprintf("%s: %s said %s: %s\n", name, r.src, humanize.Time(r.dt), r.msg))
	}
	return nil
}

func tellInsert(db *sql.DB, ch, src, dst, msg string) error {
	_, err := db.Exec("INSERT INTO tell(ch, src, dst, msg, dt, read) VALUES(?, ?, ?, ?, ?, 0)", ch, src, dst, msg, time.Now())
	return err
}

type tellRecord struct {
	id                int
	ch, src, dst, msg string
	dt                time.Time
	read              bool
}

func tellCount(db *sql.DB, ch, dst string) int {
	rows, err := db.Query("SELECT count(id) FROM tell WHERE ch = ? AND read = 0 AND dst = ?", ch, dst)
	if err != nil {
		log.Println("couldnt query")
		return 0
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			log.Printf("error scanning tell count: %s", err)
		}
	}
	return count
}

func tellRead(db *sql.DB, ch, dst string) ([]*tellRecord, error) {
	rows, err := db.Query("SELECT id, ch, src, dst, msg, dt, read FROM tell WHERE ch = ? AND read = 0 AND dst = ? ORDER BY dt", ch, dst)
	if err != nil {
		log.Println("couldnt query")
		return nil, err
	}
	defer rows.Close()

	var result []*tellRecord
	for rows.Next() {
		r := &tellRecord{}
		err = rows.Scan(&r.id, &r.ch, &r.src, &r.dst, &r.msg, &r.dt, &r.read)
		if err != nil {
			log.Println("couldnt scan")
			return nil, err
		}
		result = append(result, r)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return result, nil
	}

	for _, r := range result {
		_, err = db.Exec("UPDATE tell SET read = 1 WHERE id = ?", r.id)
		if err != nil {
			log.Printf("couldn't mark tell message id=%d as read: %s", r.id, err)
		}
	}

	return result, nil
}
