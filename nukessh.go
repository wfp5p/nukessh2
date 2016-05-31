package main

import (
	"fmt"
	"log"
	"time"

	"github.com/coreos/go-systemd/sdjournal"
)

const (
	expirecycle = time.Hour
	decay       = 10
	threshold   = 15
	threshold_root = 3
)

func main() {
	var line LineInfo
	line = make(chan string, 5)

	go watch_sdjournal(&line)
	lookForLine(line)

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
					fmt.Println("--- is a baduser, instablock")
				}

				if l.User == "root" {
					r[l.IPaddr]++
					if r[l.IPaddr] > threshold_root {
						fmt.Printf("--- too many roots from %v\n", l.IPaddr)
						// block
					}
				}

				u[l.IPaddr]++
				if u[l.IPaddr] > threshold {
					fmt.Printf("--- too many attempts from %v\n", l.IPaddr)
					// block
				}
			}
		}
	}
}

func watch_sdjournal(out *LineInfo) {
	defer func() { out.Close() }()

	t, err := sdjournal.NewJournalReader(sdjournal.JournalReaderConfig{
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
