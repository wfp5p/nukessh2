package main

import (
	"log"
	"time"
	"os"
	"os/signal"
	"syscall"

	"nukessh/blockhost"
	"github.com/coreos/go-systemd/sdjournal"
)

const (
	expirecycle = time.Hour
	blocktime = 24 * time.Hour
	decay       = 5
	threshold   = 10
	threshold_root = 3
	setname = "nukessh4"
	dbfile = "/var/cache/nukessh/nukedb"
)

func main() {
	bh, err := blockhost.New(dbfile, setname, blocktime)
	if err != nil {
		log.Fatalf("blockhost new failed: %s", err)
	}
	bh.BlockActives()

	var line LineInfo
	line = make(chan string, 5)

	go watch_sdjournal(&line)
	lookForLine(line, bh)

}

func lookForLine(line <-chan string, bh *blockhost.BlockHost) {
	ticker := time.NewTicker(expirecycle)
	u := make(map[string]int)
	r := make(map[string]int)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR2)

	for {
		select {
		case s := <-sig:
			if s == syscall.SIGUSR2 {
				log.Printf("ip map: %v\n", u)
			}
		case <-ticker.C:
			for k, v := range u {
				if v <= decay {
					delete(u, k)
					continue
				}
				u[k] -= decay
			}

			// expire all the roots
			r = make(map[string]int)

			// do the expire run for the db
			bh.ExpireDB()

		case s := <-line:
			if l, ok := LineMatch(s); ok {
//				fmt.Printf("ip: %v user: %v count: %v\n", l.IPaddr, l.User, u[l.IPaddr])

				if badusers[l.User] {
					log.Printf("%v is a baduser, instablock\n",l.User)
					u[l.IPaddr] += threshold
				}

				if l.User == "root" {
					r[l.IPaddr]++
//					fmt.Printf("   roots from %v: %v\n", l.IPaddr, r[l.IPaddr])
					if r[l.IPaddr] >= threshold_root {
						log.Printf("%v too many root attempts\n", l.IPaddr)
						u[l.IPaddr] += threshold
					}
				}

				u[l.IPaddr]++

				if u[l.IPaddr] >= threshold {
					log.Printf("%v blocked\n", l.IPaddr)
					bh.BlockHost(l.IPaddr)
					u[l.IPaddr] = 0
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
