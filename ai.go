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
	"fmt"
	"math/rand"
	"strings"

	"github.com/sorcix/irc"
)

type AI struct {
}

func NewAI() (*AI, error) {
	return &AI{}, nil
}

func (t *AI) HandleHelp() string {
	return "<bot_name>: <option 1> or <option 2>?"
}

func handleOr(conn *Conn, msg *Msg, m *irc.Message, separator string) bool {
	if !strings.Contains(m.Trailing, separator) {
		return false
	}

	options := strings.SplitN(m.Trailing, separator, 2)
	if len(options) != 2 {
		return false
	}

	// TODO: Use TensorFlow or IBM Watson for natural language analysis.
	// It's more accurate than rand.Int().
	option := strings.TrimSpace(options[rand.Int()%2])
	say(conn, msg.channel, fmt.Sprintf("%s: %s!", m.Name, option))
	return true
}

func trimNickPrefix(msg, nickSep string) (string, bool) {
	nick := fmt.Sprintf("%s%s", *nickname, nickSep)

	if !strings.HasPrefix(msg, nick) {
		return msg, false
	}

	return strings.TrimPrefix(msg, nick), true
}

func (t *AI) HandleMessage(conn *Conn, m *irc.Message) {
	msg := AcceptPRIVMSG(m)
	if msg == nil || msg.channel == "" {
		return
	}

	if !strings.HasSuffix(m.Trailing, "?") {
		return
	}
	m.Trailing = strings.TrimRight(m.Trailing, "?")

	withoutPrefix, hadPrefix := trimNickPrefix(m.Trailing, ", ")
	if !hadPrefix {
		withoutPrefix, hadPrefix = trimNickPrefix(m.Trailing, ": ")
	}
	if !hadPrefix {
		withoutPrefix, hadPrefix = trimNickPrefix(m.Trailing, " ")
	}
	if !hadPrefix {
		return
	}
	m.Trailing = withoutPrefix

	m.Trailing = strings.TrimSpace(m.Trailing)

	if handleOr(conn, msg, m, " ou ") {
		return
	}
	if handleOr(conn, msg, m, " or ") {
		return
	}
	if handleOr(conn, msg, m, "||") {
		return
	}
}
