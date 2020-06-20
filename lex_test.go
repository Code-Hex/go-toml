package toml

import (
	"testing"
)

type lexTest struct {
	name  string
	input string
	items []item
}

var (
	tEOF = mkItem(itemEOF, "")
)

func mkItem(typ itemType, text string) item {
	return item{
		typ: typ,
		val: text,
	}
}

func TestLex(t *testing.T) {
	for _, test := range []lexTest{
		{"empty", "", []item{tEOF}},
		{"comment", "# hello, world", []item{tEOF}},
		{"spaces", "    \t", []item{tEOF}},
		{"literal string quoted key", "'key'", []item{
			mkItem(itemKey, "key"),
			tEOF,
		}},
		{"basic string", `"key"`, []item{
			mkItem(itemKey, "key"),
			tEOF,
		}},
		{"basic string with \\b\\t\\n\\f\\r\\\"\\\\", `"hello\b\t\n\f\r\"\\world"`, []item{
			mkItem(itemKey, "hello\b\t\n\f\r\"\\world"),
			tEOF,
		}},
		{"basic string with unicode 4-digits", `"\u65E5\u672C\u8A9E"`, []item{
			mkItem(itemKey, "日本語"),
			tEOF,
		}},
		{"basic string with unicode 8-digits", `"\U000065e5\U0000672c\U00008a9e"`, []item{
			mkItem(itemKey, "日本語"),
			tEOF,
		}},
		{"basic string complex", `"I'm a string. \"You can quote me\". Name\tJos\u00E9\nLocation\tSF."`, []item{
			mkItem(itemKey, "I'm a string. \"You can quote me\". Name\tJos\u00E9\nLocation\tSF."),
			tEOF,
		}},
		{"invalid key", "key =", []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemError, "invalid unspecified value"),
		}},
		{"key = basic string", `key = "\u65E5\u672C\u8A9E"`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemStringValue, "日本語"),
			tEOF,
		}},
		{"key = string literal", `key = 'hello'`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemStringValue, "hello"),
			tEOF,
		}},
		{"key = multi-line basic string strings", "key = \"\"\"\nhello\nworld\"\"\"", []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemStringValue, "hello\nworld"),
			tEOF,
		}},
		{"key = multi-line basic strings on windows", "key = \"\"\"\nhello\r\nworld\"\"\"", []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemStringValue, "hello\r\nworld"),
			tEOF,
		}},
		{"key = multi-line basic strings backslash", "key = \"\"\"\\\nhello \\\n\n world\\\n\"\"\"", []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemStringValue, "hello world"),
			tEOF,
		}},
		{
			"key = multi-line basic strings flatten",
			`key = """Here"""`,
			[]item{
				mkItem(itemKey, "key"),
				mkItem(itemEqual, "="),
				mkItem(itemStringValue, `Here`),
				tEOF,
			},
		},
		{
			"key = multi-line basic strings escaped three quotation marks",
			`key = """Here are three quotation marks: ""\"."""`,
			[]item{
				mkItem(itemKey, "key"),
				mkItem(itemEqual, "="),
				mkItem(itemStringValue, `Here are three quotation marks: """.`),
				tEOF,
			},
		},
		{
			"key = multi-line basic strings escaped fifteen quotation marks",
			`key = """Here are fifteen quotation marks: ""\"""\"""\"""\"""\"."""`,
			[]item{
				mkItem(itemKey, "key"),
				mkItem(itemEqual, "="),
				mkItem(itemStringValue, `Here are fifteen quotation marks: """"""""""""""".`),
				tEOF,
			},
		},
		{
			"invalid multi-line basic strings 1",
			`key = """Here are three quotation marks: """."""`,
			[]item{
				mkItem(itemKey, "key"),
				mkItem(itemEqual, "="),
				mkItem(itemStringValue, "Here are three quotation marks: "),
				mkItem(itemError, "invalid character: `U+002E '.'`"),
			},
		},
		{
			"key = multi-line basic strings prefix '\"'",
			`key = """"This," she said, "is just a pointless statement.""""`,
			[]item{
				mkItem(itemKey, "key"),
				mkItem(itemEqual, "="),
				mkItem(itemStringValue, `"This," she said, "is just a pointless statement."`),
				tEOF,
			},
		},
	} {
		items := collect(&test)
		if !equal(items, test.items, false) {
			t.Errorf("%s: got\n\t%+v\nexpected\n\t%+v", test.name, items, test.items)
		}
	}
}

// collect gathers the emitted items into a slice.
func collect(t *lexTest) (items []item) {
	l := lex(t.name, t.input)
	for {
		item := l.nextItem()
		items = append(items, item)
		if item.typ == itemEOF || item.typ == itemError {
			break
		}
	}
	return
}

func equal(i1, i2 []item, checkPos bool) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].typ != i2[k].typ {
			return false
		}
		if i1[k].val != i2[k].val {
			return false
		}
		if checkPos && i1[k].pos != i2[k].pos {
			return false
		}
		if checkPos && i1[k].line != i2[k].line {
			return false
		}
	}
	return true
}

func Test_lexer_isNextString(t *testing.T) {
	tests := []struct {
		name  string
		lexer *lexer
		s     string
		want  bool
	}{
		{
			name: "pos 0, want true",
			lexer: &lexer{
				input: "hello",
				pos:   0,
			},
			s:    "hello",
			want: true,
		},
		{
			name: "pos 0, diff str, want false",
			lexer: &lexer{
				input: "hello",
				pos:   1,
			},
			s:    "world",
			want: false,
		},
		{
			name: "pos 1, range over, want false",
			lexer: &lexer{
				input: "hello",
				pos:   1,
			},
			s:    "hello",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := tt.lexer
			if got := l.isNextString(tt.s); got != tt.want {
				t.Errorf("lexer.isNextString() = %v, want %v", got, tt.want)
			}
		})
	}
}
