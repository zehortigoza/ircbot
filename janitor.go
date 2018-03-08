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
	"log"

	"github.com/sorcix/irc"
)

type Janitor struct {
}

func NewJanitor() (*Janitor, error) {
	return &Janitor{}, nil
}

func (j *Janitor) HandleHelp() string {
	return ""
}

// TODO: Handle when kicked.
func (j *Janitor) HandleMessage(conn *Conn, m *irc.Message) {
	switch m.Command {
	case irc.RPL_WELCOME:
		log.Printf("Server said welcome, joining channels: %s", *channels)
		conn.Encode(&irc.Message{
			Command: irc.JOIN,
			Params:  []string{*channels},
		})
	case irc.PING:
		conn.Encode(&irc.Message{
			Command:  irc.PONG,
			Params:   []string{},
			Trailing: m.Trailing,
		})
	}
}
