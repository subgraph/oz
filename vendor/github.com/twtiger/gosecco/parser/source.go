package parser

import (
	"io/ioutil"
	"strings"

	"github.com/twtiger/gosecco/tree"
)

// Source represents a source of parsing data
type Source interface {
	Parse() (tree.RawPolicy, error)
}

// FileSource represents the source of parsing coming from a file
type FileSource struct {
	// Filename is the name of the file to parse definitions from
	Filename string
}

// StringSource contains the definitions as a string
type StringSource struct {
	// Name is the name to report for this string during parsing errors
	Name string
	// Content is the actual string containing definitions
	Content string
}

// CombinedSource allow you to combine more than one source and have them parsed as a unit
type CombinedSource struct {
	// Sources is a list of the sources to parse
	Sources []Source
}

// CombineSources returns a CombinedSource with all the given sources
func CombineSources(s ...Source) *CombinedSource {
	return &CombinedSource{s}
}

// Parse implements the Source interface by parsing the file
func (s *FileSource) Parse() (tree.RawPolicy, error) {
	file, err := ioutil.ReadFile(s.Filename)
	if err != nil {
		return tree.RawPolicy{}, err
	}
	return parseLines(s.Filename, strings.Split(string(file), "\n"))
}

// Parse implements the Source interface by parsing the string
func (s *StringSource) Parse() (tree.RawPolicy, error) {
	return parseLines(s.Name, strings.Split(s.Content, "\n"))
}

// Parse implements the Source interface by parsing each one of the sources
func (s *CombinedSource) Parse() (tree.RawPolicy, error) {
	var result []interface{}
	for _, s := range s.Sources {
		rp, e := s.Parse()
		if e != nil {
			return tree.RawPolicy{}, e
		}
		result = append(result, rp.RuleOrMacros...)
	}
	return tree.RawPolicy{result}, nil
}
