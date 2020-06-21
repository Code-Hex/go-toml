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
		{"key = basic string 2", `key = "Here are fifteen apostrophes: '''''''''''''''"`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemStringValue, "Here are fifteen apostrophes: '''''''''''''''"),
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
		{
			"key = multi-line literal strings",
			"key = '''\n\nhello\nworld\n'''",
			[]item{
				mkItem(itemKey, "key"),
				mkItem(itemEqual, "="),
				mkItem(itemStringValue, "hello\nworld\n"),
				tEOF,
			},
		},
		{
			"key = multi-line literal strings flatten",
			`key = '''I [dw]on't need \d{2} apples'''`,
			[]item{
				mkItem(itemKey, "key"),
				mkItem(itemEqual, "="),
				mkItem(itemStringValue, `I [dw]on't need \d{2} apples`),
				tEOF,
			},
		},
		{
			"key = multi-line literal strings include '\"'",
			`key = '''Here are fifteen quotation marks: """""""""""""""'''`,
			[]item{
				mkItem(itemKey, "key"),
				mkItem(itemEqual, "="),
				mkItem(itemStringValue, `Here are fifteen quotation marks: """""""""""""""`),
				tEOF,
			},
		},
		{
			"key = multi-line literal strings sigle quoted",
			`key = ''''That,' she said, 'is still pointless.''''`,
			[]item{
				mkItem(itemKey, "key"),
				mkItem(itemEqual, "="),
				mkItem(itemStringValue, `'That,' she said, 'is still pointless.'`),
				tEOF,
			},
		},
		{"key = integer", `key = 01234`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemIntegerValue, "01234"),
			tEOF,
		}},
		{"key = integer 0", `key = 0`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemIntegerValue, "0"),
			tEOF,
		}},
		{"key = integer 1", `key = 1`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemIntegerValue, "1"),
			tEOF,
		}},
		{"key = integer with '_'", `key = 1_234_56789`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemIntegerValue, "1_234_56789"),
			tEOF,
		}},
		{"invalid key = integer decimal with '_'", `key = 1_`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemError, "expected integer after '_'"),
		}},
		{"key = integer prefix `+`", `key = +10`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemIntegerValue, "+10"),
			tEOF,
		}},
		{"key = integer prefix `-`", `key = -10`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemIntegerValue, "-10"),
			tEOF,
		}},
		{"key = integer hex", `key = 0xdead_BEEF`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemIntegerValue, "0xdead_BEEF"),
			tEOF,
		}},
		{"invalid key = integer hex with '_'", `key = 0x1_22_`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemError, "expected integer after '_'"),
		}},
		{"key = integer oct", `oct1 = 0o01234567`, []item{
			mkItem(itemKey, "oct1"),
			mkItem(itemEqual, "="),
			mkItem(itemIntegerValue, "0o01234567"),
			tEOF,
		}},
		{"invalid key = integer oct with '_'", `key = 0o0123_`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemError, "expected integer after '_'"),
		}},
		{"key = integer oct", `oct1 = -0o01234567`, []item{
			mkItem(itemKey, "oct1"),
			mkItem(itemEqual, "="),
			mkItem(itemIntegerValue, "-0o01234567"),
			tEOF,
		}},
		{"key = integer bin", `bin1 = 0b11010110`, []item{
			mkItem(itemKey, "bin1"),
			mkItem(itemEqual, "="),
			mkItem(itemIntegerValue, "0b11010110"),
			tEOF,
		}},
		{"invalid key = integer bin with '_'", `key = 0b01_01_`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemError, "expected integer after '_'"),
		}},
		{"key = fractional +1.0", `key = +1.0`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "+1.0"),
			tEOF,
		}},
		{"key = fractional π", `key = 3.1415`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "3.1415"),
			tEOF,
		}},
		{"key = fractional -0.001", `key = -0.001`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "-0.001"),
			tEOF,
		}},
		{"key = exponent 5e+22", `key = 5e+22`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "5e+22"),
			tEOF,
		}},
		{"key = exponent 1e06", `key = 1e06`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "1e06"),
			tEOF,
		}},
		{"key = exponent -2E-2", `key = -2E-2`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "-2E-2"),
			tEOF,
		}},
		{"key = float 6.626e-34", `key = 6.626e-34`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "6.626e-34"),
			tEOF,
		}},
		{"key = float 224_617.445_991_228", `key = 224_617.445_991_228`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "224_617.445_991_228"),
			tEOF,
		}},
		{"key = float inf", `key = inf`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "inf"),
			tEOF,
		}},
		{"key = float +inf", `key = +inf`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "+inf"),
			tEOF,
		}},
		{"key = float -inf", `key = -inf`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "-inf"),
			tEOF,
		}},
		{"key = float nan", `key = nan`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "nan"),
			tEOF,
		}},
		{"key = float +nan", `key = +nan`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "+nan"),
			tEOF,
		}},
		{"key = float -nan", `key = -nan`, []item{
			mkItem(itemKey, "key"),
			mkItem(itemEqual, "="),
			mkItem(itemFloatValue, "-nan"),
			tEOF,
		}},
		{"bool1 = true", `bool1 = true`, []item{
			mkItem(itemKey, "bool1"),
			mkItem(itemEqual, "="),
			mkItem(itemBooleanValue, "true"),
			tEOF,
		}},
		{"bool2 = false", `bool2 = false`, []item{
			mkItem(itemKey, "bool2"),
			mkItem(itemEqual, "="),
			mkItem(itemBooleanValue, "false"),
			tEOF,
		}},
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
