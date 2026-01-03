package jsonl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// MaxScannerBuffer is the maximum buffer size for the scanner (10MB).
const MaxScannerBuffer = 10 * 1024 * 1024

// Parser provides streaming parsing of JSONL files.
type Parser struct {
	scanner *bufio.Scanner
	file    *os.File
}

// NewParser creates a new parser for the given file path.
func NewParser(path string) (*Parser, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), MaxScannerBuffer)

	return &Parser{
		scanner: scanner,
		file:    file,
	}, nil
}

// NewParserFromReader creates a new parser from an io.Reader.
func NewParserFromReader(r io.Reader) *Parser {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), MaxScannerBuffer)

	return &Parser{
		scanner: scanner,
	}
}

// Close closes the underlying file if one was opened.
func (p *Parser) Close() error {
	if p.file != nil {
		return p.file.Close()
	}
	return nil
}

// Next returns the next raw entry, or nil if there are no more entries.
func (p *Parser) Next() (*RawEntry, error) {
	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return nil, fmt.Errorf("scanning: %w", err)
		}
		return nil, nil // EOF
	}

	line := p.scanner.Bytes()
	if len(line) == 0 {
		return p.Next() // Skip empty lines
	}

	var entry RawEntry
	if err := json.Unmarshal(line, &entry); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return &entry, nil
}

// NextRaw returns the next line as raw bytes without parsing.
func (p *Parser) NextRaw() ([]byte, error) {
	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return nil, fmt.Errorf("scanning: %w", err)
		}
		return nil, nil // EOF
	}
	return p.scanner.Bytes(), nil
}

// ParseAll parses all entries from the file.
func (p *Parser) ParseAll() ([]*RawEntry, error) {
	var entries []*RawEntry
	for {
		entry, err := p.Next()
		if err != nil {
			return entries, err
		}
		if entry == nil {
			break
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// ParseEntry parses a single JSON line into a RawEntry.
func ParseEntry(line []byte) (*RawEntry, error) {
	var entry RawEntry
	if err := json.Unmarshal(line, &entry); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	return &entry, nil
}

// ParseMessage parses the Message field of a RawEntry into a full Message.
func ParseMessage(entry *RawEntry) (*Message, error) {
	if entry.Message == nil {
		return nil, nil
	}

	var msg Message
	if err := json.Unmarshal(entry.Message, &msg); err != nil {
		return nil, fmt.Errorf("parsing message: %w", err)
	}
	return &msg, nil
}
