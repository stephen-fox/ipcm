package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/stephen-fox/lock"
)

func main() {
	resource := flag.String("resource", "", "The lock's resource")
	once := flag.Bool("once", false, "Only lock once")
	loopForever := flag.Bool("loop", false, "Loop forever after acquiring the lock")
	ipcTestPath := flag.String("ipcfile", "", "A file for testing IPC")
	ipcValue := flag.Int("ipcvalue", 0, "The number of times to increment the IPC value by")
	flag.Parse()

	m, err := lock.NewMutex(*resource)
	if err != nil {
		log.Fatal(err.Error())
	}

	if *once || *loopForever {
		err = m.TryLock()
		if err != nil {
			log.Fatal(err.Error())
		}
		defer m.Unlock()
	}

	if *loopForever {
		fmt.Println("ready")
		for {
			time.Sleep(1 * time.Second)
		}
	}

	if len(*ipcTestPath) > 0 {
		if *ipcValue < 1 {
			log.Fatal("ipc value must be greater than 0")
		}

		wg := &sync.WaitGroup{}
		wg.Add(*ipcValue)

		errs := make(chan error)
		for i := 0; i < *ipcValue; i++ {
			go func() {
				m.Lock()
				defer m.Unlock()
				defer wg.Done()

				raw, err := ioutil.ReadFile(*ipcTestPath)
				if err != nil {
					errs <- errors.New("failed to read IPC test file - " + err.Error())
					return
				}

				value, err := strconv.Atoi(string(raw))
				if err != nil {
					errs <- errors.New("failed to read integer from file - " + err.Error())
					return
				}

				value++
				err = ioutil.WriteFile(*ipcTestPath, []byte(strconv.Itoa(value)), 0600)
				if err != nil {
					errs <- errors.New("failed to write to IPC test file - " + err.Error())
					return
				}
			}()
		}

		wg.Wait()

		select {
		case err := <-errs:
			if err != nil {
				log.Fatalf("failed to execute IPC test - %s", err.Error())
			}
		default:
		}
	}
}
