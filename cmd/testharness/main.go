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

	m, err := lock.NewMutex(*resource)
	if err != nil {
		log.Fatal(err.Error())
	}

	err = m.TryLock()
	if err != nil {
		log.Fatal(err.Error())
	}
	defer m.Unlock()

	if *loopForever {
		for {
			fmt.Println("ready")
			time.Sleep(1 * time.Second)
		}
	}
}
