package collector

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Collect reads all .log files in the given directory and sends each line
// to the returned channel. It reads files concurrently using goroutines.
func Collect(logDir string) (<-chan string, error) {
	matches, err := filepath.Glob(filepath.Join(logDir, "*.log"))
	if err != nil {
		return nil, fmt.Errorf("glob log files: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no .log files found in %s", logDir)
	}

	lines := make(chan string, 1000)

	var wg sync.WaitGroup

	for _, path := range matches {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			if err := readFile(p, lines); err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Collector: error reading %s: %v\n", p, err)
			}
		}(path)
	}

	go func() {
		wg.Wait()
		close(lines)
	}()

	fmt.Printf("📥 Collector: reading %d log file(s)\n", len(matches))
	return lines, nil
}

func readFile(path string, out chan<- string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Increase buffer for potentially long lines.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			out <- line
		}
	}
	return scanner.Err()
}
