// Copyright (c) 2018 ircbot authors
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
	"strings"
	"time"

	"github.com/sorcix/irc"
)

type WorldCup struct {
	timezone *time.Location
	start time.Time
	end time.Time
}

func NewWorldCup(timezone string) (*WorldCup, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}

	moscow_loc, _ := time.LoadLocation("Europe/Moscow")

	start_t := time.Date(2018, time.June, 14, 18, 0, 0, 0, moscow_loc)
	start_t = start_t.In(loc)
	end_t  := time.Date(2018, time.July, 15, 18, 0, 0 ,0, moscow_loc)
	end_t = end_t.In(loc)

	return &WorldCup{timezone: loc, start: start_t, end: end_t}, nil
}

func (w *WorldCup) HandleMessage(conn *Conn, m *irc.Message) {
	msg := AcceptPRIVMSG(m)
	if msg == nil || msg.channel == "" {
		return
	}

	content := strings.Fields(msg.content)
	if len(content) == 0 {
		return
	}

	if content[0] != "%worldcup" && content[0] != "%hexa" {
		return
	}

	now := time.Now()
	now = now.In(w.timezone)
	var text string

	if now.Before(w.start) {
		diff := w.start.Sub(now)
		minutes := int(diff.Minutes())
		hours := minutes / 60
		minutes = minutes % 60
		days := hours / 24
		hours = hours % 24

		if content[0] == "%hexa" {
			text = fmt.Sprintf("%d dias, %d horas e %d minutos para o hexa, vai Brasil \\o/\n", days, hours, minutes)
		} else {
			text = fmt.Sprintf("%d days, %d hours and %d minutes until the begining of 2018 World Cup\n", days, hours, minutes)
		}
	} else {
		if now.Before(w.end) {
			text = fmt.Sprintf("2018 World Cup is happening right now, more info http://www.fifa.com/worldcup/matches/index.html\n")
		} else {
			text = fmt.Sprintf("2018 World Cup has ended\n")
		}
	}

	say(conn, m.Params[0], text)
}
