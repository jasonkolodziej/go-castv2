package parse

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

// type TokenSet []token.Token

type Token = string
type TokenSet []Token
type Parser interface {
	defaultTokens() TokenSet
	Tokens(setWith *TokenSet) *TokenSet
	// Start parsing
	Parse(s *bufio.Scanner) Parser
	// Create a new parser
	NewParser() Parser
}

type ParserFunc func() (kvTemplate *KeyValue, sectionStartDel, sectionNameDel, endSectionDel string)

type Spacing int

const (
	SUB_TRAIL Spacing = iota - 2
	SUB_LEAD
	DEFAULT
	ADD_LEAD
	ADD_TRAIL
)

func Reverse(str []string) []string {
	var x []string = str
	slices.Reverse(x)
	return x
}

func NoEmpty(strs []string) []string {
	var r []string
	for _, str := range strs {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func Append(strs []string, s string) []string {
	var r []string
	for _, str := range strs {
		r = append(r, s+str)
	}
	return r
}

func MarkWhere(strs []string, where string) []int {
	var r []int
	for i, str := range strs {
		if strings.Contains(str, where) {
			r = append(r, i)
		}
	}
	return r
}

// MarkWheres marks a string slice index where all strings/characters occur
func MarkWheres(strs []string, allWhere ...string) []int {
	var r []int
	for i, str := range strs {
		g := true
		for _, id := range allWhere {
			if !strings.Contains(str, id) {
				g = false
				break
			}
		}
		if g {
			r = append(r, i)
		}
	}
	return r
}

// Custom split function. This will split string at 'sbustring' i.e # or // etc....
func SplitAt(substring string) func(data []byte, atEOF bool) (advance int, token []byte, err error) {
	searchBytes := []byte(substring)
	searchLength := len(substring)
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		dataLen := len(data)

		// Return Nothing if at the end of file or no data passed.
		if atEOF && dataLen == 0 {
			return 0, nil, nil
		}

		// Find next separator and return token.
		if i := bytes.Index(data, searchBytes); i >= 0 {
			return i + searchLength, data[0:i], nil
		}

		// If we're at EOF, we have a final, non-terminated line. Return it.
		if atEOF {
			return dataLen, data, nil
		}

		// Request more data.
		return 0, nil, nil
	}
}

func SplitUpSections(rawData *string, endOfSectionDelimiter string, kvTemplate *KeyValue) Sections {
	data := NoEmpty(strings.Split(*rawData, endOfSectionDelimiter))
	secs := make([]*Section, len(data))
	for i := range data {
		secs[i] = &Section{rawContent: &data[i], endingToken: endOfSectionDelimiter}
		if kvTemplate != nil {
			secs[i].SetKvTemplate(*kvTemplate)
		}
	}
	return secs
}

func Parse(rawData *string, kvTemplate *KeyValue, sectionStartDel, sectionNameDel, endSectionDel Token) Sections {
	sections := SplitUpSections(rawData, endSectionDel, kvTemplate)
	for _, section := range sections {
		section.Parse(sectionStartDel, sectionNameDel)
	}
	return sections
}

func LoadFile(wd, filename string) (f *os.File, size int64, err error) {
	if wd == "" {
		wd, _ = os.Getwd()
	}
	f, err = os.Open(wd + filename)
	if err != nil {
		return
	}
	fInfo, _ := f.Stat()
	size = fInfo.Size()
	return
}

func ParseFile(filename string, parser ParserFunc) (sections Sections, err error) {
	f, _, err := LoadFile(string(filename[0]), string(filename[1:]))
	if err != nil {
		return
	}
	return ParseOpenedFile(f, parser)
}

func ParseOpenedFile(f *os.File, parser ParserFunc) (sections Sections, err error) {
	defer f.Close()
	reader, err := io.ReadAll(f)
	if err != nil {
		return
	}
	reading := string(reader)
	kvTemplate, sectionStartDel, sectionNameDel, endSectionDel := parser()
	return Parse(&reading, kvTemplate, sectionStartDel, sectionNameDel, endSectionDel), nil
}

func WriteOut(sections Sections, wd, newFilename string) error {
	if wd == "" {
		wd, _ = os.Getwd()
	}
	f, err := os.Create(wd + newFilename)
	if err != nil {
		return err
	}
	// f.WriteTo()
	for _, section := range sections {
		_, err := section.WriteTo(f)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ss *Sections) UpdateValueAt(keyNamePath string, value any) error {
	keyNames := strings.Split(keyNamePath, ".")
	for _, section := range *ss {
		if section.Name != keyNames[0] {
			continue
		} // * section was found
		valOk := section.KeyValues.Get(keyNames[1])
		if valOk != nil {
			if err := valOk.SetValue(value); err != nil {
				return fmt.Errorf("error: UpdateValueAt cannot update %s with %v", keyNamePath, value)
			}
			valOk.SetUncommented()
			return nil
		}
	}
	return fmt.Errorf("error: UpdateValueAt key %s not found", keyNamePath)
}

func (k *KeyValues) Get(kvName string) *KeyValue {
	for _, v := range *k {
		if v.KeyName == kvName {
			return &v
		}
	}
	return nil
}
