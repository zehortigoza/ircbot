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

	"github.com/sorcix/irc"
)

type Karma struct {
	db *sql.DB
}

func NewKarma(db *sql.DB) (*Karma, error) {
	sqlStmt := `
CREATE TABLE IF NOT EXISTS karma (word TEXT, src TEXT, dt DATE, value INT);
`
	_, err := db.Exec(sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("couldn't create karma database: %s", err)
	}
	return &Karma{db: db}, nil
}

func (k *Karma) HandleMessage(conn *Conn, m *irc.Message) {
	msg := AcceptPRIVMSG(m)
	if msg == nil || msg.channel == "" {
		return
	}

	content := strings.Fields(msg.content)
	if len(content) == 0 {
		return
	}

	if content[0] == "%karma" {
		if len(content) == 1 {
			say(conn, m.Params[0], "use %karma WORD...")
			return
		}
		for _, word := range content[1:] {
			say(conn, m.Params[0], fmt.Sprintf("%s has karma %d", word, k.Query(word)))
		}
		return
	}

	for _, word := range content {
		if len(word) < 3 {
			continue
		}
		switch {
		case strings.HasSuffix(word, "++"):
			k.Update(m.Prefix.Name, word[:len(word)-2], +1)
		case strings.HasSuffix(word, "--"):
			k.Update(m.Prefix.Name, word[:len(word)-2], -1)
		}
	}
}

func (k *Karma) Query(word string) int {
	rows, err := k.db.Query("SELECT COALESCE(SUM(value), 0) FROM karma WHERE word = ?", word)
	if err != nil {
		log.Printf("error querying karma: %s", err)
		return 0
	}
	defer rows.Close()

	var value int
	for rows.Next() {
		err = rows.Scan(&value)
		if err != nil {
			log.Printf("error scanning karma result: %s", err)
		}
	}

	err = rows.Err()
	if err != nil {
		log.Printf("error querying karma: %s", err)
	}
	return value

}

func (k *Karma) Update(src, word string, value int) {
	_, err := k.db.Exec("INSERT INTO karma(word, src, dt, value) VALUES(?, ?, ?, ?)", word, src, time.Now(), value)
	if err != nil {
		log.Printf("error updating karma for %s: %s", word, err)
	}
}
