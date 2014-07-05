package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/MichaelTJones/walk"
)

var duration = flag.String("d", "", "find files modified within DURATION")
var format = flag.String("f", "2006-01-02 03:04:05", "time format")
var instant = flag.String("t", "", "find files modified since TIME")
var quiet = flag.Bool("q", false, "do not print filenames")
var verbose = flag.Bool("v", false, "print summary statistics")

func main() {
	flag.Parse()

	now := time.Now()
	when := now
	switch {
	case *instant != "":
		t, err := time.Parse(*format, *instant)
		if err != nil {
			fmt.Printf("error parsing time %q, %s\n", *instant, err)
			os.Exit(1)
		}
		when = t
	case *duration != "":
		d, err := time.ParseDuration(*duration)
		if err != nil {
			fmt.Printf("error parsing duration %q, %s\n", *duration, err)
			os.Exit(2)
		}
		when = now.Add(-d)
	}

	// goroutine to collect names of recently-modified files
	var result []string
	done := make(chan bool)
	results := make(chan string, 1024)
	go func() {
		for r := range results {
			result = append(result, r)
		}
		sort.Strings(result) // simulate ordered traversal
		done <- true
	}()

	// parallel walker and walk to find recently-modified files
	var lock sync.Mutex
	var tFiles, tBytes int // total files and bytes
	var rFiles, rBytes int // recent files and bytes
	sizeVisitor := func(path string, info os.FileInfo, err error) error {
		if err == nil {
			lock.Lock()
			tFiles += 1
			tBytes += int(info.Size())
			lock.Unlock()

			if info.ModTime().After(when) {
				lock.Lock()
				rFiles += 1
				rBytes += int(info.Size())
				lock.Unlock()

				if !*quiet {
					// fmt.Printf("%s %s\n", info.ModTime(), path) // simple
					results <- path // allows sorting into "normal" order
				}
			}
		}
		return nil
	}
	for _, root := range flag.Args() {
		walk.Walk(root, sizeVisitor)
	}

	// wait for traversal results and print
	close(results) // no more results
	<-done         // wait for final results and sorting
	for _, r := range result {
		fmt.Printf("%s\n", r)
	}

	𝛥t := float64(time.Since(now)) / 1e9

	// print optional verbose summary report
	if *verbose {
		log.Printf("     total: %8d files (%7.2f%%), %13d bytes (%7.2f%%)\n",
			tFiles, 100.0, tBytes, 100.0)

		rfp := 100 * float64(rFiles) / float64(tFiles)
		rbp := 100 * float64(rBytes) / float64(tBytes)
		log.Printf("    recent: %8d files (%7.2f%%), %13d bytes (%7.2f%%) in %.4f seconds\n",
			rFiles, rfp, rBytes, rbp, 𝛥t)
	}
}
