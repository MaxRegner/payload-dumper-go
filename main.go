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

func parsePayloadBin(filename string) string {

	// Open a zip archive for reading.
	r, err := zip.OpenReader(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	// Iterate through the files in the archive,
	// printing some of their contents.
	for _, f := range r.File {
		fmt.Printf("Contents of %s:\n", f.Name)
		rc, err := f.Open()
		if err != nil {
			log.Fatal(err)
		}
		_, err = io.CopyN(ioutil.Discard, rc, 68)
		if err != nil {
			log.Fatal(err)
		}
		buf := make([]byte, 8)
		_, err = rc.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Payload Version: %d\n", binary.BigEndian.Uint64(buf))
		_, err = io.CopyN(ioutil.Discard, rc, 8)
		if err != nil {
			log.Fatal(err)
		}
		buf = make([]byte, 4)
		_, err = rc.Read(buf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Payload Manifest Signature Length: %d\n", binary.BigEndian.Uint32(buf))
		rc.Close()
	}
	return ""
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

    
	flag.BoolVar(&list, "l", false, "List partitions")
	flag.StringVar(&partitions, "p", "", "Partitions to replace")
	flag.StringVar(&outputDirectory, "o", "", "Output directory")
	flag.IntVar(&concurrency, "c", 4, "Number of workers")
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}
	filename := flag.Arg(0)

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		log.Fatalf("File does not exist: %s\n", filename)
	}

	if list {
		listPartitions(filename)
		os.Exit(0)
	}

	if partitions != "" {
		if _, err := os.Stat(outputDirectory); os.IsNotExist(err) {
			log.Fatalf("Output directory does not exist: %s\n", outputDirectory)
		}
	}

	start := time.Now()

	if err := extract(filename, partitions, outputDirectory, concurrency); err != nil {
		log.Fatal(err)
	}

	log.Printf("Extracted in %s\n", time.Since(start))
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



//add payload parse function to replace system image in payload file
func (p *Payload) AddPayloadParse() error {
	//add payload parse function to replace system image in payload file
	if err := p.parsePayload(); err != nil {
		return err
	}
	return nil
}
