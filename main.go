package main

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var matcher = regexp.MustCompile(`\d{2}:(\d{2}:\d{2}):\d{2}`)

func main() {
	if len(os.Args) == 1 {
		files, err := os.ReadDir(".")
		if err != nil {
			fmt.Println(err)
			return
		}

		var cnt int
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			ok, err := processFile(file.Name())
			if err != nil {
				fmt.Println(err)
			}

			if ok {
				cnt++
			}
		}

		fmt.Printf("Processed %d files\n", cnt)
		return
	}

	for _, filename := range os.Args[1:] {
		ok, err := processFile(filename)
		if err != nil {
			fmt.Println(err)
		}
		if !ok {
			fmt.Println("unknown file extension for:", filename)
		}
	}
}

func processFile(filename string) (bool, error) {
	filename, err := filepath.Abs(filename)
	if err != nil {
		return false, err
	}

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".csv":
		err := processCSV(filename)
		if err != nil {
			return false, err
		}

	case ".edl":
		err := processEDL(filename)
		if err != nil {
			return false, err
		}

	default:
		return false, nil
	}

	return true, nil
}

func processCSV(filename string) error {
	// Open file as CSV
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true

	headers, err := r.Read()
	if err != nil {
		return fmt.Errorf("error reading header: %w", err)
	}

	// Read all lines as map
	var lines []map[string]string
	for {
		line, err := r.Read()
		if errors.Is(io.EOF, err) {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading line: %w", err)
		}

		lines = append(lines, lineToMap(headers, line))
	}

	// Print out just the Record In and Notes field as Youtube table of contents format
	fmt.Println(filepath.Base(filename))
	for _, line := range lines {
		matches := matcher.FindStringSubmatch(line["Record In"])
		if len(matches) != 2 {
			return fmt.Errorf("error parsing time: %s", line["Record In"])
		}
		fmt.Printf("%s %s\n", matches[1], line["Notes"])
	}

	fmt.Println()

	return nil
}

func lineToMap(headers, line []string) map[string]string {
	m := make(map[string]string)
	for i, header := range headers {
		m[header] = line[i]
	}
	return m
}

func processEDL(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()
	fo, err := os.Create(
		strings.TrimSuffix(filename, filepath.Ext(filename)) + ".txt",
	)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer fo.Close()

	o := io.MultiWriter(os.Stdout, fo)

	matchTime := regexp.MustCompile(`^(\d+)\s+\d+\s+V\s+C\s+(\d{2})\:(\d{2}\:\d{2})\:\d{2}\s.*`)
	matchText := regexp.MustCompile(`^\s+\|C:(\w+)\s+\|M:([^|]+)\s\|D:\d+`)

	s := bufio.NewScanner(f)
	fmt.Println(filepath.Base(filename))
	for s.Scan() {
		matches := matchTime.FindStringSubmatch(s.Text())
		if len(matches) == 4 {
			i, err := strconv.Atoi(matches[2])
			if err != nil {
				return fmt.Errorf("error parsing line number: %w", err)
			}
			fmt.Fprintf(o, "%02d:%s ", i-1, matches[3])
		}

		matches = matchText.FindStringSubmatch(s.Text())
		if len(matches) == 3 {
			fmt.Fprintln(o, matches[2])
		}
	}

	if err := s.Err(); err != nil {
		return fmt.Errorf("error scanning file: %w", err)
	}

	fmt.Println()

	return nil
}
