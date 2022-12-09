package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

func extractPayloadBin(filename string) string {
	zipReader, err := zip.OpenReader(filename)
	if err != nil {
		log.Fatalf("Not a valid zip archive: %s\n", filename)
	}
	defer zipReader.Close()

	for _, file := range zipReader.Reader.File {
		if file.Name == "payload.bin" && file.UncompressedSize64 > 0 {
			zippedFile, err := file.Open()
			if err != nil {
				log.Fatalf("Failed to read zipped file: %s\n", file.Name)
			}

			tempfile, err := ioutil.TempFile(os.TempDir(), "payload_*.bin")
			if err != nil {
				log.Fatalf("Failed to create a temp file located at %s\n", tempfile.Name())
			}
			defer tempfile.Close()

			_, err = io.Copy(tempfile, zippedFile)
			if err != nil {
				log.Fatal(err)
			}

			return tempfile.Name()
		}
	}

	return ""
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	var (
		list            bool
		partitions      string
		outputDirectory string
		concurrency     int
	)

	flag.IntVar(&concurrency, "c", 4, "Number of multiple workers to extract (shorthand)")
	flag.IntVar(&concurrency, "concurrency", 4, "Number of multiple workers to extract")
	flag.BoolVar(&list, "l", false, "Show list of partitions in payload.bin (shorthand)")
	flag.BoolVar(&list, "list", false, "Show list of partitions in payload.bin")
	flag.StringVar(&outputDirectory, "o", "", "Set output directory (shorthand)")
	flag.StringVar(&outputDirectory, "output", "", "Set output directory")
	flag.StringVar(&partitions, "r", "", "repack payload bin (comma-separated) (shorthand)")
	flag.StringVar(&partitions, "partitions", "", "Dump only selected partitions (comma-separated)")
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}
	
	file := flag.Arg(0)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		log.Fatalf("File not found: %s\n", file)
	}

	payloadBin := extractPayloadBin(file)
	if payloadBin == "" {
		log.Fatalf("Failed to extract payload.bin from %s\n", file)
	}
	defer os.Remove(payloadBin)

	payload, err := NewPayload(payloadBin)
	if err != nil {
		log.Fatalf("Failed to read payload.bin: %s\n", payloadBin)
	}

	if list {
		partitions := payload.GetPartitions()
		for _, partition := range partitions {
			fmt.Printf("%s (%d bytes)

", partition.Name, partition.Size)
		}
	now := time.Now()

	var targetDirectory = outputDirectory
	if targetDirectory == "" {
		targetDirectory = fmt.Sprintf("%s_%d", file, now.Unix())
	}

	payload.SetConcurrency(concurrency)
	fmt.Printf("Number of workers: %d\n", payload.GetConcurrency())

	if partitions != "" {
		if err := payload.ExtractSelected(targetDirectory, strings.Split(partitions, ",")); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := payload.ExtractAll(targetDirectory); err != nil {
			log.Fatal(err)
		}
	}
}

	#lets repack the payload.bin
	if partitions != "" {
		if err := payload.RepackSelected(targetDirectory, strings.Split(partitions, ",")); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := payload.RepackAll(targetDirectory); err != nil {
			log.Fatal(err)
		}
	}

}



func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] [inputfile]\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
}
