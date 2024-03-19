package parse

import (
	"bufio"
	"slices"
	"strings"
)

// type TokenSet []token.Token

type Token string
type TokenSet []Token
type Parser interface {
	defaultTokens() TokenSet
	Tokens(setWith *TokenSet) *TokenSet
	// Start parsing
	Parse(s *bufio.Scanner) Parser
	// Create a new parser
	NewParser() Parser
}

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
