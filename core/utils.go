package core

import (
	"log"
	"os"
	"path/filepath"
)

func ExpandPath(dir string) string {
	if len(dir) < 2 {
		return dir
	}
	if dir[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Unable to find user home directory: %v", err)
		}
		dir = filepath.Join(home, dir[2:])
	}
	return dir
}
