package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/stephen-fox/lock"
)

func main() {
	resource := flag.String("resource", "", "The lock's resource")
	loopForever := flag.Bool("loop", false, "If specified, loop forever after acquiring the lock")
	flag.Parse()

	l, err := lock.NewAcquirer().
		SetResource(*resource).
		Acquire()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer l.Release()

	if *loopForever {
		for {
			fmt.Println("ready")
			time.Sleep(1 * time.Second)
		}
	}
}
