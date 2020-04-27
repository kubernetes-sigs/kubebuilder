/*
Copyright 2019 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package spansql contains types and a parser for the Cloud Spanner SQL dialect.

To parse, use one of the Parse functions (ParseDDL, ParseDDLStmt, ParseQuery, etc.).

Sources:
	https://cloud.google.com/spanner/docs/lexical
	https://cloud.google.com/spanner/docs/query-syntax
	https://cloud.google.com/spanner/docs/data-definition-language
*/
package spansql

/*
This file is structured as follows:

- There are several exported ParseFoo functions that accept an input string
  and return a type defined in types.go. This is the principal API of this package.
  These functions are implemented as wrappers around the lower-level functions,
  with additional checks to ensure things such as input exhaustion.
- The token and parser types are defined. These constitute the lexical token
  and parser machinery. parser.next is the main way that other functions get
  the next token, with parser.back providing a single token rewind, and
  parser.sniff, parser.eat and parser.expect providing lookahead helpers.
- The parseFoo methods are defined, matching the SQL grammar. Each consumes its
  namesake production from the parser. There are also some fooParser helper vars
  defined that abbreviate the parsing of some of the regular productions.
*/

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

const debug = false

func debugf(format string, args ...interface{}) {
	if !debug {
		return
	}
	fmt.Fprintf(os.Stderr, "spansql debug: "+format+"\n", args...)
}

// ParseDDL parses a DDL file.
func ParseDDL(s string) (DDL, error) {
	p := newParser(s)

	var ddl DDL
	for {
		p.skipSpace()
		if p.done {
			break
		}

		stmt, err := p.parseDDLStmt()
		if err != nil {
			return DDL{}, err
		}
		ddl.List = append(ddl.List, stmt)

		tok := p.next()
		if tok.err == io.EOF {
			break
		} else if tok.err != nil {
			return DDL{}, tok.err
		}
		if tok.value == ";" {
			continue
		} else {
			return DDL{}, p.errorf("unexpected token %q", tok.value)
		}
	}
	if p.Rem() != "" {
		return DDL{}, fmt.Errorf("unexpected trailing contents %q", p.Rem())
	}
	return ddl, nil
}

// ParseDDLStmt parses a single DDL statement.
func ParseDDLStmt(s string) (DDLStmt, error) {
	p := newParser(s)
	stmt, err := p.parseDDLStmt()
	if err != nil {
		return nil, err
	}
	if p.Rem() != "" {
		return nil, fmt.Errorf("unexpected trailing contents %q", p.Rem())
	}
	return stmt, nil
}

// ParseQuery parses a query string.
func ParseQuery(s string) (Query, error) {
	p := newParser(s)
	q, err := p.parseQuery()
	if err != nil {
		return Query{}, err
	}
	if p.Rem() != "" {
		return Query{}, fmt.Errorf("unexpected trailing query contents %q", p.Rem())
	}
	return q, nil
}

type token struct {
	value string
	err   error

	typ     tokenType
	int64   int64
	float64 float64
	string  string // unquoted form
}

type tokenType int

const (
	unknownToken tokenType = iota
	int64Token
	float64Token
	stringToken
	bytesToken
)

func (t *token) String() string {
	if t.err != nil {
		return fmt.Sprintf("parse error: %v", t.err)
	}
	return strconv.Quote(t.value)
}

type parser struct {
	s      string // Remaining input.
	done   bool   // Whether the parsing is finished (success or error).
	backed bool   // Whether back() was called.
	cur    token
}

func newParser(s string) *parser {
	return &parser{
		s: s,
	}
}

// Rem returns the unparsed remainder, ignoring space.
func (p *parser) Rem() string {
	rem := p.s
	if p.backed {
		rem = p.cur.value + rem
	}
	i := 0
	for ; i < len(rem); i++ {
		if !isSpace(rem[i]) {
			break
		}
	}
	return rem[i:]
}

func (p *parser) String() string {
	if p.backed {
		return fmt.Sprintf("next tok: %s (rem: %q)", &p.cur, p.s)
	}
	return fmt.Sprintf("rem: %q", p.s)
}

func (p *parser) errorf(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	p.cur.err = err
	p.done = true
	return err
}

func isInitialIdentifierChar(c byte) bool {
	// https://cloud.google.com/spanner/docs/lexical#identifiers
	switch {
	case 'A' <= c && c <= 'Z':
		return true
	case 'a' <= c && c <= 'z':
		return true
	case c == '_':
		return true
	}
	return false
}

func isIdentifierChar(c byte) bool {
	// https://cloud.google.com/spanner/docs/lexical#identifiers
	// This doesn't apply the restriction that an identifier cannot start with [0-9],
	// nor does it check against reserved keywords.
	switch {
	case 'A' <= c && c <= 'Z':
		return true
	case 'a' <= c && c <= 'z':
		return true
	case '0' <= c && c <= '9':
		return true
	case c == '_':
		return true
	}
	return false
}

func isHexDigit(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

func isOctalDigit(c byte) bool {
	return '0' <= c && c <= '7'
}

func (p *parser) consumeNumber() {
	/*
		int64_value:
			{ decimal_value | hex_value }

		decimal_value:
			[-]0—9+

		hex_value:
			[-]0[xX]{0—9|a—f|A—F}+

		(float64_value is not formally specified)

		float64_value :=
			  [+-]DIGITS.[DIGITS][e[+-]DIGITS]
			| [DIGITS].DIGITS[e[+-]DIGITS]
			| DIGITSe[+-]DIGITS
	*/

	i, neg, base := 0, false, 10
	float, e, dot := false, false, false
	if p.s[i] == '-' {
		neg = true
		i++
	} else if p.s[i] == '+' {
		// This isn't in the formal grammar, but is mentioned informally.
		// https://cloud.google.com/spanner/docs/lexical#integer-literals
		i++
	}
	if strings.HasPrefix(p.s[i:], "0x") || strings.HasPrefix(p.s[i:], "0X") {
		base = 16
		i += 2
	}
	d0 := i
digitLoop:
	for i < len(p.s) {
		switch c := p.s[i]; {
		case '0' <= c && c <= '9':
			i++
		case base == 16 && 'A' <= c && c <= 'F':
			i++
		case base == 16 && 'a' <= c && c <= 'f':
			i++
		case base == 10 && (c == 'e' || c == 'E'):
			if e {
				p.errorf("bad token %q", p.s[:i])
				return
			}
			// Switch to consuming float.
			float, e = true, true
			i++

			if i < len(p.s) && (p.s[i] == '+' || p.s[i] == '-') {
				i++
			}
		case base == 10 && c == '.':
			if dot || e { // any dot must come before E
				p.errorf("bad token %q", p.s[:i])
				return
			}
			// Switch to consuming float.
			float, dot = true, true
			i++
		default:
			break digitLoop
		}
	}
	if d0 == i {
		p.errorf("no digits in numeric literal")
		return
	}
	sign := ""
	if neg {
		sign = "-"
	}
	p.cur.value, p.s = p.s[:i], p.s[i:]
	var err error
	if float {
		p.cur.typ = float64Token
		p.cur.float64, err = strconv.ParseFloat(sign+p.cur.value[d0:], 64)
	} else {
		p.cur.typ = int64Token
		p.cur.int64, err = strconv.ParseInt(sign+p.cur.value[d0:], base, 64)
	}
	if err != nil {
		p.errorf("bad numeric literal %q: %v", p.cur.value, err)
	}
}

func (p *parser) consumeString() {
	// https://cloud.google.com/spanner/docs/lexical#string-and-bytes-literals

	delim := p.stringDelimiter()
	if p.cur.err != nil {
		return
	}

	p.cur.string, p.cur.err = p.consumeStringContent(delim, false, true, "string literal")
	p.cur.typ = stringToken
}

func (p *parser) consumeRawString() {
	// https://cloud.google.com/spanner/docs/lexical#string-and-bytes-literals

	p.s = p.s[1:] // consume 'R'
	delim := p.stringDelimiter()
	if p.cur.err != nil {
		return
	}

	p.cur.string, p.cur.err = p.consumeStringContent(delim, true, true, "raw string literal")
	p.cur.typ = stringToken
}

func (p *parser) consumeBytes() {
	// https://cloud.google.com/spanner/docs/lexical#string-and-bytes-literals

	p.s = p.s[1:] // consume 'B'
	delim := p.stringDelimiter()
	if p.cur.err != nil {
		return
	}

	p.cur.string, p.cur.err = p.consumeStringContent(delim, false, false, "bytes literal")
	p.cur.typ = bytesToken
}

func (p *parser) consumeRawBytes() {
	// https://cloud.google.com/spanner/docs/lexical#string-and-bytes-literals

	p.s = p.s[2:] // consume 'RB'
	delim := p.stringDelimiter()
	if p.cur.err != nil {
		return
	}

	p.cur.string, p.cur.err = p.consumeStringContent(delim, true, false, "raw bytes literal")
	p.cur.typ = bytesToken
}

// stringDelimiter returns the opening string delimiter.
func (p *parser) stringDelimiter() string {
	c := p.s[0]
	if c != '"' && c != '\'' {
		p.errorf("invalid string literal")
		return ""
	}
	// Look for triple.
	if len(p.s) >= 3 && p.s[1] == c && p.s[2] == c {
		return p.s[:3]
	}
	return p.s[:1]
}

// consumeStringContent consumes a string-like literal, including its delimiters.
//
//   - delim is opening delimiter.
//   - raw is true on consuming raw string, otherwise it is false.
//   - unicode is true if unicode escape sequence (\uXXXX or \UXXXXXXXX), otherwise it is false.
//   - name is current consuming token name.
//
// It is designed for consuming string, bytes literals, and also backquoted identifiers.
func (p *parser) consumeStringContent(delim string, raw, unicode bool, name string) (string, error) {
	// https://cloud.google.com/spanner/docs/lexical#string-and-bytes-literals

	if len(delim) == 3 {
		name = "triple-quoted " + name
	}

	i := len(delim)
	var content []byte

	for i < len(p.s) {
		if strings.HasPrefix(p.s[i:], delim) {
			i += len(delim)
			p.s = p.s[i:]
			return string(content), nil
		}

		if p.s[i] == '\\' {
			i++
			if i >= len(p.s) {
				return "", p.errorf("unclosed %s", name)
			}

			if raw {
				content = append(content, '\\', p.s[i])
				i++
				continue
			}

			switch p.s[i] {
			case 'a':
				i++
				content = append(content, '\a')
			case 'b':
				i++
				content = append(content, '\b')
			case 'f':
				i++
				content = append(content, '\f')
			case 'n':
				i++
				content = append(content, '\n')
			case 'r':
				i++
				content = append(content, '\r')
			case 't':
				i++
				content = append(content, '\t')
			case 'v':
				i++
				content = append(content, '\v')
			case '\\':
				i++
				content = append(content, '\\')
			case '?':
				i++
				content = append(content, '?')
			case '"':
				i++
				content = append(content, '"')
			case '\'':
				i++
				content = append(content, '\'')
			case '`':
				i++
				content = append(content, '`')
			case 'x', 'X':
				i++
				if !(i+1 < len(p.s) && isHexDigit(p.s[i]) && isHexDigit(p.s[i+1])) {
					return "", p.errorf("illegal escape sequence: hex escape sequence must be followed by 2 hex digits")
				}
				c, err := strconv.ParseUint(p.s[i:i+2], 16, 64)
				if err != nil {
					return "", p.errorf("illegal escape sequence: invalid hex digits: %q: %v", p.s[i:i+2], err)
				}
				content = append(content, byte(c))
				i += 2
			case 'u', 'U':
				t := p.s[i]
				if !unicode {
					return "", p.errorf("illegal escape sequence: \\%c", t)
				}

				i++
				size := 4
				if t == 'U' {
					size = 8
				}
				if i+size-1 >= len(p.s) {
					return "", p.errorf("illegal escape sequence: \\%c escape sequence must be followed by %d hex digits", t, size)
				}
				for j := 0; j < size; j++ {
					if !isHexDigit(p.s[i+j]) {
						return "", p.errorf("illegal escape sequence: \\%c escape sequence must be followed by %d hex digits", t, size)
					}
				}
				c, err := strconv.ParseUint(p.s[i:i+size], 16, 64)
				if err != nil {
					return "", p.errorf("illegal escape sequence: invalid \\%c digits: %q: %v", t, p.s[i:i+size], err)
				}
				if 0xD800 <= c && c <= 0xDFFF || 0x10FFFF < c {
					return "", p.errorf("illegal escape sequence: invalid codepoint: %x", c)
				}
				var buf [utf8.UTFMax]byte
				n := utf8.EncodeRune(buf[:], rune(c))
				content = append(content, buf[:n]...)
				i += size
			case '0', '1', '2', '3', '4', '5', '6', '7':
				if !(i+2 < len(p.s) && isOctalDigit(p.s[i+1]) && isOctalDigit(p.s[i+2])) {
					return "", p.errorf("illegal escape sequence: octal escape sequence must be followed by 3 octal digits")
				}
				c, err := strconv.ParseUint(p.s[i:i+3], 8, 64)
				if err != nil {
					return "", p.errorf("illegal escape sequence: invalid octal digits: %q: %v", p.s[i:i+3], err)
				}
				if c >= 256 {
					return "", p.errorf("illegal escape sequence: octal digits overflow: %q (%d)", p.s[i:i+3], c)
				}
				content = append(content, byte(c))
				i += 3
			default:
				return "", p.errorf("illegal escape sequence: \\%c", p.s[i])
			}

			continue
		}

		if p.s[i] == '\n' && len(delim) != 3 { // newline is only allowed inside triple-quoted.
			return "", p.errorf("newline forbidden in %s", name)
		}

		content = append(content, p.s[i])
		i++
	}

	return "", p.errorf("unclosed %s", name)
}

var operators = map[string]bool{
	// TODO: There's duplication here with symbolicOperators,
	// but this should go away with more bespoke handling inside parser.advance.
	"<":  true,
	"<=": true,
	">":  true,
	">=": true,
	"=":  true,
	"!=": true,
	"<>": true,
}

func isSpace(c byte) bool {
	// Per https://cloud.google.com/spanner/docs/lexical, informally,
	// whitespace is defined as "space, backspace, tab, newline".
	switch c {
	case ' ', '\b', '\t', '\n':
		return true
	}
	return false
}

// skipSpace skips past any space or comments.
func (p *parser) skipSpace() bool {
	i := 0
	for i < len(p.s) {
		if isSpace(p.s[i]) {
			i++
			continue
		}
		// Comments.
		term := ""
		if p.s[i] == '#' {
			term = "\n"
		} else if i+1 < len(p.s) && p.s[i] == '-' && p.s[i+1] == '-' {
			term = "\n"
		} else if i+1 < len(p.s) && p.s[i] == '/' && p.s[i+1] == '*' {
			term = "*/"
		}
		if term == "" {
			break
		}
		ti := strings.Index(p.s[i:], term)
		if ti < 0 {
			p.errorf("unterminated comment")
			return false
		}
		i += ti + len(term)
	}
	p.s = p.s[i:]
	if p.s == "" {
		p.done = true
	}
	return i > 0
}

// advance moves the parser to the next token, which will be available in p.cur.
func (p *parser) advance() {
	p.skipSpace()
	if p.done {
		return
	}
	p.cur.err = nil
	p.cur.typ = unknownToken
	// TODO: backtick (`) for quoted identifiers.
	// TODO: array, struct, date, timestamp literals
	switch p.s[0] {
	case ',', ';', '(', ')', '*':
		// Single character symbol.
		p.cur.value, p.s = p.s[:1], p.s[1:]
		return
	// String literal prefix.
	case 'B', 'b', 'R', 'r', '"', '\'':
		// "B", "b", "BR", "Rb" etc are valid string literal prefix, however "BB", "rR" etc are not.
		raw, bytes := false, false
		for i := 0; i < 4 && i < len(p.s); i++ {
			switch {
			case !raw && (p.s[i] == 'R' || p.s[i] == 'r'):
				raw = true
				continue
			case !bytes && (p.s[i] == 'B' || p.s[i] == 'b'):
				bytes = true
				continue
			case p.s[i] == '"' || p.s[i] == '\'':
				switch {
				case raw && bytes:
					p.consumeRawBytes()
				case raw:
					p.consumeRawString()
				case bytes:
					p.consumeBytes()
				default:
					p.consumeString()
				}
				return
			}
			break
		}
	}
	if p.s[0] == '@' || isInitialIdentifierChar(p.s[0]) {
		// Start consuming identifier.
		i := 1
		for i < len(p.s) && isIdentifierChar(p.s[i]) {
			i++
		}
		p.cur.value, p.s = p.s[:i], p.s[i:]
		return
	}
	if len(p.s) >= 2 && (p.s[0] == '+' || p.s[0] == '-' || p.s[0] == '.') && ('0' <= p.s[1] && p.s[1] <= '9') {
		// [-+.] followed by a digit.
		p.consumeNumber()
		return
	}
	if '0' <= p.s[0] && p.s[0] <= '9' {
		p.consumeNumber()
		return
	}
	// More single character symbols.
	// These are deliberately below the numeric literal parsing.
	switch p.s[0] {
	case '-', '+':
		p.cur.value, p.s = p.s[:1], p.s[1:]
		return
	}

	// Look for operator (two or one bytes).
	for i := 2; i >= 1; i-- {
		if i <= len(p.s) && operators[p.s[:i]] {
			p.cur.value, p.s = p.s[:i], p.s[i:]
			return
		}
	}

	p.errorf("unexpected byte %#x", p.s[0])
}

// back steps the parser back one token. It cannot be called twice in succession.
func (p *parser) back() {
	if p.backed {
		panic("parser backed up twice")
	}
	p.done = false
	p.backed = true
	// If an error was being recovered, we wish to ignore the error.
	// Don't do that for io.EOF since that'll be returned next.
	if p.cur.err != io.EOF {
		p.cur.err = nil
	}
}

// next returns the next token.
func (p *parser) next() *token {
	if p.backed || p.done {
		p.backed = false
		return &p.cur
	}
	p.advance()
	if p.done && p.cur.err == nil {
		p.cur.value = ""
		p.cur.err = io.EOF
	}
	return &p.cur
}

// sniff reports whether the next N tokens are as specified.
func (p *parser) sniff(want ...string) bool {
	// Store current parser state and restore on the way out.
	orig := *p
	defer func() { *p = orig }()

	for _, w := range want {
		tok := p.next()
		if tok.err != nil || tok.value != w {
			return false
		}
	}
	return true
}

// eat reports whether the next N tokens are as specified,
// then consumes them.
func (p *parser) eat(want ...string) bool {
	// Store current parser state so we can restore if we get a failure.
	orig := *p

	for _, w := range want {
		tok := p.next()
		if tok.err != nil || tok.value != w {
			// Mismatch.
			*p = orig
			return false
		}
	}
	return true
}

func (p *parser) expect(want string) error {
	tok := p.next()
	if tok.err != nil {
		return tok.err
	}
	if tok.value != want {
		return p.errorf("got %q while expecting %q", tok.value, want)
	}
	return nil
}

func (p *parser) parseDDLStmt() (DDLStmt, error) {
	debugf("parseDDLStmt: %v", p)

	/*
		statement:
			{ create_database | create_table | create_index | alter_table | drop_table | drop_index }
	*/

	// TODO: support create_database

	if p.sniff("CREATE", "TABLE") {
		ct, err := p.parseCreateTable()
		return ct, err
	} else if p.sniff("CREATE") {
		// The only other statement starting with CREATE is CREATE INDEX,
		// which can have UNIQUE or NULL_FILTERED as the token after CREATE.
		ci, err := p.parseCreateIndex()
		return ci, err
	} else if p.sniff("ALTER", "TABLE") {
		a, err := p.parseAlterTable()
		return a, err
	} else if p.eat("DROP") {
		// These statements are simple.
		//	DROP TABLE table_name
		//	DROP INDEX index_name
		tok := p.next()
		if tok.err != nil {
			return nil, tok.err
		}
		kind := tok.value
		if kind != "TABLE" && kind != "INDEX" {
			return nil, p.errorf("got %q, want TABLE or INDEX", kind)
		}
		name, err := p.parseTableOrIndexOrColumnName()
		if err != nil {
			return nil, err
		}
		if kind == "TABLE" {
			return DropTable{Name: name}, nil
		}
		return DropIndex{Name: name}, nil
	}

	return nil, p.errorf("unknown DDL statement")
}

func (p *parser) parseCreateTable() (CreateTable, error) {
	debugf("parseCreateTable: %v", p)

	/*
		CREATE TABLE table_name(
			[column_def, ...] )
			primary_key [, cluster]

		primary_key:
			PRIMARY KEY ( [key_part, ...] )

		cluster:
			INTERLEAVE IN PARENT table_name [ ON DELETE { CASCADE | NO ACTION } ]
	*/

	if err := p.expect("CREATE"); err != nil {
		return CreateTable{}, err
	}
	if err := p.expect("TABLE"); err != nil {
		return CreateTable{}, err
	}
	tname, err := p.parseTableOrIndexOrColumnName()
	if err != nil {
		return CreateTable{}, err
	}

	ct := CreateTable{Name: tname}
	err = p.parseCommaList(func(p *parser) error {
		cd, err := p.parseColumnDef()
		if err != nil {
			return err
		}
		ct.Columns = append(ct.Columns, cd)
		return nil
	})
	if err != nil {
		return CreateTable{}, err
	}

	if err := p.expect("PRIMARY"); err != nil {
		return CreateTable{}, err
	}
	if err := p.expect("KEY"); err != nil {
		return CreateTable{}, err
	}
	ct.PrimaryKey, err = p.parseKeyPartList()
	if err != nil {
		return CreateTable{}, err
	}

	if p.eat(",", "INTERLEAVE") {
		if err := p.expect("IN"); err != nil {
			return CreateTable{}, err
		}
		if err := p.expect("PARENT"); err != nil {
			return CreateTable{}, err
		}
		pname, err := p.parseTableOrIndexOrColumnName()
		if err != nil {
			return CreateTable{}, err
		}
		ct.Interleave = &Interleave{
			Parent:   pname,
			OnDelete: NoActionOnDelete,
		}
		// The ON DELETE clause is optional; it defaults to NoActionOnDelete.
		if p.eat("ON", "DELETE") {
			od, err := p.parseOnDelete()
			if err != nil {
				return CreateTable{}, err
			}
			ct.Interleave.OnDelete = od
		}
	}

	return ct, nil
}

func (p *parser) parseCreateIndex() (CreateIndex, error) {
	debugf("parseCreateIndex: %v", p)

	/*
		CREATE [UNIQUE] [NULL_FILTERED] INDEX index_name
			ON table_name ( key_part [, ...] ) [ storing_clause ] [ , interleave_clause ]

		index_name:
			{a—z|A—Z}[{a—z|A—Z|0—9|_}+]

		storing_clause:
			STORING ( column_name [, ...] )

		interleave_clause:
			INTERLEAVE IN table_name
	*/

	var unique, nullFiltered bool

	if err := p.expect("CREATE"); err != nil {
		return CreateIndex{}, err
	}
	if p.eat("UNIQUE") {
		unique = true
	}
	if p.eat("NULL_FILTERED") {
		nullFiltered = true
	}
	if err := p.expect("INDEX"); err != nil {
		return CreateIndex{}, err
	}
	iname, err := p.parseTableOrIndexOrColumnName()
	if err != nil {
		return CreateIndex{}, err
	}
	if err := p.expect("ON"); err != nil {
		return CreateIndex{}, err
	}
	tname, err := p.parseTableOrIndexOrColumnName()
	if err != nil {
		return CreateIndex{}, err
	}
	ci := CreateIndex{
		Name:  iname,
		Table: tname,

		Unique:       unique,
		NullFiltered: nullFiltered,
	}
	ci.Columns, err = p.parseKeyPartList()
	if err != nil {
		return CreateIndex{}, err
	}

	if p.eat("STORING") {
		ci.Storing, err = p.parseColumnNameList()
		if err != nil {
			return CreateIndex{}, err
		}
	}

	if p.eat(",", "INTERLEAVE", "IN") {
		ci.Interleave, err = p.parseTableOrIndexOrColumnName()
		if err != nil {
			return CreateIndex{}, err
		}
	}

	return ci, nil
}

func (p *parser) parseAlterTable() (AlterTable, error) {
	debugf("parseAlterTable: %v", p)

	/*
		alter_table:
			ALTER TABLE table_name { table_alteration | table_column_alteration }

		table_alteration:
			{ ADD COLUMN column_def | DROP COLUMN column_name |
				SET ON DELETE { CASCADE | NO ACTION } }

		table_column_alteration:
			ALTER COLUMN column_name { { scalar_type | array_type } [NOT NULL] | SET options_def }
	*/

	if err := p.expect("ALTER"); err != nil {
		return AlterTable{}, err
	}
	if err := p.expect("TABLE"); err != nil {
		return AlterTable{}, err
	}
	tname, err := p.parseTableOrIndexOrColumnName()
	if err != nil {
		return AlterTable{}, err
	}
	a := AlterTable{Name: tname}

	tok := p.next()
	if tok.err != nil {
		return AlterTable{}, tok.err
	}
	switch tok.value {
	default:
		return AlterTable{}, p.errorf("got %q, expected ADD or DROP or SET or ALTER", tok.value)
	case "ADD":
		if err := p.expect("COLUMN"); err != nil {
			return AlterTable{}, err
		}
		cd, err := p.parseColumnDef()
		if err != nil {
			return AlterTable{}, err
		}
		a.Alteration = AddColumn{Def: cd}
		return a, nil
	case "DROP":
		if err := p.expect("COLUMN"); err != nil {
			return AlterTable{}, err
		}
		name, err := p.parseTableOrIndexOrColumnName()
		if err != nil {
			return AlterTable{}, err
		}
		a.Alteration = DropColumn{Name: name}
		return a, nil
	case "SET":
		if err := p.expect("ON"); err != nil {
			return AlterTable{}, err
		}
		if err := p.expect("DELETE"); err != nil {
			return AlterTable{}, err
		}
		od, err := p.parseOnDelete()
		if err != nil {
			return AlterTable{}, err
		}
		a.Alteration = SetOnDelete{Action: od}
		return a, nil
	}
	// TODO: "ALTER"
}

func (p *parser) parseColumnDef() (ColumnDef, error) {
	debugf("parseColumnDef: %v", p)

	/*
		column_def:
			column_name {scalar_type | array_type} [NOT NULL] [options_def]
	*/

	name, err := p.parseTableOrIndexOrColumnName()
	if err != nil {
		return ColumnDef{}, err
	}

	cd := ColumnDef{Name: name}

	cd.Type, err = p.parseType()
	if err != nil {
		return ColumnDef{}, err
	}

	tok := p.next()
	if tok.err != nil || tok.value != "NOT" {
		// End of the column_def.
		p.back()
		return cd, nil
	}
	if err := p.expect("NULL"); err != nil {
		return ColumnDef{}, err
	}
	cd.NotNull = true

	return cd, nil
}

func (p *parser) parseKeyPartList() ([]KeyPart, error) {
	var list []KeyPart
	err := p.parseCommaList(func(p *parser) error {
		kp, err := p.parseKeyPart()
		if err != nil {
			return err
		}
		list = append(list, kp)
		return nil
	})
	return list, err
}

func (p *parser) parseKeyPart() (KeyPart, error) {
	debugf("parseKeyPart: %v", p)

	/*
		key_part:
			column_name [{ ASC | DESC }]
	*/

	name, err := p.parseTableOrIndexOrColumnName()
	if err != nil {
		return KeyPart{}, err
	}

	kp := KeyPart{Column: name}

	tok := p.next()
	if tok.err != nil {
		// End of the key_part.
		p.back()
		return kp, nil
	}
	switch tok.value {
	case "ASC":
	case "DESC":
		kp.Desc = true
	default:
		p.back()
	}

	return kp, nil
}

func (p *parser) parseColumnNameList() ([]string, error) {
	var list []string
	err := p.parseCommaList(func(p *parser) error {
		n, err := p.parseTableOrIndexOrColumnName()
		if err != nil {
			return err
		}
		list = append(list, n)
		return nil
	})
	return list, err
}

var baseTypes = map[string]TypeBase{
	"BOOL":      Bool,
	"INT64":     Int64,
	"FLOAT64":   Float64,
	"STRING":    String,
	"BYTES":     Bytes,
	"DATE":      Date,
	"TIMESTAMP": Timestamp,
}

func (p *parser) parseType() (Type, error) {
	debugf("parseType: %v", p)

	/*
		array_type:
			ARRAY< scalar_type >

		scalar_type:
			{ BOOL | INT64 | FLOAT64 | STRING( length ) | BYTES( length ) | DATE | TIMESTAMP }
		length:
			{ int64_value | MAX }
	*/

	var t Type

	tok := p.next()
	if tok.err != nil {
		return Type{}, tok.err
	}
	if tok.value == "ARRAY" {
		t.Array = true
		if err := p.expect("<"); err != nil {
			return Type{}, err
		}
		tok = p.next()
		if tok.err != nil {
			return Type{}, tok.err
		}
	}
	base, ok := baseTypes[tok.value]
	if !ok {
		return Type{}, p.errorf("got %q, want scalar type", tok.value)
	}
	t.Base = base

	if t.Base == String || t.Base == Bytes {
		if err := p.expect("("); err != nil {
			return Type{}, err
		}

		tok = p.next()
		if tok.err != nil {
			return Type{}, tok.err
		}
		if tok.value == "MAX" {
			t.Len = MaxLen
		} else if tok.typ == int64Token {
			t.Len = tok.int64
		} else {
			return Type{}, p.errorf("got %q, want MAX or int64", tok.value)
		}

		if err := p.expect(")"); err != nil {
			return Type{}, err
		}
	}

	if t.Array {
		if err := p.expect(">"); err != nil {
			return Type{}, err
		}
	}

	return t, nil
}

func (p *parser) parseQuery() (Query, error) {
	debugf("parseQuery: %v", p)

	/*
		query_statement:
			[ table_hint_expr ][ join_hint_expr ]
			query_expr

		query_expr:
			{ select | ( query_expr ) | query_expr set_op query_expr }
			[ ORDER BY expression [{ ASC | DESC }] [, ...] ]
			[ LIMIT count [ OFFSET skip_rows ] ]
	*/

	// TODO: hints, sub-selects, etc.

	// TODO: use a case-insensitive select.
	if err := p.expect("SELECT"); err != nil {
		return Query{}, err
	}
	p.back()
	sel, err := p.parseSelect()
	if err != nil {
		return Query{}, err
	}
	q := Query{Select: sel}

	if p.eat("ORDER", "BY") {
		for {
			o, err := p.parseOrder()
			if err != nil {
				return Query{}, err
			}
			q.Order = append(q.Order, o)

			if !p.eat(",") {
				break
			}
		}
	}

	if p.eat("LIMIT") {
		lim, err := p.parseLimitCount()
		if err != nil {
			return Query{}, err
		}
		q.Limit = lim
	}

	return q, nil
}

func (p *parser) parseSelect() (Select, error) {
	debugf("parseSelect: %v", p)

	/*
		select:
			SELECT  [{ ALL | DISTINCT }]
				{ [ expression. ]* | expression [ [ AS ] alias ] } [, ...]
			[ FROM from_item [ tablesample_type ] [, ...] ]
			[ WHERE bool_expression ]
			[ GROUP BY expression [, ...] ]
			[ HAVING bool_expression ]
	*/
	if err := p.expect("SELECT"); err != nil {
		return Select{}, err
	}

	var sel Select

	// TODO: ALL|DISTINCT

	// Read expressions for the SELECT list.
	for {
		expr, err := p.parseExpr()
		if err != nil {
			return Select{}, err
		}
		sel.List = append(sel.List, expr)

		if p.eat(",") {
			continue
		}
		break
	}

	if p.eat("FROM") {
		for {
			from, err := p.parseSelectFrom()
			if err != nil {
				return Select{}, err
			}
			if p.sniff("TABLESAMPLE") {
				ts, err := p.parseTableSample()
				if err != nil {
					return Select{}, err
				}
				from.TableSample = &ts
			}
			sel.From = append(sel.From, from)

			if p.eat(",") {
				continue
			}
			break
		}
	}

	if p.eat("WHERE") {
		where, err := p.parseBoolExpr()
		if err != nil {
			return Select{}, err
		}
		sel.Where = where
	}

	// TODO: GROUP BY, HAVING

	return sel, nil
}

func (p *parser) parseSelectFrom() (SelectFrom, error) {
	// TODO: support more than a single table name.
	tname, err := p.parseTableOrIndexOrColumnName()
	return SelectFrom{Table: tname}, err
}

func (p *parser) parseTableSample() (TableSample, error) {
	var ts TableSample

	if err := p.expect("TABLESAMPLE"); err != nil {
		return ts, err
	}

	tok := p.next()
	switch {
	case tok.err != nil:
		return ts, tok.err
	case tok.value == "BERNOULLI":
		ts.Method = Bernoulli
	case tok.value == "RESERVOIR":
		ts.Method = Reservoir
	default:
		return ts, p.errorf("got %q, want BERNOULLI or RESERVOIR", tok.value)
	}

	if err := p.expect("("); err != nil {
		return ts, err
	}

	// The docs say "numeric_value_expression" here,
	// but that doesn't appear to be defined anywhere.
	size, err := p.parseExpr()
	if err != nil {
		return ts, err
	}
	ts.Size = size

	tok = p.next()
	switch {
	case tok.err != nil:
		return ts, tok.err
	case tok.value == "PERCENT":
		ts.SizeType = PercentTableSample
	case tok.value == "ROWS":
		ts.SizeType = RowsTableSample
	default:
		return ts, p.errorf("got %q, want PERCENT or ROWS", tok.value)
	}

	if err := p.expect(")"); err != nil {
		return ts, err
	}

	return ts, nil
}

func (p *parser) parseOrder() (Order, error) {
	/*
		expression [{ ASC | DESC }]
	*/

	expr, err := p.parseExpr()
	if err != nil {
		return Order{}, err
	}
	o := Order{Expr: expr}

	tok := p.next()
	switch {
	case tok.err == nil && tok.value == "ASC":
	case tok.err == nil && tok.value == "DESC":
		o.Desc = true
	default:
		p.back()
	}

	return o, nil
}

func (p *parser) parseLimitCount() (Limit, error) {
	// "only literal or parameter values"
	// https://cloud.google.com/spanner/docs/query-syntax#limit-clause-and-offset-clause

	tok := p.next()
	if tok.err != nil {
		return nil, tok.err
	}
	if tok.typ == int64Token {
		return IntegerLiteral(tok.int64), nil
	}
	// TODO: check character sets.
	if strings.HasPrefix(tok.value, "@") {
		return Param(tok.value[1:]), nil
	}
	return nil, p.errorf("got %q, want literal or parameter", tok.value)
}

func (p *parser) parseExprList() ([]Expr, error) {
	var list []Expr
	err := p.parseCommaList(func(p *parser) error {
		e, err := p.parseExpr()
		if err != nil {
			return err
		}
		list = append(list, e)
		return nil
	})
	return list, err
}

/*
Expressions

Cloud Spanner expressions are not formally specified.
The set of operators and their precedence is listed in
https://cloud.google.com/spanner/docs/functions-and-operators#operators.

parseExpr works as a classical recursive descent parser, splitting
precedence levels into separate methods, where the call stack is in
ascending order of precedence:
	parseExpr
	orParser
	andParser
	parseIsOp
	parseComparisonOp
	parseArithOp
	parseLit

TODO: there are more levels to break out, esp. in parseArithOp
*/

func (p *parser) parseExpr() (Expr, error) {
	debugf("parseExpr: %v", p)

	return orParser.parse(p)
}

// binOpParser is a generic meta-parser for binary operations.
// It assumes the operation is left associative.
type binOpParser struct {
	LHS, RHS func(*parser) (Expr, error)
	Op       string
	ArgCheck func(Expr) error
	Combiner func(lhs, rhs Expr) Expr
}

func (bin binOpParser) parse(p *parser) (Expr, error) {
	expr, err := bin.LHS(p)
	if err != nil {
		return nil, err
	}

	for {
		if !p.eat(bin.Op) {
			break
		}
		rhs, err := bin.RHS(p)
		if err != nil {
			return nil, err
		}
		if bin.ArgCheck != nil {
			if err := bin.ArgCheck(expr); err != nil {
				return nil, p.errorf("%v", err)
			}
			if err := bin.ArgCheck(rhs); err != nil {
				return nil, p.errorf("%v", err)
			}
		}
		expr = bin.Combiner(expr, rhs)
	}
	return expr, nil
}

// Break initialisation loop.
func init() { orParser = orParserShim }

var (
	boolExprCheck = func(expr Expr) error {
		if _, ok := expr.(BoolExpr); !ok {
			return fmt.Errorf("got %T, want a boolean expression", expr)
		}
		return nil
	}

	orParser binOpParser

	orParserShim = binOpParser{
		LHS:      andParser.parse,
		RHS:      andParser.parse,
		Op:       "OR",
		ArgCheck: boolExprCheck,
		Combiner: func(lhs, rhs Expr) Expr {
			return LogicalOp{LHS: lhs.(BoolExpr), Op: Or, RHS: rhs.(BoolExpr)}
		},
	}
	andParser = binOpParser{
		LHS:      (*parser).parseLogicalNot,
		RHS:      (*parser).parseLogicalNot,
		Op:       "AND",
		ArgCheck: boolExprCheck,
		Combiner: func(lhs, rhs Expr) Expr {
			return LogicalOp{LHS: lhs.(BoolExpr), Op: And, RHS: rhs.(BoolExpr)}
		},
	}
)

func (p *parser) parseLogicalNot() (Expr, error) {
	if !p.eat("NOT") {
		return p.parseIsOp()
	}
	be, err := p.parseBoolExpr()
	if err != nil {
		return nil, err
	}
	return LogicalOp{Op: Not, RHS: be}, nil
}

func (p *parser) parseIsOp() (Expr, error) {
	debugf("parseIsOp: %v", p)

	expr, err := p.parseComparisonOp()
	if err != nil {
		return nil, err
	}

	tok := p.next()
	if tok.err != nil || tok.value != "IS" {
		p.back()
		return expr, nil
	}

	isOp := IsOp{LHS: expr}
	if p.eat("NOT") {
		isOp.Neg = true
	}

	tok = p.next()
	if tok.err != nil {
		return nil, tok.err
	}
	switch tok.value {
	case "NULL":
		isOp.RHS = Null
	case "TRUE":
		isOp.RHS = True
	case "FALSE":
		isOp.RHS = False
	default:
		return nil, p.errorf("got %q, want NULL or TRUE or FALSE", tok.value)
	}

	return isOp, nil
}

var symbolicOperators = map[string]ComparisonOperator{
	"<":  Lt,
	"<=": Le,
	">":  Gt,
	">=": Ge,
	"=":  Eq,
	"!=": Ne,
	"<>": Ne,
}

func (p *parser) parseComparisonOp() (Expr, error) {
	debugf("parseComparisonOp: %v", p)

	expr, err := p.parseArithOp()
	if err != nil {
		return nil, err
	}

	for {
		tok := p.next()
		if tok.err != nil {
			p.back()
			break
		}
		var op ComparisonOperator
		var ok, rhs2 bool
		if tok.value == "NOT" {
			tok := p.next()
			switch {
			case tok.err != nil:
				// TODO: Does this need to push back two?
				return nil, err
			case tok.value == "LIKE":
				op, ok = NotLike, true
			case tok.value == "BETWEEN":
				op, ok, rhs2 = NotBetween, true, true
			default:
				// TODO: Does this need to push back two?
				return nil, p.errorf("got %q, want LIKE or BETWEEN", tok.value)
			}
		} else if tok.value == "LIKE" {
			op, ok = Like, true
		} else if tok.value == "BETWEEN" {
			op, ok, rhs2 = Between, true, true
		} else {
			op, ok = symbolicOperators[tok.value]
		}
		if !ok {
			p.back()
			break
		}

		rhs, err := p.parseArithOp()
		if err != nil {
			return nil, err
		}
		co := ComparisonOp{LHS: expr, Op: op, RHS: rhs}

		if rhs2 {
			if err := p.expect("AND"); err != nil {
				return nil, err
			}
			rhs2, err := p.parseArithOp()
			if err != nil {
				return nil, err
			}
			co.RHS2 = rhs2
		}

		expr = co
	}
	return expr, nil
}

func (p *parser) parseArithOp() (Expr, error) {
	// TODO: actually parse arithmetic operations.

	if p.eat("(") {
		e, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if err := p.expect(")"); err != nil {
			return nil, err
		}
		return Paren{Expr: e}, nil
	}

	lit, err := p.parseLit()
	if err != nil {
		return nil, err
	}

	// If the literal was an identifier, and there's an open paren next,
	// this is a function invocation.
	if id, ok := lit.(ID); ok && p.sniff("(") {
		list, err := p.parseExprList()
		if err != nil {
			return nil, err
		}
		return Func{
			Name: string(id),
			Args: list,
		}, nil
	}

	return lit, nil
}

func (p *parser) parseLit() (Expr, error) {
	tok := p.next()
	if tok.err != nil {
		return nil, tok.err
	}

	switch tok.typ {
	case int64Token:
		return IntegerLiteral(tok.int64), nil
	case float64Token:
		return FloatLiteral(tok.float64), nil
	case stringToken:
		return StringLiteral(tok.string), nil
	case bytesToken:
		return BytesLiteral(tok.string), nil
	}

	// Handle some reserved keywords and special tokens that become specific values.
	// TODO: Handle the other 92 keywords.
	switch tok.value {
	case "TRUE":
		return True, nil
	case "FALSE":
		return False, nil
	case "NULL":
		return Null, nil
	case "*":
		return Star, nil
	}

	// TODO: more types of literals (array, struct, date, timestamp).

	// Try a parameter.
	// TODO: check character sets.
	if strings.HasPrefix(tok.value, "@") {
		return Param(tok.value[1:]), nil
	}
	return ID(tok.value), nil
}

func (p *parser) parseBoolExpr() (BoolExpr, error) {
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	be, ok := expr.(BoolExpr)
	if !ok {
		return nil, p.errorf("got non-bool expression %T", expr)
	}
	return be, nil
}

func (p *parser) parseTableOrIndexOrColumnName() (string, error) {
	/*
		table_name and column_name and index_name:
				{a—z|A—Z}[{a—z|A—Z|0—9|_}+]
	*/

	tok := p.next()
	if tok.err != nil {
		return "", tok.err
	}
	// TODO: enforce restrictions
	return tok.value, nil
}

func (p *parser) parseOnDelete() (OnDelete, error) {
	/*
		CASCADE
		NO ACTION
	*/

	tok := p.next()
	if tok.err != nil {
		return 0, tok.err
	}
	if tok.value == "CASCADE" {
		return CascadeOnDelete, nil
	}
	if tok.value != "NO" {
		return 0, p.errorf("got %q, want NO or CASCADE", tok.value)
	}
	if err := p.expect("ACTION"); err != nil {
		return 0, err
	}
	return NoActionOnDelete, nil
}

// parseCommaList parses a parenthesized comma-separated list,
// delegating to f for the individual element parsing.
func (p *parser) parseCommaList(f func(*parser) error) error {
	if err := p.expect("("); err != nil {
		return err
	}
	for {
		if p.eat(")") {
			return nil
		}

		err := f(p)
		if err != nil {
			return err
		}

		// ")" or "," should be next.
		tok := p.next()
		if tok.err != nil {
			return err
		}
		if tok.value == ")" {
			return nil
		} else if tok.value == "," {
			continue
		} else {
			return p.errorf(`got %q, want ")" or ","`, tok.value)
		}
	}
}
