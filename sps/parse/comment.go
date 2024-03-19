package parse

import "bufio"

type Comment struct {
	Parser
	maybeKV   bool
	commented bool
	t         *TokenSet
	p         *string
}

func (c *Comment) defaultTokens() TokenSet {
	return TokenSet{"//", "; //", ";//"}
}

func (c *Comment) Tokens(setWith *TokenSet) *TokenSet {
	if setWith != nil {
		c.t = setWith
	}
	return c.t
}

func (c Comment) isKeyValueComment() bool {
	return c.maybeKV
}

func (c *Comment) Parse(s *bufio.Scanner) Parser {
	if s == nil {
		return c
	}
	return c
}

func (c *Comment) NewParser() Parser {
	return &Comment{}
}
