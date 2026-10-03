package challenge

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2"
)

type Learning struct {
	Requires []string `toml:"requires"`
	Teaches  []string `toml:"teaches"`
}

type Concept struct {
	ID       string   `toml:"id"`
	Title    string   `toml:"title"`
	Requires []string `toml:"requires"`
	Related  []string `toml:"related"`
}

var conceptID = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func LoadConcepts(repo string) (map[string]Concept, error) {
	root, err := os.OpenRoot(repo)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	f, err := root.Open("knowledge/catalog.toml")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("knowledge catalog must be a regular file")
	}
	data, err := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 1<<20 || !utf8.Valid(data) {
		return nil, errors.New("knowledge catalog must be bounded UTF-8")
	}
	var catalog struct {
		Concepts []Concept `toml:"concepts"`
	}
	if err := toml.Unmarshal(data, &catalog); err != nil {
		return nil, err
	}
	if len(catalog.Concepts) > 256 {
		return nil, errors.New("too many knowledge concepts")
	}
	result := make(map[string]Concept, len(catalog.Concepts))
	for _, item := range catalog.Concepts {
		if !conceptID.MatchString(item.ID) || len(item.ID) > 64 || item.Title == "" {
			return nil, errors.New("invalid knowledge concept")
		}
		if _, exists := result[item.ID]; exists {
			return nil, errors.New("duplicate knowledge concept")
		}
		item.Requires = append([]string{}, item.Requires...)
		item.Related = append([]string{}, item.Related...)
		result[item.ID] = item
	}
	for _, item := range result {
		for _, id := range append(append([]string{}, item.Requires...), item.Related...) {
			if _, exists := result[id]; !exists || id == item.ID {
				return nil, fmt.Errorf("invalid concept reference %q", id)
			}
		}
	}
	return result, nil
}
