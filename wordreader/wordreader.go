package wordreader

// Copyright 2018 Joshua Rubin <joshua@rubixconsulting.com>
// All rights reserved

import (
	"bufio"
	"bytes"
	"io"
	"unicode"
	"unicode/utf8"
)

const (
	carriageReturn = '\u000d'
	lineFeed       = '\u000a'
	doubleQuote    = '\u0022'
	singleQuote    = '\u0027'
	zwj            = '\u200d'
)

// WordReader is an interface wrapping a basic ReadWord method.
//
// ReadWord reads a single word, returning the word or any error encountered.
// At the end of the input it will return an empty word and io.EOF.
type WordReader interface {
	ReadWord() (string, error)
}

// New returns a new WordReader
func New(r io.Reader) WordReader {
	return &wordReader{
		Reader: bufio.NewReader(r),
	}
}

// wordReader takes an input io.Reader and parses it into words using the
// Unicode word-splitting algorithm in <URL:http://unicode.org/reports/tr29/>.
//
// Src is a bufio.Reader rather than an io.Reader, because word-reading requires
// the ability to read a rune at a time.
type wordReader struct {
	*bufio.Reader
	Buf bytes.Buffer
}

func (wr *wordReader) emitWord() (string, error) {
	word := wr.Buf.String()
	wr.Buf.Reset()
	return word, nil
}

func (wr *wordReader) emitWordPushRune(r rune) (string, error) {
	word := wr.Buf.String()
	wr.Buf.Reset()
	_, _ = wr.Buf.WriteRune(r) // #nosec

	// if the word is zero-length, try again
	if len(word) == 0 {
		return wr.ReadWord()
	}

	return word, nil
}

func getLastRune(data []byte) (r rune, size int) {
	r = utf8.RuneError

	if len(data) == 0 {
		return r, 0
	}

	pos := len(data) - 1
	if c := data[pos]; c < utf8.RuneSelf {
		return rune(c), 1
	}

	for pos--; pos >= 0 && r == utf8.RuneError; pos-- {
		r, size = utf8.DecodeRune(data[pos:])
	}

	return
}

func (wr *wordReader) lastRune() (rune, rune, rune) {
	lastRune := utf8.RuneError
	secondToLastRune := utf8.RuneError

	word := wr.Buf.Bytes()
	lastRuneLiteral, _ := getLastRune(word)

	for i := len(word); i >= 0; i-- {
		r, size := getLastRune(word[:i])
		if r == utf8.RuneError {
			break
		}
		i -= size - 1

		if extend(r) || format(r) {
			continue
		}

		if lastRune == utf8.RuneError {
			lastRune = r
			continue
		}

		if secondToLastRune == utf8.RuneError {
			secondToLastRune = r
			break
		}
	}

	return lastRune, lastRuneLiteral, secondToLastRune
}

func ahLetter(r rune) bool {
	return unicode.In(r, tableALetter, tableHebrewLetter)
}

func midLetter(r rune) bool {
	return unicode.In(r, tableMidLetter)
}

func midnum(r rune) bool {
	return unicode.In(r, tableMidNum)
}

func midNumLetQ(r rune) bool {
	if r == singleQuote {
		return true
	}

	return unicode.In(r, tableMidNumLet)
}

func numeric(r rune) bool {
	return unicode.In(r, tableNumeric)
}

func hebrew(r rune) bool {
	return unicode.In(r, tableHebrewLetter)
}

func katakana(r rune) bool {
	return unicode.In(r, tableKatakana)
}

func extendNumLet(r rune) bool {
	return unicode.In(r, tableExtendNumLet)
}

func eModifier(r rune) bool {
	return unicode.In(r, tableEModifier)
}

func eBase(r rune) bool {
	return unicode.In(r, tableEBase)
}

func ebg(r rune) bool {
	return unicode.In(r, tableEBaseGAZ)
}

func extend(r rune) bool {
	return unicode.In(r, tableExtend)
}

func format(r rune) bool {
	return unicode.In(r, tableFormat)
}

func glueAfterZWJ(r rune) bool {
	return unicode.In(r, tableGlueAfterZWJ)
}

func newline(r rune) bool {
	return unicode.In(r, tableNewline)
}

func ri(r rune) bool {
	return unicode.In(r, tableRegionalIndicator)
}

// ReadWord returns a single word from a wordReader's source.
func (wr *wordReader) ReadWord() (string, error) {
	for {
		r, _, err := wr.ReadRune()
		if err == io.EOF && wr.Buf.Len() > 0 {
			return wr.emitWord()
		}

		if err != nil {
			return "", err
		}

		lastRune, lastRuneLiteral, secondToLastRune := wr.lastRune()
		nextRune := wr.peekRune()

		switch {
		// Do not break within CRLF.
		case lastRuneLiteral == carriageReturn && r == lineFeed:
			// WB3	CR	×	LF
			_, _ = wr.Buf.WriteRune(r) // #nosec

		// Otherwise break before and after Newlines (including CR and LF)

		case newline(lastRune) || lastRune == carriageReturn || lastRune == lineFeed:
			// WB3a	(Newline | CR | LF)	÷
			return wr.emitWordPushRune(r)
		case newline(r) || r == carriageReturn || r == lineFeed:
			// WB3b	÷	(Newline | CR | LF)
			return wr.emitWordPushRune(r)

		// Do not break within emoji zwj sequences.

		case lastRune == zwj && (glueAfterZWJ(r) || ebg(r)):
			// WB3c	ZWJ	×	(Glue_After_Zwj | EBG)
			_, _ = wr.Buf.WriteRune(r) // #nosec

		// Ignore Format and Extend characters, except after sot, CR, LF, and
		// Newline. (See Section 6.2, Replacing Ignore Rules.) This also has the
		// effect of: Any × (Format | Extend | ZWJ

		case extend(r) || format(r) || r == zwj:
			// WB4	X (Extend | Format | ZWJ)*	→	X
			_, _ = wr.Buf.WriteRune(r) // #nosec

		// Do not break between most letters.

		case ahLetter(lastRune) && ahLetter(r):
			// WB5	AHLetter	×	AHLetter
			_, _ = wr.Buf.WriteRune(r) // #nosec

		// Do not break letters across certain punctuation.

		case ahLetter(lastRune) && (midLetter(r) || midNumLetQ(r)) && ahLetter(nextRune):
			// WB6	AHLetter	×	(MidLetter | MidNumLetQ) AHLetter
			_, _ = wr.Buf.WriteRune(r) // #nosec
		case ahLetter(secondToLastRune) && (midLetter(lastRune) || midNumLetQ(lastRune)) && ahLetter(r):
			// WB7	AHLetter (MidLetter | MidNumLetQ)	×	AHLetter
			_, _ = wr.Buf.WriteRune(r) // #nosec
		case hebrew(lastRune) && r == singleQuote:
			// WB7a		Hebrew_Letter	×	Single_Quote
			_, _ = wr.Buf.WriteRune(r) // #nosec
		case hebrew(lastRune) && r == doubleQuote && hebrew(nextRune):
			// WB7b		Hebrew_Letter	×	Double_Quote Hebrew_Letter
			_, _ = wr.Buf.WriteRune(r) // #nosec
		case hebrew(secondToLastRune) && lastRune == doubleQuote && hebrew(r):
			// WB7c		Hebrew_Letter Double_Quote	×	Hebrew_Letter
			_, _ = wr.Buf.WriteRune(r) // #nosec

		// Do not break within sequences of digits, or digits adjacent to
		// letters (“3a”, or “A3”).

		case numeric(lastRune) && numeric(r):
			// WB8	Numeric	×	Numeric
			_, _ = wr.Buf.WriteRune(r) // #nosec
		case ahLetter(lastRune) && numeric(r):
			// WB9	AHLetter	×	Numeric
			_, _ = wr.Buf.WriteRune(r) // #nosec
		case numeric(lastRune) && ahLetter(r):
			// WB10	Numeric	×	AHLetter
			_, _ = wr.Buf.WriteRune(r) // #nosec

		// Do not break within sequences, such as “3.2” or “3,456.789”.

		case numeric(secondToLastRune) && (midnum(lastRune) || midNumLetQ(lastRune)) && numeric(r):
			// WB11	Numeric (MidNum | MidNumLetQ)	×	Numeric
			_, _ = wr.Buf.WriteRune(r) // #nosec
		case numeric(lastRune) && (midnum(r) || midNumLetQ(r)) && numeric(nextRune):
			// WB12	Numeric	×	(MidNum | MidNumLetQ) Numeric
			_, _ = wr.Buf.WriteRune(r) // #nosec

		// Do not break between Katakana.

		case katakana(lastRune) && katakana(r):
			// WB13	Katakana	×	Katakana
			_, _ = wr.Buf.WriteRune(r) // #nosec

		// Do not break from extenders.

		case (ahLetter(lastRune) || numeric(lastRune) || katakana(lastRune) || extendNumLet(lastRune)) && extendNumLet(r):
			// WB13a	(AHLetter | Numeric | Katakana | ExtendNumLet)	×	ExtendNumLet
			_, _ = wr.Buf.WriteRune(r) // #nosec
		case extendNumLet(lastRune) && (ahLetter(r) || numeric(r) || katakana(r)):
			// WB13b	ExtendNumLet	×	(AHLetter | Numeric | Katakana)
			_, _ = wr.Buf.WriteRune(r) // #nosec

		// Do not break within emoji modifier sequences.

		case (eBase(lastRune) || ebg(lastRune)) && eModifier(r):
			// WB14	(E_Base | EBG)	×	E_Modifier
			_, _ = wr.Buf.WriteRune(r) // #nosec

		// Do not break within emoji flag sequences. That is, do not break
		// between regional indicator (RI) symbols if there is an odd number of
		// RI characters before the break point.

		case !ri(secondToLastRune) && ri(lastRune) && ri(r):
			// WB15	^ (RI RI)* RI	×	RI
			// WB16	[^RI] (RI RI)* RI	×	RI
			_, _ = wr.Buf.WriteRune(r) // #nosec

		default:
			return wr.emitWordPushRune(r)
		}
	}
}

func (wr *wordReader) peekRune() rune {
	r, _, err := wr.ReadRune()
	if err != nil {
		return utf8.RuneError
	}
	_ = wr.UnreadRune() // #nosec
	return r
}
