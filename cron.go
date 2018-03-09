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
	// minute, hour, day, month, wday
	values [5]int
	msg string
	dest string
	owner string
}

type Cron struct {
	irc_conn *Conn
	jobs []CronJob
	job_id uint32
	mutex *sync.Mutex
	timezone *time.Location
	admin string
}

func cron_work(cron *Cron) {
	for true {
		now := time.Now()
		if now.Second() != 0 {
			time.Sleep(time.Second)
			continue
		}

		now = now.In(cron.timezone)

		cron.mutex.Lock()
		for _, job := range cron.jobs {
			if job.values[0] != -1 && job.values[0] != now.Minute() {
				continue
			}
			if job.values[1] != -1 && job.values[1] != now.Hour() {
				continue
			}
			if job.values[2] != -1 && job.values[2] != now.Day() {
				continue
			}
			if job.values[3] != -1 && job.values[3] != int(now.Month()) {
				continue
			}
			if job.values[4] != -1 && job.values[4] != int(now.Weekday()) {
				continue
			}

			if strings.HasPrefix(job.dest, "#") {
				say(cron.irc_conn, job.dest, fmt.Sprintf("%s said: %s\n", job.owner, job.msg))
			} else {
				say(cron.irc_conn, job.dest, job.msg)
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
	cron.irc_conn = conn
	cron.mutex = &sync.Mutex{}
	cron.timezone = loc
	cron.admin = admin

	go cron_work(&cron)
	return &cron, nil
}

func (c *Cron) HandleHelp() string {
	return "%cron <minute 0-59> <hour 0-23> <day of month 1-31> <month 1-12> <day of week 0-6)> <message> | %cron list | %cron del <id>"
}

func (c *Cron) CronJobInfo(job *CronJob) string {
	return fmt.Sprintf("{ id: %d min: %d hour: %d day: %d month: %d wday: %d dest: %s owner: %s msg: %s }", job.id, job.values[0], job.values[1], job.values[2], job.values[3], job.values[4], job.dest, job.owner, job.msg)
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

	parts := strings.SplitN(msg.content, " ", 7)
	if parts[0] != "%cron" {
		return
	}

	if len(parts) >= 2 && parts[1] == "list" {
		say(conn, dest, "Cron jobs:\n\n")
		c.mutex.Lock()
		for _, job := range c.jobs {
			// only print jobs of the current channel current private chat
			// admin is the only one that can print jobs from all channels from a private chat
			if (job.dest == dest || (dest == c.admin && strings.HasPrefix(job.dest, "#"))) {
				say(conn, dest, fmt.Sprintf("    %s\n", c.CronJobInfo(&job)))
			}
		}
		c.mutex.Unlock()
		return
	}

	if len(parts) >= 3 && parts[1] == "del" {
		v, err := strconv.Atoi(parts[2]);
		if (err != nil || v < 0) {
			say(conn, dest, fmt.Sprintf("Invalid cron job id %s\n", parts[2]))
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
			if (msg.src == job.owner  || msg.src == c.admin) {
				c.jobs = append(c.jobs[:i], c.jobs[i+1:]...)
				say(conn, dest, "Cron job deleted\n")
			} else {
				say(conn, dest, "You don't have the rights to delete this cron job\n")
			}
			break
		}
		c.mutex.Unlock()

		if found == false {
			say(conn, dest, fmt.Sprintf("Cron job with id %d not found\n", v))
		}

		return
	}

	if len(parts) < 7 {
		say(conn, dest, fmt.Sprintf("Usage: %s\n", c.HandleHelp()))
		return
	}

	var job CronJob

	for i := 1; i < 6; i++ {
		if (parts[i] == "*") {
			job.values[i - 1] = -1
		} else {
			v, err := strconv.Atoi(parts[i]);
			if err != nil {
				say(conn, dest, fmt.Sprintf("Invalid number: %s\nUsage: %s\n", parts[i], c.HandleHelp()))
				return
			}

			switch i {
			case 1:
				if (v < 0 || v > 59) {
					say(conn, dest, fmt.Sprintf("Invalid minute: %d\nUsage: %s\n", v, c.HandleHelp()))
					return
				}
				break
			case 2:
				if (v < 0 || v > 23) {
					say(conn, dest, fmt.Sprintf("Invalid hour: %d\nUsage: %s\n", v, c.HandleHelp()))
					return
				}
				break
			case 3:
				if (v < 1 || v > 31) {
					say(conn, dest, fmt.Sprintf("Invalid day: %d\nUsage: %s\n", v, c.HandleHelp()))
				}
				break
			case 4:
				if (v < 1 || v > 12) {
					say(conn, dest, fmt.Sprintf("Invalid month: %d\nUsage: %s\n", v, c.HandleHelp()))
					return
				}
				break
			case 5:
				if (v < 0 || v > 6) {
					say(conn, dest, fmt.Sprintf("Invalid week day: %d\nUsage: %s\n", v, c.HandleHelp()))
					return
				}
				break
			}

			job.values[i - 1] = v
		}
	}

	job.id = c.job_id
	c.job_id++
	job.msg = parts[6]
	job.owner = msg.src
	job.dest = dest

	say(conn, dest, fmt.Sprintf("Cron job created %s", c.CronJobInfo(&job)))

	c.mutex.Lock()
	c.jobs = append(c.jobs, job)
	c.mutex.Unlock()
}
