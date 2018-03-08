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
	"crypto/tls"
	"database/sql"
	"flag"
	"io"
	"log"
	"net"
	"strings"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sorcix/irc"
)

var (
	serverAddr = flag.String("server", "localhost:6667", "server to connect")
	useTLS     = flag.Bool("tls", false, "use TLS to connect")
	insecure   = flag.Bool("insecure", false, "don't verify the certificate when using TLS")
	nickname   = flag.String("nick", "bot", "nickname used by the bot")
	channels   = flag.String("channels", "#test", "list of channels, comma separated")
	dbFilename = flag.String("db", "bot.db", "database file used by commands")
	verbose    = flag.Bool("v", false, "verbose mode")
	timezone   = flag.String("timezone", "America/Los_Angeles", "Timezone")
)

type Handler interface {
	HandleMessage(conn *Conn, m *irc.Message)
	HandleHelp() string
}

type HandlerFunc func(conn *Conn, m *irc.Message)

func (f HandlerFunc) HandleMessage(conn *Conn, m *irc.Message) {
	f(conn, m)
}

func mainHandleMessage(conn *Conn, m *irc.Message, handlers []Handler) {
	msg := AcceptPRIVMSG(m)
	if msg == nil || msg.channel == "" {
		return
	}

	content := strings.Fields(msg.content)
	if len(content) == 0 {
		return
	}

	if content[0] != "%help" {
		return
	}

	say(conn, m.Params[0], "Commands available:\n")
	for _, h := range handlers {
		help := h.HandleHelp()
		if len(help) > 0 {
			say(conn, m.Params[0], fmt.Sprintf("    %s\n", help))
		}
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("")
	flag.Parse()

	db, err := sql.Open("sqlite3", *dbFilename)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var handlers []Handler
	add := func(h Handler) {
		handlers = append(handlers, h)
	}

	//
	// Create and add the handlers.
	//

	janitor, err := NewJanitor()
	if err != nil {
		log.Fatal(err)
	}
	add(janitor)

	oka, err := NewOka()
	if err != nil {
		log.Fatal(err)
	}
	add(oka)

	tell, err := NewTell(db)
	if err != nil {
		log.Fatal(err)
	}
	add(tell)

	karma, err := NewKarma(db)
	if err != nil {
		log.Fatal(err)
	}
	add(karma)

	ai, err := NewAI()
	if err != nil {
		log.Fatal(err)
	}
	add(ai)

	worldcup, err := NewWorldCup(*timezone)
	if err != nil {
		log.Fatal(err)
	}
	add(worldcup)

	//
	// Connect to the server.
	//

	var c io.ReadWriteCloser
	if *useTLS {
		var cfg *tls.Config
		if *insecure {
			log.Println("NOTE: skipping certificate verification!")
			cfg = &tls.Config{InsecureSkipVerify: true}
		}
		c, err = tls.Dial("tcp", *serverAddr, cfg)
	} else {
		c, err = net.Dial("tcp", *serverAddr)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	conn := &Conn{
		Encoder: irc.NewEncoder(c),
		Decoder: irc.NewDecoder(c),
		conn:    c,
	}

	conn.Encode(&irc.Message{
		Command: irc.NICK,
		Params:  []string{*nickname},
	})
	conn.Encode(&irc.Message{
		Command:  irc.USER,
		Params:   []string{*nickname, "0", "*"},
		Trailing: *nickname,
	})

	//
	// Process messages.
	//

	// TODO: Give the handlers a channel so they can asynchronously send messages.

	// TODO: Handle when we get disconnected.
	for {
		m, err := conn.Decode()
		if err != nil {
			log.Fatal(err)
		}
		if *verbose {
			log.Print(m)
		}
		mainHandleMessage(conn, m, handlers)
		for _, h := range handlers {
			h.HandleMessage(conn, m)
		}
	}
}

type Conn struct {
	*irc.Encoder
	*irc.Decoder
	conn io.ReadWriteCloser
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

// TODO: Make a say that takes varargs, so can use as Printf.
func say(conn *Conn, to string, msg string) {
	conn.Encode(&irc.Message{
		Command:  irc.PRIVMSG,
		Params:   []string{to},
		Trailing: msg,
	})
}

type Msg struct {
	channel string
	src     string
	content string
}

type UserJoin struct {
    user    string
    channel string
}

func AcceptPRIVMSG(m *irc.Message) *Msg {
	if m.Command != irc.PRIVMSG {
		return nil
	}
	// TODO: Remember what case is this covering.
	if len(m.Params) == 0 {
		return nil
	}
	msg := &Msg{
		src:     m.Prefix.Name,
		content: m.Trailing,
	}
	if strings.HasPrefix(m.Params[0], "#") {
		msg.channel = m.Params[0]
	}
	return msg
}

func AcceptJoin(m *irc.Message) *UserJoin {
    if m.Command != irc.JOIN {
        return nil
    }
    join := &UserJoin{
        user: m.Prefix.Name,
    }
    if strings.HasPrefix(m.Trailing, "#") {
        join.channel = m.Trailing
    }

    return join
}
