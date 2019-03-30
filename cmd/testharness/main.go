package main

import (
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
	resource := flag.String("resource", "", "The mutex's resource")
	loopForever := flag.Bool("loop", false, "Loop forever after acquiring the mutex")
	ipcTestPath := flag.String("ipcfile", "", "A file for testing IPC")
	ipcValue := flag.Int("ipcvalue", 0, "The number of times to increment the IPC value by")

	flag.Parse()

	m, err := lock.NewMutex(*resource)
	if err != nil {
		log.Fatalln(err.Error())
	}

	if len(*ipcTestPath) > 0 {
		err := doInterProcessCommunicationTest(m, *ipcTestPath, *ipcValue)
		if err != nil {
			log.Fatalln(err.Error())
		}

		return
	}

	err = m.TimedTryLock(1 * time.Second)
	if err != nil {
		log.Fatalln(err.Error())
	}
	defer m.Unlock()

	if *loopForever {
		fmt.Println("ready")
		for {
			time.Sleep(1 * time.Second)
		}
	}
}

func doInterProcessCommunicationTest(m lock.Mutex, ipcValueFilePath string, maxValue int) error {
	if maxValue < 1 {
		return fmt.Errorf("ipc value must be greater than 0")
	}

	wg := &sync.WaitGroup{}
	wg.Add(maxValue)

	errs := make(chan error)
	for i := 0; i < maxValue; i++ {
		go func() {
			m.Lock()
			defer m.Unlock()
			defer wg.Done()

			raw, err := ioutil.ReadFile(ipcValueFilePath)
			if err != nil {
				errs <- fmt.Errorf("failed to read ipc test file - " + err.Error())
				return
			}

			value, err := strconv.Atoi(string(raw))
			if err != nil {
				errs <- fmt.Errorf("failed to parse ipc file's value - " + err.Error())
				return
			}

			value++
			err = ioutil.WriteFile(ipcValueFilePath, []byte(strconv.Itoa(value)), 0600)
			if err != nil {
				errs <- fmt.Errorf("failed to write to ipc test file - " + err.Error())
				return
			}
		}()
	}

	rejoin := make(chan struct{})
	go func() {
		wg.Wait()
		close(rejoin)
	}()

	select {
	case err := <-errs:
		if err != nil {
			return fmt.Errorf("failed to execute IPC test - %s", err.Error())
		}
	case <-rejoin:
	}

	return nil
}
