package main

import (
	"fmt"
	"os"

	"github.com/hpcloud/tail"
)

func main() {
	config := tail.Config{Follow: true, ReOpen: true, Poll: true,
		Logger: tail.DiscardingLogger,
		Location: &tail.SeekInfo{0, os.SEEK_END}}

	line := make(chan string, 5)

	go tailFile("/tmp/bozo", config, line)
	lookForLine(line)

}

func lookForLine(line <-chan string) {
	for s := range line {
		fmt.Println(s)
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
