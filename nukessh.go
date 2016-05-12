package main

import (
	"fmt"
	"os"

	"github.com/hpcloud/tail"
)

func main() {
	config := tail.Config{Follow: true, ReOpen: true, Poll: true,
		Logger: tail.DiscardingLogger}
	// if flag.NFlag() < 1 {
	// 	fmt.Println("need one or more files as arguments")
	// 	os.Exit(1)
	// }

	done := make(chan bool)
	files := os.Args[1:]
	for _, filename := range files {
		// fmt.Println(filename)
		go tailFile(filename, config, done)
	}

	for _, _ = range files {
	 	<-done
	}
}

func tailFile(filename string, config tail.Config, done chan bool) {
	defer func() { done <- true }()
	t, err := tail.TailFile(filename, config)
	if err != nil {
		fmt.Println("this was err from tail")
		fmt.Println(err)
		return
	}
	for line := range t.Lines {
		fmt.Println(line.Text)
	}
	err = t.Wait()
	if err != nil {
		fmt.Println("this was err from wait")
		fmt.Println(err)
	}
}
