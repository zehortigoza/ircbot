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
	"strings"

	"github.com/sorcix/irc"
)

type Oka struct {
}

func NewOka() (*Oka, error) {
	return &Oka{}, nil
}

func (o *Oka) HandleHelp() string {
	return "%oka"
}

func (o *Oka) HandleMessage(conn *Conn, m *irc.Message) {
	msg := AcceptPRIVMSG(m)
	if msg == nil || msg.channel == "" || !strings.HasPrefix(msg.content, "%oka") {
		return
	}
	log.Printf("oka was requested by %s", m.Prefix.Name)
	say(conn, m.Params[0], "valeu")
}
