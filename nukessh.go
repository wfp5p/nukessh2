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

var (
	rx *regexp.Regexp
)

func init() {
	rx = regexp.MustCompile(`sshd\[\d+\]:\s+Failed password for (?:invalid\s+user\s+)?(.*) from (\d+\.\d+\.\d+\.\d+)\s+port`)
}

func main() {
	if len(os.Args) == 1 {
        fmt.Printf("usage: %s\n", filepath.Base(os.Args[0]))
        os.Exit(1)
    }

	config := tail.Config{Follow: true, ReOpen: true, Poll: true,
		Logger: tail.DiscardingLogger,
		Location: &tail.SeekInfo{0, os.SEEK_END}}

	line := make(chan string, 5)

	go tailFile(os.Args[1], config, line)
	lookForLine(line)

}

func LineMatch(line string) (login SshLogin, found bool) {
	if m :=  rx.FindAllStringSubmatch(line, -1); m != nil {
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
	ticker := time.NewTicker(60 * time.Minute)
	x := make(map[string]int)

	for {
		select {
		case <- ticker.C:
			fmt.Println("time for an expire run")
			for k, v := range x {
				fmt.Printf("* ip: %v %v\n", k, v)
			}
		case s := <- line:
			if l, ok := LineMatch(s); ok {
				fmt.Printf("ip: %v user: %v\n", l.IPaddr, l.User)
				x[l.IPaddr]++
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
