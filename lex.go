package toml

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// https://github.com/toml-lang/toml

type itemType int

const (
	itemError itemType = iota
	itemEOF

	itemWhiteSpace
	// Newline means LF (0x0A) or CRLF (0x0D 0x0A).
	itemNewLine
)

const (
	eof = -1
)

// item represents a token or text string returned from the scanner.
type item struct {
	typ  itemType // The type of this item.
	pos  Pos      // The starting position, in bytes, of this item in the input string.
	val  string   // The value of this item.
	line int      // The line number at the start of this item.
}

// stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	name      string    // the name of the input; used only for error reports
	input     string    // the string being scanned
	pos       Pos       // current position in the input
	start     Pos       // start position of this item
	width     Pos       // width of last rune read from input
	items     chan item // channel of scanned items
	line      int       // 1+number of newlines seen
	startLine int       // start line of this item
}

// lex creates a new scanner for the input string.
func lex(name, input string) *lexer {
	l := &lexer{
		name:      name,
		input:     input,
		items:     make(chan item),
		line:      1,
		startLine: 1,
	}
	go l.run()
	return l
}

// run runs the state machine for the lexer.
func (l *lexer) run() {
	for state := lexText; state != nil; {
		state = state(l)
	}
	close(l.items)
}

func (l *lexer) next() rune {
	if l.isEOF() {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = Pos(w)
	l.pos += l.width
	// https://github.com/toml-lang/toml#spec
	// Newline means LF (0x0A) or CRLF (0x0D 0x0A).
	switch r {
	case '\r':
		return l.next()
	case '\n':
		l.line++
	}
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
	// Correct newline count.
	if l.width == 1 && l.input[l.pos] == '\n' {
		l.line--
		if int(l.pos) > 0 && l.input[l.pos-1] == '\r' {
			l.pos-- // l.width == 1
		}
	}
}

// nextItem returns the next item from the input.
// Called by the parser, not in the lexing goroutine.
func (l *lexer) nextItem() item {
	return <-l.items
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.start, l.input[l.start:l.pos], l.startLine}
	l.start = l.pos
	l.startLine = l.line
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, l.start, fmt.Sprintf(format, args...), l.startLine}
	return nil
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.line += strings.Count(l.input[l.start:l.pos], "\n")
	l.start = l.pos
	l.startLine = l.line
}

func (l *lexer) skip() {
	l.next()
	l.ignore()
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	// https://github.com/toml-lang/toml#spec
	// Whitespace means tab (0x09) or space (0x20)
	return r == 0x09 || r == 0x20
}

func (l *lexer) isEOF() bool {
	return int(l.pos) >= len(l.input)
}

func lexText(l *lexer) stateFn {
	for {
		next := l.peek()
		if next == eof {
			l.next()
			break
		}
		switch next {
		case '#':
			return lexComment(l)
		}

		if isSpace(next) {
			l.skip()
			continue
		}
	}
	l.emit(itemEOF)
	return nil
}

func lexComment(l *lexer) stateFn {
	for {
		c := l.next()
		if 0x00 <= c && c <= 0x08 || 0x0a <= c && c <= 0x1f || c == 0x7f {
			return l.errorf("unexpected control character in comment: `%c`", c)
		}
		if c == '\n' || c == eof {
			break
		}
	}
	l.ignore()
	return lexText
}
