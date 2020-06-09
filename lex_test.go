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
