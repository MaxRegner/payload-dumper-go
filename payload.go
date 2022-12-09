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
	zipReader, err := zip.OpenReader(filename
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
	flag.StringVar(&partitions, "p", "", "Set partitions to extract (shorthand)")
	flag.StringVar(&partitions, "partitions", "", "Set partitions to extract")
	flag.Parse()

	if len(os.Args) == 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if flag.NArg() == 0 {
		log.Fatal("Please specify a payload.bin file")
	}

	if flag.NArg() > 1 {
		log.Fatal("Please specify only one payload.bin file")
	}

	if outputDirectory == "" {
		log.Fatal("Please specify an output directory")
	}

	if _, err := os.Stat(outputDirectory); os.IsNotExist(err) {
		os.Mkdir(outputDirectory, 0755)
	}

	if partitions == "" {
		log.Fatal("Please specify partitions to extract")
	}

	if concurrency < 1 {
		log.Fatal("Please specify a concurrency greater than 0")
	}

	payloadFile := flag.Arg(0)
	if _, err := os.Stat(payload
		log.Fatal(err)
	}

	payloadBin := extractPayloadBin(payloadFile)
	if payloadBin == "" {
		log.Fatalf("Failed to extract payload.bin from %s\n", payloadFile)
	}

	defer os.Remove(payloadBin)

	payloadReader, err := NewReader(payloadBin, 0)
	if err != nil {
		log.Fatal(err)
	}

	defer payloadReader.Close()

	if list {
		for _, partition := range payloadReader.Partitions {
			fmt.Println(partition.Name)
		}
		os.Exit(0)
	}

	partitionsToExtract := strings.Split(partitions, ",")
	for _, partitionName := range partitionsToExtract {
		partitionFound := false
		for _, partition := range payloadReader.Partitions {
			if partition.Name == partitionName {
				partitionFound = true
				break
			}
		}

		if !partitionFound {
			log.Fatalf("Partition %s not found in payload.bin\n", partitionName)
		}
	}

	
