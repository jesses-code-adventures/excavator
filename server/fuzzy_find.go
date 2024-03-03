package server

import (
	"log"
	"strings"
    "path/filepath"

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
func (s *Server) FuzzyFind(search string, fromRoot bool) []core.SelectableListItem {
	log.Println("in server fuzzy search fn")
	var dir string
	var files []string
	var samples []core.SelectableListItem
	if len(search) == 0 {
		return make([]core.SelectableListItem, 0)
	}
	if fromRoot {
		dir = s.State.Root
	} else {
		dir = s.State.Dir
	}
	collectionTags := s.FuzzyFindCollectionTags(search)
	log.Println("collection tags", collectionTags)
	log.Println("searching for: ", search)
	log.Println("dir: ", dir)
	// err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if !ContainsAllSubstrings(path, search) || strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".asd") {
	// 		return nil
	// 	}
	// 	if (strings.HasSuffix(path, ".wav") || strings.HasSuffix(path, ".mp3") || strings.HasSuffix(path, ".flac")) && !d.IsDir() {
	// 		entries = append(entries, d)
	// 	}
	// 	files = append(files, d)
	// 	matchedTags := make([]core.CollectionTag, 0)
	// 	for _, tag := range collectionTags {
	// 		if strings.Contains(tag.FilePath, path) {
	// 			matchedTags = append(matchedTags, tag)
	// 		}
	// 	}
	// 	s.State.pushChoice(core.TaggedDirentry{Path: path, Tags: matchedTags, Dir: false})
	// 	return nil
	// })
    if !strings.HasSuffix(dir, "/") {
        dir = dir + "/"
    }
    globStr := dir + "*"
    log.Println("globstr: ", globStr)
	matches, err := filepath.Glob(globStr)
	if err != nil {
		log.Fatalf("Failed to read samples directory: %v", err)
	}
	for _, match := range matches {
        log.Println("match: ", match)
		files = append(files, match)
		// matchedTags := make([]core.CollectionTag, 0)
		// for _, tag := range collectionTags {
		// 	if strings.Contains(tag.FilePath, match) {
		// 		matchedTags = append(matchedTags, tag)
		// 	}
		// }
		// s.State.pushChoice(core.TaggedDirentry{FilePath: match, Tags: matchedTags, Dir: false})
		// return nil
	}
	return samples
}
