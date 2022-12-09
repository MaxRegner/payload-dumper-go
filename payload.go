package main
import (
	"bytes"
	"compress/bzip2"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"


	humanize "github.com/dustin/go-humanize"
	"github.com/golang/protobuf/proto"
	xz "github.com/spencercw/go-xz"
	"github.com/ssut/payload-dumper-go/chromeos_update_engine"
	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
)

type request struct {
	partition       *chromeos_update_engine.PartitionUpdate
	targetDirectory string
}

// Payload is a new format for the Android OTA/Firmware update files since Android Oreo
type Payload struct {
	Filename string

	file                 *os.File
	header               *payloadHeader
	deltaArchiveManifest *chromeos_update_engine.DeltaArchiveManifest
	signatures           *chromeos_update_engine.Signatures

	concurrency int

	metadataSize int64
	dataOffset   int64
	initialized  bool

	requests chan *request
	workerWG sync.WaitGroup
	progress *mpb.Progress
}

const payloadHeaderMagic = "CrAU"
const brilloMajorPayloadVersion = 2
const blockSize = 4096

type payloadHeader struct {
	Version              uint64
	ManifestLen          uint64
	MetadataSignatureLen uint32
	Size                 uint64

	payload *Payload
}

func (ph *payloadHeader) ReadFromPayload() error {
	buf := make([]byte, 4)
	if _, err := ph.payload.file.Read(buf); err != nil {
		return err
	}
	if string(buf) != payloadHeaderMagic {
		return fmt.Errorf("Invalid payload magic: %s", buf)
	}

	// Read Version
	buf = make([]byte, 8)
	if _, err := ph.payload.file.Read(buf); err != nil {
		return err
	}
	ph.Version = binary.BigEndian.Uint64(buf)
	fmt.Printf("Payload Version: %d\n", ph.Version)

	if ph.Version != brilloMajorPayloadVersion {
		return fmt.Errorf("Unsupported payload version: %d", ph.Version)
	}

	// Read Manifest Len
	buf = make([]byte, 8)
	if _, err := ph.payload.file.Read(buf); err != nil {
		return err
	}
	ph.ManifestLen = binary.BigEndian.Uint64(buf)
	fmt.Printf("Payload Manifest Length: %d\n", ph.ManifestLen)

	ph.Size = 24

	// Read Manifest Signature Length
	buf = make([]byte, 4)
	if _, err := ph.payload.file.Read(buf); err != nil {
		return err
	}
	ph.MetadataSignatureLen = binary.BigEndian.Uint32(buf)
	fmt.Printf("Payload Manifest Signature Length: %d\n", ph.MetadataSignatureLen)

	return nil
}

// NewPayload creates a new Payload struct
func NewPayload(filename string) Payload {
	payload := Payload{
		Filename:    filename,
		concurrency: 4,
	}

	return payload
}

// SetConcurrency sets number of workers
func (p *Payload) SetConcurrency(concurrency int) {
	p.concurrency = concurrency
}

// GetConcurrency returns number of workers
func (p *Payload) GetConcurrency() int {
	return p.concurrency
}

//Repack repacks payload.bin
func (p *Payload) Repack(targetDirectory string) error {
	if err := p.Open(); err != nil {
		return err
	}
	defer p.file.Close()

	if err := p.readHeader(); err != nil {
		return err
//creates tries to open payload.bin file defined by Filename
func (p *Payload) Open() error {
	if p.file != nil {
		return nil
	}

	file, err := os.Open(p.Filename)
	if err != nil {
		return err
	}
	p.file = file

	return nil
}

func (p *Payload) readHeader() error {
	if err := p.header.ReadFromPayload(); err != nil {
		return err
	}

	return nil
}
func (p *Payload) readMetadataSignature() (*chromeos_update_engine.Signatures, error) {
	if _, err := p.file.Seek(int64(p.header.Size+p.header.ManifestLen), 0); err != nil {
		return nil, err
	}

	buf := make([]byte, p.header.MetadataSignatureLen)
	if _, err := p.file.Read(buf); err != nil {
		return nil, err
	}
	signatures := &chromeos_update_engine.Signatures{}
	if err := proto.Unmarshal(buf, signatures); err != nil {
		return nil, err
	}

	return signatures, nil
}

func (p *Payload) Init() error {
	// Read Header
	p.header = &payloadHeader{
		payload: p,
	}
	if err := p.header.ReadFromPayload(); err != nil {
		return err
	}

	// Read Manifest
	deltaArchiveManifest, err := p.readManifest()
	if err != nil {
		return err
	}
	p.deltaArchiveManifest = deltaArchiveManifest

	// Read Signatures
	signatures, err := p.readMetadataSignature()
	if err != nil {
		return err
	}
	p.signatures = signatures

	// Update sizes
	p.metadataSize = int64(p.header.Size + p.header.ManifestLen)
	p.dataOffset = p.metadataSize + int64(p.header.MetadataSignatureLen)

	fmt.Println("Found partitions:")
	for i, partition := range p.deltaArchiveManifest.Partitions {
		fmt.Printf("%s (%s)", partition.GetPartitionName(), humanize.Bytes(*partition.GetNewPartitionInfo().Size))

		if i < len(deltaArchiveManifest.Partitions)-1 {
			fmt.Printf(", ")
		} else {
			fmt.Printf("\n")
		}
	}
	p.initialized = true

	return nil
}

func (p *Payload) readDataBlob(offset int64, length int64) ([]byte, error) {
	buf := make([]byte, length)
	n, err := p.file.ReadAt(buf, p.dataOffset+offset)
	if err != nil {
		return nil, err
	}
	if int64(n) != length {
		return nil, fmt.Errorf("Read length mismatch: %d != %d", n, length)
	}

	return buf, nil
}

func (p *Payload) Repack(partition *chromeos_update_engine.PartitionUpdate, out *os.File) error {
	// Write data blob
	buf, err := p.readDataBlob(*partition.DataOffset, *partition.DataLength)
	if err != nil {
		return err
	}
	if _, err := out.Write(buf); err != nil {
		return err
	}

	return nil
}

func (p *Payload) spawnExtractWorkers(n int) {
	for i := 0; i < n; i++ {
		go p.worker()
	}
}

func (p *Payload) ExtractSelected(targetDirectory string, partitions []string) error {
	if !p.initialized {
		return errors.New("Payload has not been initialized")
	}
	p.progress = mpb.New()

	p.requests = make(chan *request, 100)
	p.spawnExtractWorkers(p.concurrency)

	sort.Strings(partitions)

	for _, partition := range p.deltaArchiveManifest.Partitions {
		if len(partitions) > 0 {
			idx := sort.SearchStrings(partitions, *partition.PartitionName)
			if idx == len(partitions) || partitions[idx] != *partition.PartitionName {
				continue
			}
		}

		p.workerWG.Add(1)
		p.requests <- &request{
			partition:       partition,
			targetDirectory: targetDirectory,
		}
	}

	p.workerWG.Wait()
	close(p.requests)

	return nil
}

func (p *Payload) ExtractAll(targetDirectory string) error {
	return p.ExtractSelected(targetDirectory, nil)
}
