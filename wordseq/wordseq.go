package wordseq

// Copyright 2018 Joshua Rubin <joshua@rubixconsulting.com>
// Released under the MIT license

import (
	"container/heap"
	"crypto/sha1"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"jrubin.io/nr/wordreader"
)

type Sequence struct {
	Words []string
	Count int

	index int
}

type seqHeap map[int]*Sequence

var _ heap.Interface = (*seqHeap)(nil)

func (h seqHeap) Len() int {
	return len(h)
}

func (h seqHeap) Less(i, j int) bool {
	// TODO(jrubin) what if the counts are equal?
	return h[i].Count > h[j].Count
}

func (h seqHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h seqHeap) Push(x interface{}) {
	s := x.(*Sequence)
	s.index = len(h)
	h[len(h)] = s
}

func (h seqHeap) Pop() interface{} {
	item := h[len(h)-1]
	delete(h, len(h)-1)
	item.index = -1
	return item
}

// basically the same as unicode.IsSpace but works on strings and includes CRLF
func isSpace(s string) bool {
	switch s {
	case " ", "\t", "\n", "\v", "\f", "\r", "\u0085", "\u00a0", "\r\n", "\n\r":
		return true
	}
	return false
}

func Process(n io.Reader, seqSize, topN int) ([]*Sequence, error) {
	wr := wordreader.New(n)

	window := make([]string, 0, seqSize+1)

	// cache needed to index by sequence words
	cache := map[[sha1.Size]byte]*Sequence{}

	// heap needed to keep sorted sequence counts
	h := seqHeap{}
	heap.Init(h)

	for {
		// read in a word at a time
		word, err := wr.ReadWord()

		if err == io.EOF {
			break // finished reading words
		}

		if err != nil {
			return nil, err
		}

		if isSpace(word) {
			continue
		}

		w := make([]rune, 0, utf8.RuneCountInString(word))
		for _, r := range word {
			if unicode.IsPunct(r) {
				// ignore punctuation
				continue
			}

			// convert to lower case
			// TODO(jrubin) should runes such as 'Ü' be equivalent to 'u'
			w = append(w, unicode.ToLower(r))
		}

		if len(w) == 0 {
			continue
		}

		window = append(window, string(w))

		if len(window) < seqSize {
			// the window isn't yet full, continue adding words until it is
			continue
		}

		seq := window       // seq holds the current N word sequence
		window = window[1:] // slide the window to the right

		// sha1 to ensure key size is fixed while remaining fast enough
		// NULL can't exist in the word, so use it as a joiner
		key := sha1.Sum([]byte(strings.Join(seq, "\x00")))

		if item, ok := cache[key]; ok {
			item.Count++
			heap.Fix(h, item.index)
			continue
		}

		item := &Sequence{
			Words: seq,
			Count: 1,
		}
		cache[key] = item
		heap.Push(h, item)
	}

	// build the return slice limited to the topN most frequent sequences

	ret := make([]*Sequence, 0, topN)

	for len(ret) < topN && h.Len() > 0 {
		ret = append(ret, heap.Pop(h).(*Sequence))
	}

	return ret, nil
}