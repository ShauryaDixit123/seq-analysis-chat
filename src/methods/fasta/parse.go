package fasta

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode"
)

var (
	ErrEmpty       = errors.New("fasta: no records found")
	ErrBadSequence = errors.New("fasta: invalid sequence characters")
)

// Record is one FASTA entry (header + sequence).
type Record struct {
	Header   string
	Sequence string
}

// IsFASTA reports whether the filename looks like FASTA (.fa, .fasta, .fna, …).
func IsFASTA(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".fa", ".fasta", ".fna", ".ffn", ".faa", ".frn":
		return true
	default:
		return false
	}
}

// Parse reads one or more FASTA records from r.
func Parse(r io.Reader) ([]Record, error) {
	scanner := bufio.NewScanner(r)
	const maxLine = 10 * 1024 * 1024 // 10 MiB per line
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, maxLine)

	var records []Record
	var header string
	var seq strings.Builder

	flush := func() error {
		if header == "" {
			return nil
		}
		normalized, err := normalizeSequence(seq.String())
		if err != nil {
			return err
		}
		if normalized == "" {
			return fmt.Errorf("fasta: empty sequence for %q", header)
		}
		records = append(records, Record{Header: header, Sequence: normalized})
		header = ""
		seq.Reset()
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line[0] == '>' {
			if err := flush(); err != nil {
				return nil, err
			}
			header = strings.TrimSpace(strings.TrimPrefix(line, ">"))
			continue
		}
		if header == "" {
			return nil, errors.New("fasta: sequence line before header")
		}
		seq.WriteString(line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if err := flush(); err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, ErrEmpty
	}
	return records, nil
}

// First returns the first record or an error if the file is empty.
func First(r io.Reader) (Record, error) {
	records, err := Parse(r)
	if err != nil {
		return Record{}, err
	}
	return records[0], nil
}

func normalizeSequence(raw string) (string, error) {
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		if unicode.IsSpace(r) {
			continue
		}
		r = unicode.ToUpper(r)
		if !isValidBase(r) {
			return "", fmt.Errorf("%w: %q", ErrBadSequence, r)
		}
		b.WriteRune(r)
	}
	return b.String(), nil
}

func isValidBase(r rune) bool {
	switch r {
	case 'A', 'C', 'G', 'T', 'U', 'N', '-', '.', '*':
		return true
	default:
		return false
	}
}
