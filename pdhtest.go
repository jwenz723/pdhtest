package main

import (
	"fmt"
	"os"
	"log"
	"bufio"
	"github.com/lxn/win"
)

func main() {
	const COUNTERS_FILE= "counters.txt"

	done := make(chan struct{})
	countersChannel := make(chan string)
	go readCounterConfigFile(COUNTERS_FILE, countersChannel)
	go processCounters(countersChannel, done)
	
	<- done
	fmt.Println("processed all counters")
}

func readCounterConfigFile(file string, countersChannel chan string) {
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		countersChannel <- scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("error reading file (%s): %s\n", file, err)
	}

	close(countersChannel)
}

func processCounters(countersChannel chan string, done chan struct{}) {
	for counter := range countersChannel {
		var queryHandle win.PDH_HQUERY
		var counterHandle win.PDH_HCOUNTER

		ret := win.PdhOpenQuery(0, 0, &queryHandle)
		if ret != win.ERROR_SUCCESS {
			fmt.Println("PdhOpenQuery failed", counter)
			continue
		}

		ret = win.PdhValidatePath(counter)
		if ret == win.PDH_CSTATUS_BAD_COUNTERNAME {
			fmt.Println("PdhValidatePath failed", counter)
			continue
		}

		ret = win.PdhAddEnglishCounter(queryHandle, counter, 0, &counterHandle)
		if ret != win.ERROR_SUCCESS {
			fmt.Println("PdhAddEnglishCounter failed", counter)
			continue
		}

		ret = win.PdhCollectQueryData(queryHandle)
		if ret != win.ERROR_SUCCESS {
			fmt.Println("PdhCollectQueryData failed", counter)
			continue
		}
	}

	done <- struct{}{}
}