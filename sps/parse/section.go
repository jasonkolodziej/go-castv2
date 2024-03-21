package parse

import (
	"slices"
	"strings"
)

type Section struct {
	Name        string
	KeyValues   []KeyValue
	Description []string
	// use for internal purposes
	rawContent    *string
	startingToken string
	endingToken   string
	kvTemplate    *KeyValue
}

func (c Section) defaultTokens() TokenSet {
	return TokenSet{"=", "{", "};"}
}

func (c *Section) CreateKeyValues(rawKvs, newKvDelimiter string) []KeyValue {
	// Create KV Lines
	kvLines := CreateKvLines(rawKvs, newKvDelimiter)
	// filter out where there are multiline comments
	kvIdxs := MarkWheres(kvLines, c.kvTemplate.GetDelimitersForAssertion()...)
	c.KeyValues = CreateKvs(kvLines, kvIdxs, c)
	for _, v := range c.KeyValues {
		for i2, v2 := range v.Description {
			v.Description[i2] = strings.Trim(v2, v.commentDelimiter)
		}
	}
	return c.KeyValues
}

func (c Section) KV() []KeyValue {
	return c.KeyValues
}

func (c *Section) SetKvTemplate(kv KeyValue) {
	c.kvTemplate = &kv
}

// DEPRECATED
func HandleSection(name string, description []string, rawKvs string,
	kvDelimeters string, vDelimeters ...string) Section {
	sec := &Section{Name: name, Description: description}
	// Add and clean Section description
	for i, v := range sec.Description {
		sec.Description[i] = strings.Trim(v, "/ ")
	}
	// Create KV Lines
	kvLines := CreateKvLines(rawKvs, kvDelimeters)
	// filter out where there are multiline comments
	kvIdxs := MarkWheres(kvLines, vDelimeters...)
	sec.KeyValues = CreateKvs(kvLines, kvIdxs, sec)
	for _, v := range sec.KeyValues {
		for i2, v2 := range v.Description {
			v.Description[i2] = strings.Trim(v2, "/ ")
		}
	}
	// println((kv.comments))
	return *sec
}

func (sec *Section) HandleSection(description []string, rawKvs string,
	kvDelimeters string) *Section {
	sec.Description = description
	// Add and clean Section description
	for i, v := range sec.Description {
		// TODO: Handle comment in section if differs
		sec.Description[i] = strings.Trim(v, "/ ")
	}
	sec.CreateKeyValues(rawKvs, kvDelimeters)
	// // Create KV Lines
	// kvLines := CreateKvLines(rawKvs, kvDelimeters)
	// // filter out where there are multiline comments
	// kvIdxs := MarkWheres(kvLines, vDelimeters...)
	// sec.KeyValues = CreateKvs(kvLines, kvIdxs, sec)
	// for _, v := range sec.KeyValues {
	// 	for i2, v2 := range v.Description {
	// 		v.Description[i2] = strings.Trim(v2, "/ ")
	// 	}
	// }
	// println((kv.comments))
	return sec
}

// ! Try pointers & use for each

func (s *Section) FindBeginningOfSection(startOfSectionDelimiter string, sectionNameDelimiter *string) (beginSection []string, sectionContent []string) {
	s.startingToken = startOfSectionDelimiter
	sectionContent = strings.Split(*s.rawContent, s.startingToken)
	if len(sectionContent) > 2 {
		// Handle subsections
		println("err: Splitting BeginningOfSection(); subsections may be present")
	} else if len(sectionContent) < 2 {
		// Handle subsections
		return nil, nil
	}
	// split up new lines

	test := strings.Split(strings.ReplaceAll(sectionContent[0], "\r", ""), "\n")
	beginSection = NoEmpty(test)
	// if the first line in beginSection contains = but NOT comment delimiters
	if strings.Contains(beginSection[0], *sectionNameDelimiter) && !strings.Contains(beginSection[0], "//") {
		s.Name = beginSection[0]
	} else if s.Name == "" {
		// assume there are comments before section Name, so reverse
		slices.Reverse(beginSection)
	}
	if s.Name == "" && strings.Contains(beginSection[0], *sectionNameDelimiter) && !strings.Contains(beginSection[0], "//") {
		s.Name = beginSection[0]
		beginSection = beginSection[1:]
		slices.Reverse(beginSection)
	} else {
		// worst case search
		println("err: FindBeginOfSection WORST CASE")
	}
	// Trim name
	s.Name = strings.Trim(s.Name, *sectionNameDelimiter)
	s.Description = beginSection

	return
}

func (s *Section) Parse(sectionStartDel, sectionNameDel string) *Section {
	sDescription, sectionContent := s.FindBeginningOfSection(sectionStartDel, &sectionNameDel)
	if len(sectionContent) == 2 {
		// Handle subsections
		s = s.HandleSection(sDescription, sectionContent[1], "")
	}
	return s
}

func (c *Section) NewParser() Parser {
	return &Comment{}
}
