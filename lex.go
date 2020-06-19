package toml

import (
	"fmt"
	"strconv"
	"strings"
)

// https://github.com/toml-lang/toml

type itemType int

const (
	itemError itemType = iota
	itemEOF
	itemKey
	itemEqual
	itemStringValue
	itemIntegerValue
	itemFloatValue
	itemBooleanValue
	itemTimeValue
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
	items     chan item // channel of scanned items
	line      int       // 1+number of newlines seen
	startLine int       // start line of this item

	buf strings.Builder
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
		return eof
	}
	r := l.input[l.pos]
	l.pos++
	if r == '\n' {
		l.line++
	}
	return rune(r)
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos--
	// Correct newline count.
	if l.input[l.pos] == '\n' {
		l.line--
	}
}

// nextItem returns the next item from the input.
// Called by the parser, not in the lexing goroutine.
func (l *lexer) nextItem() item {
	return <-l.items
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	if l.isEOF() {
		return eof
	}
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) isNextString(s string) bool {
	ln := Pos(len(s))
	wantPos := l.pos + ln
	if int(wantPos) > len(l.input) {
		return false
	}
	return l.input[l.pos:wantPos] == s
}

func (l *lexer) writeRune(r rune) {
	l.buf.WriteRune(r)
	l.ignore()
}

func (l *lexer) emitBuffer(t itemType) {
	l.items <- item{t, l.start, l.buf.String(), l.startLine}
	l.start = l.pos
	l.startLine = l.line
	l.buf.Reset()
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
	// l.line += strings.Count(l.input[l.start:l.pos], "\n")
	l.start = l.pos
	l.startLine = l.line
}

func (l *lexer) skip() {
	l.next()
	l.ignore()
}

// skipN skips next n characters. use this when want to skip string.
func (l *lexer) skipN(n int) {
	l.pos += Pos(n)
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
			break
		}
		switch next {
		case '#':
			return lexComment(l)
		case '=':
			return lexEqual
		case '\r', '\n':
			// https://github.com/toml-lang/toml#spec
			// Newline means LF (0x0A) or CRLF (0x0D 0x0A).
			l.next()
		}

		if isSpace(next) {
			l.skip()
			continue
		}

		if isKey(next) {
			return lexKey(l)
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

func lexEqual(l *lexer) stateFn {
	l.next()
	l.emit(itemEqual)
	return lexValue
}

func lexValue(l *lexer) stateFn {
	for {
		c := l.next()
		if c == eof {
			return l.errorf("invalid unspecified value")
		}

		if isSpace(c) {
			l.ignore()
			continue
		}
		switch c {
		case '"':
			// check multiline first
			if l.isNextString(`""`) {
				l.skipN(3)
				if err := scanMultiLineBasicStrings(l); err != nil {
					return l.errorf(err.Error())
				}
				l.emitBuffer(itemStringValue)
				break
			}
			if err := scanBasicString(l); err != nil {
				return l.errorf(err.Error())
			}
			l.emitBuffer(itemStringValue)
			l.skip()
		case '\'':
			l.ignore()
			// check multiline first
			if err := scanLiteralString(l); err != nil {
				return l.errorf(err.Error())
			}
			l.emit(itemStringValue)
			l.skip() // skip "'" at last
		}
		return lexText
	}
}

// https://github.com/toml-lang/toml#keys
func isKey(c rune) bool {
	isDigit := '0' <= c && c <= '9'
	isLetters := 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
	isDash := c == '-'
	isUnderscore := c == '_'
	isQuoted := c == '"' || c == '\''
	return isDigit || isLetters || isDash || isUnderscore || isQuoted
}

func lexKey(l *lexer) stateFn {
	for {
		c := l.next()
		if !isKey(c) && c != '\r' && c != '\n' {
			l.backup()
			l.emit(itemKey)
			return lexText
		}
		switch c {
		case '"', '\'':
			return lexQuotedKey(l, c)
		}
	}
}

func lexQuotedKey(l *lexer, delim rune) stateFn {
	l.ignore() // ignore delim

	// if "Literal strings"
	if delim == '\'' {
		if err := scanLiteralString(l); err != nil {
			return l.errorf(err.Error())
		}
		l.emit(itemKey)
		l.skip() // skip "'" at last
		return lexText
	}
	// if "Basic strings"
	if delim == '"' {
		if err := scanBasicString(l); err != nil {
			return l.errorf(err.Error())
		}
		l.emitBuffer(itemKey)
		l.skip()
		return lexText
	}
	return l.errorf("unsupported delimiter: `%c`", delim)
}

func scanLiteralString(l *lexer) error {
	x := strings.Index(l.input[l.pos:], "'")
	if x < 0 {
		return fmt.Errorf("failed to lex literal strings key: `%s`", l.input[l.pos:])
	}
	l.pos += Pos(x)
	return nil
}

func scanBasicString(l *lexer) error {
	for {
		c := l.next()

		switch c {
		case '"':
			return nil
		case '\\':
			l.ignore()
			esc := l.next()
			switch esc {
			case 'b':
				l.writeRune('\b')
			case 't':
				l.writeRune('\t')
			case 'n':
				l.writeRune('\n')
			case 'f':
				l.writeRune('\f')
			case 'r':
				l.writeRune('\r')
			case '"':
				l.writeRune('"')
			case '\\':
				l.writeRune('\\')
			case 'u':
				var code string
				for i := 0; i < 4; i++ {
					cc := l.next()
					if !isHex(cc) {
						return fmt.Errorf("unexpected character: `%#U`", cc)
					}
					code += string(cc)
				}
				i, err := strconv.ParseInt(code, 16, 32)
				if err != nil {
					return fmt.Errorf("invalid unicode escape: `\\u%s`", code)
				}
				if !isUnicodeScalar(i) {
					return fmt.Errorf("invalid unicode scalar: `\\u%s`", code)
				}
				l.writeRune(rune(i))
			case 'U':
				var code string
				for i := 0; i < 8; i++ {
					cc := l.next()
					if !isHex(cc) {
						return fmt.Errorf("unexpected character: `%#U`", cc)
					}
					code += string(cc)
				}
				i, err := strconv.ParseInt(code, 16, 64)
				if err != nil {
					return fmt.Errorf("invalid unicode escape: `\\u%s`", code)
				}
				if !isUnicodeScalar(i) {
					return fmt.Errorf("invalid unicode scalar: `\\u%s`", code)
				}
				l.writeRune(rune(i))
			default:
				return fmt.Errorf("invalid escape: `%#U`", esc)
			}
		default:
			l.writeRune(c)
		}
	}
}

func scanMultiLineBasicStrings(l *lexer) error {
	onHead := true
	backSlash := false

	for {
		if l.isNextString(`"""`) {
			l.skipN(3)
			return nil
		}

		c := l.next()
		// A newline immediately following the opening delimiter will be trimmed.
		for onHead && (c == '\n' || c == '\r') {
			c = l.next()
		}
		onHead = false

		if c == '\\' {
			backSlash = true
			continue
		}
		if backSlash {
			// When the last non-whitespace character on a line is an unescaped \,
			// it will be trimmed along with all whitespace (including newlines)
			// up to the next non-whitespace character or closing delimiter.
			if c == ' ' || c == '\n' || c == '\r' {
				continue
			}
		}

		backSlash = false
		l.writeRune(c)
	}
}

func isHex(r rune) bool {
	return '0' <= r && r <= '9' || 'a' <= r && r <= 'f' || 'A' <= r && r <= 'F'
}

// The escape codes must be valid Unicode scalar values.
// http://unicode.org/glossary/#unicode_scalar_value
func isUnicodeScalar(i int64) bool {
	return 0 <= i && i <= 0xD7FF || 0xE000 <= i && i <= 0x10FFFF
}
