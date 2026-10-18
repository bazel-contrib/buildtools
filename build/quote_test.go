/*
Copyright 2016 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package build

import (
	"strings"
	"testing"
)

var quoteTests = []struct {
	q   string // quoted
	s   string // unquoted (actual string)
	std bool   // q is standard form for s
}{
	{`""`, "", true},
	{`''`, "", false},
	{`"hello"`, `hello`, true},
	{`'hello'`, `hello`, false},
	{`"quote\"here"`, `quote"here`, true},
	{`'quote\"here'`, `quote"here`, false},
	{`'quote"here'`, `quote"here`, false},
	{`"quote'here"`, `quote'here`, true},
	{`"quote\'here"`, `quote'here`, false},
	{`'quote\'here'`, `quote'here`, false},
	{`"""hello " ' world "" asdf ''' foo"""`, `hello " ' world "" asdf ''' foo`, true},
	{`"foo\\(bar"`, `foo\(bar`, true},
	{`"""hello
world"""`, "hello\nworld", true},

	{`"\\a\\b\\f\n\r\t\\v\000\377"`, "\\a\\b\\f\n\r\t\\v\000\xFF", true},
	{`"\\a\\b\\f\n\r\t\\v\x00\xff"`, "\\a\\b\\f\n\r\t\\v\000\xFF", false},
	{`"\\a\\b\\f\n\r\t\\v\000\xFF"`, "\\a\\b\\f\n\r\t\\v\000\xFF", false},
	{`"\\a\\b\\f\n\r\t\\v\000\377\"'\\\003\200"`, "\\a\\b\\f\n\r\t\\v\x00\xFF\"'\\\x03\x80", true},
	{`"\\a\\b\\f\n\r\t\\v\x00\xff\"'\\\x03\x80"`, "\\a\\b\\f\n\r\t\\v\x00\xFF\"'\\\x03\x80", false},
	{`"\\a\\b\\f\n\r\t\\v\000\xFF\"'\\\x03\x80"`, "\\a\\b\\f\n\r\t\\v\x00\xFF\"'\\\x03\x80", false},
	{`"\\a\\b\\f\n\r\t\\v\000\xFF\"\'\\\x03\x80"`, "\\a\\b\\f\n\r\t\\v\x00\xFF\"'\\\x03\x80", false},
	{
		`"cat $(SRCS) | grep '\s*ip_block:' | sed -e 's/\s*ip_block: \"\([^ ]*\)\"/    \x27\\1\x27,/g' >> $@; "`,
		"cat $(SRCS) | grep '\\s*ip_block:' | sed -e 's/\\s*ip_block: \"\\([^ ]*\\)\"/    '\\1',/g' >> $@; ",
		false,
	},
	{
		`"cat $(SRCS) | grep '\\s*ip_block:' | sed -e 's/\\s*ip_block: \"\\([^ ]*\\)\"/    \x27\\1\x27,/g' >> $@; "`,
		"cat $(SRCS) | grep '\\s*ip_block:' | sed -e 's/\\s*ip_block: \"\\([^ ]*\\)\"/    '\\1',/g' >> $@; ",
		false,
	},
	{
		`"cat $(SRCS) | grep '\\s*ip_block:' | sed -e 's/\\s*ip_block: \"\\([^ ]*\\)\"/    '\\1',/g' >> $@; "`,
		"cat $(SRCS) | grep '\\s*ip_block:' | sed -e 's/\\s*ip_block: \"\\([^ ]*\\)\"/    '\\1',/g' >> $@; ",
		true,
	},
}

var unquoteErrorTests = []struct {
	q  string // quoted
	s  string // unquoted, empty if unquote(s) will fail
	ok bool   // true iff unquote(s) should succeed
}{
	{`"\1"`, "\u0001", true},
	{`"\12"`, "\u000A", true},
	{`"\123"`, "\u0053", true},
	{`"\400"`, "", false},
	{`"\x"`, "", false},
	{`"\x1"`, "", false},
	{`"\x12"`, "\u0012", true},
	{`"\u"`, "", false},
	{`"\u1"`, "", false},
	{`"\u12"`, "", false},
	{`"\u123"`, "", false},
	{`"\u1234"`, "\u1234", true},
	{`"\uD7FF"`, "\uD7FF", true},
	{`"\uD800"`, "", false},
	{`"\uDFFF"`, "", false},
	{`"\uE000"`, "\uE000", true},
	{`"\uFFFF"`, "\uFFFF", true},
	{`"\u0000"`, "\u0000", true},
	{`"\U"`, "", false},
	{`"\U1"`, "", false},
	{`"\U12"`, "", false},
	{`"\U123"`, "", false},
	{`"\U1234"`, "", false},
	{`"\U12345"`, "", false},
	{`"\U123456"`, "", false},
	{`"\U1234567"`, "", false},
	{`"\U00012345"`, "\U00012345", true},
	{`"\U0000D7FF"`, "\uD7FF", true},
	{`"\U0000D800"`, "", false},
	{`"\U0000DFFF"`, "", false},
	{`"\U0000E000"`, "\uE000", true},
	{`"\U0000FFFF"`, "\uFFFF", true},
	{`"\U00000000"`, "\u0000", true},
	{`"\U0010FFFF"`, "\U0010FFFF", true},
	{`"\U00110000"`, "", false},
	{`"\UFFFFFFFF"`, "", false},
}

func TestQuote(t *testing.T) {
	for _, tt := range quoteTests {
		if !tt.std {
			continue
		}
		q := quote(tt.s, strings.HasPrefix(tt.q, `"""`))
		if q != tt.q {
			t.Errorf("quote(%#q) = %s, want %s", tt.s, q, tt.q)
		}
	}
}

func TestUnquote(t *testing.T) {
	for _, tt := range quoteTests {
		s, triple, err := Unquote(tt.q)
		wantTriple := strings.HasPrefix(tt.q, `"""`) || strings.HasPrefix(tt.q, `'''`)
		if s != tt.s || triple != wantTriple || err != nil {
			t.Errorf("unquote(%s) = %#q, %v, %v want %#q, %v, nil", tt.q, s, triple, err, tt.s, wantTriple)
		}
	}
}

var fstringUnquoteTests = []struct {
	q      string // quoted source (with f prefix)
	s      string // expected unquoted value
	triple bool
}{
	{`f""`, ``, false},
	{`f'single'`, `single`, false},
	{`f"\n{x}\t"`, "\n{x}\t", false},
	{`f"got \"{value}\""`, `got "{value}"`, false},
	{`f"""line1
{x}
line3"""`, "line1\n{x}\nline3", true},
}

func TestUnquoteFString(t *testing.T) {
	for _, tt := range fstringUnquoteTests {
		s, triple, err := Unquote(tt.q)
		if err != nil || s != tt.s || triple != tt.triple {
			t.Errorf("Unquote(%s) = %#q, %v, %v; want %#q, %v, nil", tt.q, s, triple, err, tt.s, tt.triple)
		}
	}
}

func TestUnquoteErrors(t *testing.T) {
	for _, tt := range unquoteErrorTests {
		s, triple, err := Unquote(tt.q)
		if tt.ok && (s != tt.s || err != nil) {
			t.Errorf("Unquote(%s) = %#q, %v, %v want %#q, %v, nil", tt.q, s, triple, err, tt.s, triple)
		} else if !tt.ok && err == nil {
			t.Errorf("Unquote(%s) = %#q, %v, %v want %#q, %v, non-nil", tt.q, s, triple, err, tt.s, triple)
		}
	}
}
