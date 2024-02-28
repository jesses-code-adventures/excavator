package core

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

// watches the logfile
func Watch(filePath string, n int) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			file, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer file.Close()
			scanner := bufio.NewScanner(file)
			lines := make([]string, 0)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
				if len(lines) > n {
					lines = lines[1:]
				}
			}
			fmt.Print("\033[H\033[2J")
			for _, line := range lines {
				fmt.Println(line)
			}
			// Handle error from scanner.Err()
			if err := scanner.Err(); err != nil {
				return err
			}
		}
	}
}
