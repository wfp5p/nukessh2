package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/hpcloud/tail"
)

type SshLogin struct {
	IPaddr string
	User string
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
	rx = regexp.MustCompile(`sshd\[\d+\]:\s+Failed password for (?:invalid\s+user\s+)?(.*) from (\d+\.\d+\.\d+\.\d+)\s+port`)

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
	if len(os.Args) == 1 {
        fmt.Printf("usage: %s\n", filepath.Base(os.Args[0]))
        os.Exit(1)
    }

	tailconfig := tail.Config{Follow: true, ReOpen: true, Poll: true,
//		Logger: tail.DiscardingLogger,
		Location: &tail.SeekInfo{0, os.SEEK_END}}

	line := make(chan string, 5)

	go tailFile(os.Args[1], tailconfig, line)
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

func tailFile(filename string, config tail.Config, out chan<- string) {
	defer func() { close(out) }()

	t, err := tail.TailFile(filename, config)
	if err != nil {
		fmt.Println(err)
		return
	}

	for line := range t.Lines {
		out <- line.Text
	}

	err = t.Wait()
	if err != nil {
		fmt.Println(err)
	}
}
