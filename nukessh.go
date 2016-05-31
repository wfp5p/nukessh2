package main

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
)

type SshLogin struct {
	IPaddr string
	User string
}

type LineInfo chan string

func (w LineInfo) Write(p []byte) (int, error) {
	if len(p) < 1 {
		return 0, nil
	}

	w <- string(p)
	return len(p), nil
}

func (w LineInfo) Close() error {
	close(w)
	return nil
}

const (
	expirecycle = time.Hour
	decay = 10
)

var (
	rx *regexp.Regexp
	badusers = make(map[string]bool)
)

func init() {
	// This is the /var/log/secure version of the regexp
	// rx = regexp.MustCompile(`sshd\[\d+\]:\s+Failed password for (?:invalid\s+user\s+)?(.*) from (\d+\.\d+\.\d+\.\d+)\s+port`)

	// This is the systemd version of the regexp
	rx = regexp.MustCompile(`MESSAGE=Failed password for (?:invalid\s+user\s+)?(.*) from (\d+\.\d+\.\d+\.\d+)\s+port`)

	for _, u := range []string{
		"admin",
		"administrator",
		"anaconda",
		"apache",
		"bin",
		"bugzilla",
		"cacti",
		"cactiuser",
		"cron",
		"cthulhu",
		"db2inst",
		"deploy",
		"dff",
		"eggdrop",
		"fskjl32l32",
		"ftp",
		"ftpuser",
		"git",
		"gopher",
		"guest",
		"hadoop",
		"hastur",
		"itc",
		"john",
		"letmein",
		"log",
		"mail",
		"marine",
		"mcgrath",
		"munin",
		"mysql",
		"nagios",
		"navy",
		"news",
		"nobody",
		"oracle",
		"pi",
		"postfix",
		"postgres",
		"r00t",
		"samba",
		"sfdjlkfkjd",
		"squid",
		"staff",
		"support",
		"system",
		"teamspeak",
		"test",
		"testuser",
		"tomcat",
		"user",
		"viridian",
		"vyatta",
		"webmaster",
		"www",
		"zabbix",
		"zhangyan",
	} {
		badusers[u] = true
	}
}

func main() {
	var line LineInfo
	line = make(chan string, 5)

	go watch_sdjournal(&line)
	lookForLine(line)

}

func LineMatch(line string) (login SshLogin, found bool) {
	if m := rx.FindAllStringSubmatch(line, -1); m != nil {
		found = true
		login.IPaddr = m[0][2]
		login.User = m[0][1]
		return login, true
	} else {
		found = false
	}

	return login, found
}

func lookForLine(line <-chan string) {
	ticker := time.NewTicker(expirecycle)
	u := make(map[string]int)
	r := make(map[string]int)

	for {
		select {
		case <-ticker.C:
			fmt.Println("time for an expire run")
			for k, v := range u {
				fmt.Printf("* ip: %v %v\n", k, v)
				if v <= decay {
					delete(u, k)
					continue
				}
				u[k] -= decay
			}
			// expire all the roots
			r = make(map[string]int)
		case s := <-line:
			if l, ok := LineMatch(s); ok {
				fmt.Printf("ip: %v user: %v\n", l.IPaddr, l.User)

				if badusers[l.User] {
					fmt.Println("   is a baduser, instablock")
				}

				if l.User == "root" {
					r["root"]++
				}

				u[l.IPaddr]++
			}
		}
	}
}

func watch_sdjournal(out *LineInfo) {
	defer func() { out.Close() }()

	t, err :=  sdjournal.NewJournalReader(sdjournal.JournalReaderConfig{
		Since: time.Duration(-1) * time.Second,
		Matches: []sdjournal.Match{
			{
				Field: sdjournal.SD_JOURNAL_FIELD_SYSLOG_IDENTIFIER,
				Value: "sshd",
			},
		},
	})

	if err != nil {
		log.Fatalf("Error opening journal: %s", err)
	}

	defer t.Close()

	done := make(chan time.Time)

	if err = t.Follow(done, out); err != sdjournal.ErrExpired {
		log.Fatalf("Error during follow: %s", err)
	}
}
