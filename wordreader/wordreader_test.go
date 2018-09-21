package wordreader

// Copyright 2018 Joshua Rubin <joshua@rubixconsulting.com>
// All rights reserved

import (
	"io"
	"strings"
	"testing"
)

type splitTest struct {
	str   string
	words []string
}

var tests = []splitTest{
	// all tests include
	// <URL:http://unicode.org/reports/tr29/#WB1> &
	// <URL:http://unicode.org/reports/tr29/#WB1> by default

	{" foo ", []string{" ", "foo", " "}},
	{"don't go", []string{"don't", " ", "go"}},
	{"וכו׳ פרד״ס", []string{"וכו׳", " ", "פרד״ס"}},

	// http://unicode.org/reports/tr29/#WB3
	{"foo\r\nbar", []string{"foo", "\r\n", "bar"}},
	{"\r\nfoo\r\nbar\r\n", []string{"\r\n", "foo", "\r\n", "bar", "\r\n"}},

	// http://unicode.org/reports/tr29/#WB3a
	// http://unicode.org/reports/tr29/#WB3b
	{"foo\nbar", []string{"foo", "\n", "bar"}},
	{"\nfoo\n", []string{"\n", "foo", "\n"}},
	{"foo\rbar", []string{"foo", "\r", "bar"}},
	{"\rfoo\r", []string{"\r", "foo", "\r"}},
	{"foo\vbar", []string{"foo", "\v", "bar"}},

	// http://unicode.org/reports/tr29/#WB3c
	{"👨‍👩‍👧", []string{"👨‍👩‍👧"}},
	{"‍👨‍👩‍👧‍", []string{"‍👨‍👩‍👧‍"}}, // there's a ZWJ fore & aft here

	// TODO: http://unicode.org/reports/tr29/#WB4

	// http://unicode.org/reports/tr29/#WB5
	{"fooכbar baz", []string{"fooכbar", " ", "baz"}},

	// http://unicode.org/reports/tr29/#WB6
	// http://unicode.org/reports/tr29/#WB7
	{"foo:bar baz·quux", []string{"foo:bar", " ", "baz·quux"}},
	{"foo.bar baz'quux", []string{"foo.bar", " ", "baz'quux"}},

	// http://unicode.org/reports/tr29/#WB7a
	// http://unicode.org/reports/tr29/#WB7b
	// http://unicode.org/reports/tr29/#WB7c
	{"וכו' פרד\"ס", []string{"וכו'", " ", "פרד\"ס"}},

	// http://unicode.org/reports/tr29/#WB8
	{"12 34 56", []string{"12", " ", "34", " ", "56"}},

	// http://unicode.org/reports/tr29/#WB9
	// http://unicode.org/reports/tr29/#WB10
	{"12ab34 56אב78", []string{"12ab34", " ", "56אב78"}},

	// http://unicode.org/reports/tr29/#WB11
	// http://unicode.org/reports/tr29/#WB12
	{"1.21 gigawatts", []string{"1.21", " ", "gigawatts"}},
	{"1,21 gigawatten", []string{"1,21", " ", "gigawatten"}},

	// http://unicode.org/reports/tr29/#WB13
	{"ツアひらがな", []string{"ツア", "ひ", "ら", "が", "な"}},

	// TODO: add tests for WB13a, WB13b & WB14

	// http://unicode.org/reports/tr29/#WB15
	// http://unicode.org/reports/tr29/#WB16
	{"ab🇺🇸🇺🇸 cd", []string{"ab", "🇺🇸", "🇺🇸", " ", "cd"}},
	{"ab🇺🇸🇺🇸🇺🇸 cd", []string{"ab", "🇺🇸", "🇺🇸", "🇺🇸", " ", "cd"}},
	{"b🇺🇸🇺🇸🇺🇸 cd", []string{"b", "🇺🇸", "🇺🇸", "🇺🇸", " ", "cd"}},
	{"b🇺🇺🇸🇺🇸 cd", []string{"b", "🇺🇺", "🇸🇺", "🇸", " ", "cd"}},

	// http://unicode.org/reports/tr29/#WB999
	{"Հայոց գրեր 학생들은 읽기와น\u0e31กเร\u0e35ยนได\u0e49เร", []string{"Հայոց", " ", "գրեր", " ", "학생들은", " ", "읽기와", "น\u0e31", "ก", "เ", "ร\u0e35", "ย", "น", "ไ", "ด\u0e49", "เ", "ร"}},
	{"ನ\u0cc0ವ\u0cc1 ಹ\u0cc7ಗ\u0cbfದ\u0ccdದ\u0cc0ರ\u0cbf", []string{"ನ\u0cc0ವ\u0cc1", " ", "ಹ\u0cc7ಗ\u0cbfದ\u0ccdದ\u0cc0ರ\u0cbf"}},
	{"尔布尔", []string{"尔", "布", "尔"}},
	{"य\u0942 क\u0948स\u0947 ह\u0948\u0902", []string{"य\u0942", " ", "क\u0948स\u0947", " ", "ह\u0948\u0902"}},
	{"foo, bar and baz", []string{"foo", ",", " ", "bar", " ", "and", " ", "baz"}},

	// some tests inspired by coveralls
	{"\"foo bar baz\"", []string{"\"", "foo", " ", "bar", " ", "baz", "\""}},
	{"\"\u05d0", []string{"\"", "\u05d0"}},
	{"\u05d0\u05d0\"\u05d0", []string{"\u05d0\u05d0\"\u05d0"}},
	// per http://unicode.org/reports/tr29/#WB7b &
	// http://unicode.org/reports/tr29/#WB7c, a double quote
	// _before_ a Hebrew letter is not part of a word with the
	// letter
	{"\u05d0\u05d0\"\u05d0", []string{"\u05d0\u05d0\"\u05d0"}},
	{"foo'-dot", []string{"foo", "'", "-", "dot"}},
	{"Āll A\u0301ll test\u00adi\u00adfy\u00ading test·\u00adi2\u00adfyア\u00ading", []string{"Āll", " ", "A\u0301ll", " ", "test\u00adi\u00adfy\u00ading", " ", "test", "·\u00ad", "i2\u00adfy", "ア\u00ad", "ing"}},
	{"ア'", []string{"ア", "'"}},
	{"foo\u202fbar格\u202f尔", []string{"foo\u202fbar", "格", "\u202f", "尔"}},
	{"كنت أردت أن أقر", []string{"كنت", " ", "أردت", " ", "أن", " ", "أقر"}},
	{"ነጋሲ ደገፋ ሃይሉ ኢትዮጵያ ተወልዶ ኣደገ።", []string{"ነጋሲ", " ", "ደገፋ", " ", "ሃይሉ", " ", "ኢትዮጵያ", " ", "ተወልዶ", " ", "ኣደገ", "።"}},
	{"foo bar", []string{"foo", " ", "bar"}},
	{"foo. bar", []string{"foo", ".", " ", "bar"}},
	{"foo 3.2 bar", []string{"foo", " ", "3.2", " ", "bar"}},
	{"foo 3,456.789 bar", []string{"foo", " ", "3,456.789", " ", "bar"}},
}

func TestWordSplitter(t *testing.T) {
	t.Run("ReadWord", func(t *testing.T) {
		t.Parallel()

		for _, test := range tests {
			str := test.str
			words := test.words
			wr := New(strings.NewReader(str))

			var err error
			var readWord string

			for _, word := range words {
				if readWord, err = wr.ReadWord(); err != nil {
					t.Fatal(err)
				}

				if readWord != word {
					t.Errorf("%s != %s", readWord, word)
				}
			}

			if err == nil {
				readWord, err = wr.ReadWord()
				if readWord != "" {
					t.Error("readWord wasn't empty")
				}
			}

			if err != io.EOF {
				t.Error("err != io.EOF")
			}
		}
	})

	t.Run("Misc", func(t *testing.T) {
		t.Parallel()

		wr := New(strings.NewReader("a"))

		word, err := wr.ReadWord()
		if err != nil {
			t.Fatal(err)
		}

		if word != "a" {
			t.Error("word != a")
		}

		word, err = wr.ReadWord()
		if err != io.EOF {
			t.Fatal(err)
		}

		if word != "" {
			t.Errorf("word(%s) != \"\"", word)
		}
	})
}
