package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/stephen-fox/lock"
)

func main() {
	location := flag.String("location", "", "The lock's well-known location")
	loopForever := flag.Bool("loop", false, "If specified, loop forever after acquiring the lock")
	flag.Parse()

	l, err := lock.NewAcquirer().
		SetLocation(*location).
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
