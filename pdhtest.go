package main

import (
	"fmt"
	"os"
	"log"
	"bufio"
	"github.com/lxn/win"
	"unsafe"
	"time"
)

func main() {
	const COUNTERS_FILE= "counters.txt"

	statusChan := make(chan bool)
	//done := make(chan struct{})
	countersChannel := make(chan string)
	go readCounterConfigFile(COUNTERS_FILE, countersChannel)
	go processCounters(countersChannel, statusChan)

	successCount := 0
	failCount := 0
	for success := range statusChan {
		if success {
			successCount++
		} else {
			failCount++
		}
	}
	
	//<- done
	fmt.Printf("processed all counters. Success: %d. Failures: %d\n", successCount, failCount)
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

func processCounters(countersChannel chan string, status chan bool) {
	var queryHandle win.PDH_HQUERY
	ret := win.PdhOpenQuery(0, 0, &queryHandle)
	if ret != win.ERROR_SUCCESS {
		fmt.Println("PdhOpenQuery failed")
	}

	counterHandles := map[string]*win.PDH_HCOUNTER{}

	for counter := range countersChannel {
		var c win.PDH_HCOUNTER
		ret = win.PdhValidatePath(counter)
		if ret == win.PDH_CSTATUS_BAD_COUNTERNAME {
			fmt.Println("PdhValidatePath failed", counter)
			status <- false
			continue
		}

		ret = win.PdhAddEnglishCounter(queryHandle, counter, 0, &c)
		if ret != win.ERROR_SUCCESS &&
			ret != win.PDH_CSTATUS_NO_OBJECT {
			fmt.Println("PdhAddEnglishCounter failed", counter)
			status <- false
			continue
		}

		status <- true
		counterHandles[counter] = &c
	}

	ret = win.PdhCollectQueryData(queryHandle)
	if ret != win.ERROR_SUCCESS {
		fmt.Println("PdhCollectQueryData failed")
	} else {
		// start data collection
		for {
			valCount := 0
			ret := win.PdhCollectQueryData(queryHandle)
			if ret == win.ERROR_SUCCESS {
				for _, v := range counterHandles {
					var bufSize uint32
					var bufCount uint32
					var size = uint32(unsafe.Sizeof(win.PDH_FMT_COUNTERVALUE_ITEM_DOUBLE{}))
					var emptyBuf [1]win.PDH_FMT_COUNTERVALUE_ITEM_DOUBLE // need at least 1 addressable null ptr.

					ret = win.PdhGetFormattedCounterArrayDouble(*v, &bufSize, &bufCount, &emptyBuf[0])
					if ret == win.PDH_MORE_DATA {
						filledBuf := make([]win.PDH_FMT_COUNTERVALUE_ITEM_DOUBLE, bufCount*size)
						ret = win.PdhGetFormattedCounterArrayDouble(*v, &bufSize, &bufCount, &filledBuf[0])
						if ret == win.ERROR_SUCCESS {
							for i := 0; i < int(bufCount); i++ {
								//c := filledBuf[i]
								//s := win.UTF16PtrToString(c.SzName)
								//
								//fmt.Printf("%s[%s] = %f\n", k, s, c.FmtValue.DoubleValue)
								valCount++
							}
						}
					}
				}
			} else {
				fmt.Printf("failed to obtain instances\n")
			}

			fmt.Printf("Collected %d values\n", valCount)
			time.Sleep(1 * time.Second)
		}
	}

	close(status)
}