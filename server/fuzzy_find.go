package server

import (
	"io/fs"
	"log"
	"path"
	"path/filepath"
	"strings"

	"github.com/jesses-code-adventures/excavator/core"
)

func ContainsAllSubstrings(s1 string, s2 string) bool {
	words := strings.Fields(s2)
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)
	for _, word := range words {
		if !strings.Contains(s1, word) {
			return false
		}
	}
	return true
}

// Standard function for getting the necessary files from a dir with their associated tags
func (s *Server) FuzzyFind(search string, fromRoot bool) {
	var dir string
	if len(search) == 0 {
		return
	}
	log.Println("in server fuzzy search fn")
	if fromRoot {
		dir = s.State.Root
	} else {
		dir = s.State.Dir
	}
	collectionTags := s.FuzzyFindCollectionTags(search)
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !ContainsAllSubstrings(path.Base(p), search) || strings.HasPrefix(p, ".") || strings.HasSuffix(p, ".asd") || strings.HasSuffix(p, ".nki") {
			return nil
		}
		if (strings.HasSuffix(p, ".wav") || strings.HasSuffix(p, ".mp3") || strings.HasSuffix(p, ".flac")) && !d.IsDir() {
			matchedTags := make([]core.CollectionTag, 0)
			for _, tag := range collectionTags {
				if strings.Contains(tag.FilePath, p) {
					matchedTags = append(matchedTags, tag)
				}
			}
			s.State.pushChoice(core.NewTaggedDirEntry(p, matchedTags, false))
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to read samples directory: %v", err)
	}
}
