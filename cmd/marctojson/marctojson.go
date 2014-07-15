package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/miku/marc21"
	"github.com/miku/marctools"
	"io"
	"log"
	"os"
	"runtime"
)

type Work struct {
	record        *marc21.Record
	filterMap     map[string]bool
	includeLeader bool
	recordMapChan chan map[string]interface{}
}

func RecordToMap(work Work) {
	recordMap := marctools.RecordToMap(work.record, work.filterMap, work.includeLeader)
	work.recordMapChan <- recordMap
}

// preformance data points:
// 9798925 records, sequential
// $ time go run cmd/marctojson/marctojson.go fixtures/biglok.mrc > /dev/null
// real	7m18.731s
// user	6m16.256s
// sys	1m13.612s

// 9798925 records, single short-lived goroutine
// $ time go run cmd/marctojson/marctojson.go fixtures/biglok.mrc > /dev/null
// real	12m49.862s
// user	12m39.992s
// sys	3m23.380s
func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	ignore := flag.Bool("i", false, "ignore marc errors (not recommended)")
	version := flag.Bool("v", false, "prints current program version and exit")

	metaVar := flag.String("m", "", "a key=value pair to pass to meta")
	filterVar := flag.String("r", "", "only dump the given tags (e.g. 001,003)")
	leaderVar := flag.Bool("l", false, "dump the leader as well")
	plainVar := flag.Bool("p", false, "plain mode: dump without content and meta")

	var PrintUsage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] MARCFILE\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if *version {
		fmt.Println(marctools.AppVersion)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		PrintUsage()
		os.Exit(1)
	}

	file, err := os.Open(flag.Args()[0])
	if err != nil {
		log.Fatalln(err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	// all recordMaps are placed on this channel
	recordMapChan := make(chan map[string]interface{})

	filterMap := marctools.StringToMapSet(*filterVar)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	for {
		record, err := marc21.ReadRecord(file)
		if err == io.EOF {
			break
		}
		if err != nil {
			if *ignore {
				fmt.Fprintf(os.Stderr, "Skipping, since -i was set. Error: %s\n", err)
				continue
			} else {
				log.Fatalln(err)
			}
		}

		work := Work{record: record, filterMap: filterMap, includeLeader: *leaderVar, recordMapChan: recordMapChan}
		// recordMap := marctools.RecordToMap(record, filterMap, *leaderVar)
		go RecordToMap(work)
		recordMap := <-recordMapChan

		if *plainVar {
			b, err := json.Marshal(recordMap)
			if err != nil {
				log.Fatalf("error: %s", err)
			}
			writer.Write(b)
			writer.Write([]byte("\n"))
		} else {
			mainMap := make(map[string]interface{})
			mainMap["content"] = recordMap
			metamap, err := marctools.KeyValueStringToMap(*metaVar)
			if err != nil {
				log.Fatalln(err)
			}
			mainMap["meta"] = metamap
			b, err := json.Marshal(mainMap)
			if err != nil {
				log.Fatalf("error: %s", err)
			}
			writer.Write(b)
			writer.Write([]byte("\n"))
		}
	}
}
