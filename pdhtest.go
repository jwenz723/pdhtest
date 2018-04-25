package main

import (
	"fmt"
	"os"
	"log"
	"bufio"
	"github.com/lxn/win"
	"unsafe"
)

func main() {
	const COUNTERS_FILE= "counters.txt"

	statusChan := make(chan uint32)
	countersChannel := make(chan string)
	go readCounterConfigFile(COUNTERS_FILE, countersChannel)
	go processCounters(countersChannel, statusChan)

	statusCounts := map[uint32]int{}
	for status := range statusChan {
		if _, ok := statusCounts[status]; !ok {
			statusCounts[status] = 0
		}
		statusCounts[status]++
	}

	fmt.Println("Counts of error codes:")
	for k, v := range statusCounts {
		fmt.Printf("\t- %x: %d\n", k, v)
	}
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

func processCounters(countersChannel chan string, status chan uint32) {
	var queryHandle win.PDH_HQUERY
	counterHandles := map[string]*win.PDH_HCOUNTER{}

	ret := win.PdhOpenQuery(0, 0, &queryHandle)
	if ret != win.ERROR_SUCCESS {
		status <- ret
	} else {
		for counter := range countersChannel {
			var c win.PDH_HCOUNTER
			ret = win.PdhValidatePath(counter)
			if ret == win.PDH_CSTATUS_BAD_COUNTERNAME {
				status <- ret
				continue
			}

			ret = win.PdhAddEnglishCounter(queryHandle, counter, 0, &c)
			if ret != win.ERROR_SUCCESS {
				status <- ret
				continue
			}

			status <- ret
			counterHandles[counter] = &c
		}

		ret = win.PdhCollectQueryData(queryHandle)
		if ret != win.ERROR_SUCCESS {
			status <- ret
		} else {
			counterCount := 0
			instanceCount := 0
			ret := win.PdhCollectQueryData(queryHandle)
			if ret == win.ERROR_SUCCESS {
				for _, v := range counterHandles {
					var bufSize uint32
					var bufCount uint32
					var size= uint32(unsafe.Sizeof(win.PDH_FMT_COUNTERVALUE_ITEM_DOUBLE{}))
					var emptyBuf [1]win.PDH_FMT_COUNTERVALUE_ITEM_DOUBLE // need at least 1 addressable null ptr.

					ret = win.PdhGetFormattedCounterArrayDouble(*v, &bufSize, &bufCount, &emptyBuf[0])
					if ret == win.PDH_MORE_DATA {
						counterCount++
						filledBuf := make([]win.PDH_FMT_COUNTERVALUE_ITEM_DOUBLE, bufCount*size)
						ret = win.PdhGetFormattedCounterArrayDouble(*v, &bufSize, &bufCount, &filledBuf[0])
						if ret == win.ERROR_SUCCESS {
							for i := 0; i < int(bufCount); i++ {
								instanceCount++
							}
						}
					}
				}
			} else {
				status <- ret
			}

			fmt.Printf("Counters with values: %d (%d instances)\n", counterCount, instanceCount)
		}
	}

	close(status)
}