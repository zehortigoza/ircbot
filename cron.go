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
	"strconv"
	"sync"
	"time"

	"github.com/sorcix/irc"
)

type CronJob struct {
	id uint32
	min, hour, day, month, weekDay int
	msg string
	dest string
	owner string
}

type Cron struct {
	ircConn *Conn
	jobs []CronJob
	jobId uint32
	mutex *sync.Mutex
	timezone *time.Location
	admin string
}

func cronWork(cron *Cron) {
	for {
		now := time.Now()
		if now.Second() != 0 {
			time.Sleep(time.Second)
			continue
		}

		now = now.In(cron.timezone)

		cron.mutex.Lock()
		for _, job := range cron.jobs {
			if job.min != -1 && job.min != now.Minute() {
				continue
			}
			if job.hour != -1 && job.hour != now.Hour() {
				continue
			}
			if job.day != -1 && job.day != now.Day() {
				continue
			}
			if job.month != -1 && job.month != int(now.Month()) {
				continue
			}
			if job.weekDay != -1 && job.weekDay != int(now.Weekday()) {
				continue
			}

			if strings.HasPrefix(job.dest, "#") {
				say(cron.ircConn, job.dest, fmt.Sprintf("Cron message from %s: %s\n", job.owner, job.msg))
			} else {
				say(cron.ircConn, job.dest, fmt.Sprintf("Cron message: %s\n", job.msg))
			}
		}
		cron.mutex.Unlock()

		time.Sleep(time.Second)
	}
}

func NewCron(conn *Conn, timezone string, admin string) (*Cron, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}

	var cron Cron
	cron.ircConn = conn
	cron.mutex = &sync.Mutex{}
	cron.timezone = loc
	cron.admin = admin

	go cronWork(&cron)
	return &cron, nil
}

func (c *Cron) HandleHelp() string {
	return "%cron add <minute 0-59> <hour 0-23> <day of month 1-31> <month 1-12> <day of week 0-6)> <message> | %cron list | %cron del <id>"
}

func (c *Cron) CronJobInfo(job *CronJob) string {
	return fmt.Sprintf("{ id: %d min: %d hour: %d day: %d month: %d wday: %d dest: %s owner: %s msg: %s }", job.id, job.min, job.hour, job.day, job.month, job.weekDay, job.dest, job.owner, job.msg)
}

func (c *Cron) HandleList(dest string) {
	say(c.ircConn, dest, "Cron jobs:\n\n")
	c.mutex.Lock()
	for _, job := range c.jobs {
		// only print jobs of the current channel current private chat
		// admin is the only one that can print jobs from all channels from a private chat
		// TODO check if src/dest is authenticated: /msg NickServ acc <nick>
		if (job.dest == dest || (dest == c.admin && strings.HasPrefix(job.dest, "#"))) {
			say(c.ircConn, dest, fmt.Sprintf("    %s\n", c.CronJobInfo(&job)))
		}
	}
	c.mutex.Unlock()
}

func (c *Cron) HandleDel(dest string, parts []string, msg *Msg) {
	v, err := strconv.Atoi(parts[2]);
	if (err != nil || v < 0) {
		say(c.ircConn, dest, fmt.Sprintf("Invalid cron job id %s\n", parts[2]))
		return
	}

	found := false
	c.mutex.Lock()
	for i := 0; i < len(c.jobs); i++ {
		job := c.jobs[i]

		if (job.id != uint32(v)) {
			continue
		}

		found = true
		// only owner of job or admin can delete a job
		// TODO check if src is authenticated: /msg NickServ acc <nick>
		if (msg.src == job.owner  || msg.src == c.admin) {
			c.jobs = append(c.jobs[:i], c.jobs[i+1:]...)
			say(c.ircConn, dest, "Cron job deleted\n")
		} else {
			say(c.ircConn, dest, "You don't have the rights to delete this cron job\n")
		}
		break
	}
	c.mutex.Unlock()

	if found == false {
		say(c.ircConn, dest, fmt.Sprintf("Cron job with id %d not found\n", v))
	}
}

func (c *Cron) stringToCronInt(dest string, part string, val *int) bool {
	if part == "*" {
		*val = -1
	} else {
		v, err := strconv.Atoi(part);
		if err != nil {
			say(c.ircConn, dest, fmt.Sprintf("Invalid number: %s\nUsage: %s\n", part, c.HandleHelp()))
			return false
		}
		*val = v
	}

	return true
}

func (c *Cron) HandleAdd(dest string, parts []string, msg *Msg) {
	if len(parts) < 8 {
		say(c.ircConn, dest, fmt.Sprintf("Usage: %s\n", c.HandleHelp()))
		return
	}

	var job CronJob

	if (!c.stringToCronInt(dest, parts[2], &job.min)) {
		return
	}
	if (job.min != -1 && (job.min < 0 || job.min > 59)) {
		say(c.ircConn, dest, fmt.Sprintf("Invalid minute: %d\nUsage: %s\n", job.min, c.HandleHelp()))
		return
	}

	if (!c.stringToCronInt(dest, parts[3], &job.hour)) {
		return
	}
	if (job.hour != -1 && (job.hour < 0 || job.hour > 23)) {
		say(c.ircConn, dest, fmt.Sprintf("Invalid hour: %d\nUsage: %s\n", job.hour, c.HandleHelp()))
		return
	}

	if (!c.stringToCronInt(dest, parts[4], &job.day)) {
		return
	}
	if (job.day != -1 && (job.day < 1 || job.day > 31)) {
		say(c.ircConn, dest, fmt.Sprintf("Invalid day: %d\nUsage: %s\n", job.day, c.HandleHelp()))
		return
	}

	if (!c.stringToCronInt(dest, parts[5], &job.month)) {
		return
	}
	if (job.month != -1 && (job.month < 1 || job.month > 12)) {
		say(c.ircConn, dest, fmt.Sprintf("Invalid month: %d\nUsage: %s\n", job.month, c.HandleHelp()))
		return
	}

	if (!c.stringToCronInt(dest, parts[6], &job.weekDay)) {
		return
	}
	if (job.weekDay != -1 && (job.weekDay < 0 || job.weekDay > 6)) {
		say(c.ircConn, dest, fmt.Sprintf("Invalid week day: %d\nUsage: %s\n", job.weekDay, c.HandleHelp()))
		return
	}

	job.id = c.jobId
	c.jobId++
	job.msg = parts[7]
	// TODO check if src is authenticated: /msg NickServ acc <nick>
	job.owner = msg.src
	job.dest = dest

	say(c.ircConn, dest, fmt.Sprintf("Cron job created %s", c.CronJobInfo(&job)))

	c.mutex.Lock()
	c.jobs = append(c.jobs, job)
	c.mutex.Unlock()
}

func (c *Cron) HandleMessage(conn *Conn, m *irc.Message) {
	msg := AcceptPRIVMSG(m)
	if msg == nil {
		return
	}

	var dest string
	if (msg.channel != "") {
		dest = msg.channel
	} else {
		dest = msg.src
	}

	parts := strings.SplitN(msg.content, " ", 8)
	if parts[0] != "%cron" {
		return
	}

	if len(parts) >= 2 && parts[1] == "list" {
		c.HandleList(dest)
		return
	}

	if len(parts) >= 3 && parts[1] == "del" {
		c.HandleDel(dest, parts, msg)
		return
	}

	if len(parts) >= 3 && parts[1] == "add" {
		c.HandleAdd(dest, parts, msg)
		say(c.ircConn, "nickserv", "zehortigoza2")
		return
	}

	say(conn, dest, fmt.Sprintf("Usage: %s\n", c.HandleHelp()))
}
