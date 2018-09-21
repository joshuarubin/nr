package wordseq

// Copyright 2018 Joshua Rubin <joshua@rubixconsulting.com>
// Released under the MIT license

import (
	"bytes"
	"container/heap"
	"io"
	"strings"
	"testing"
)

func TestHeap(t *testing.T) {
	h := seqHeap{}
	heap.Init(h)

	seq := &Sequence{
		Words: []string{"a", "b", "c"},
		Count: 1,
	}

	heap.Push(h, seq)

	seq = &Sequence{
		Words: []string{"b", "c", "d"},
		Count: 2,
	}

	heap.Push(h, seq)

	seq.Count++
	heap.Fix(h, seq.index)

	seq = heap.Pop(h).(*Sequence)
	if seq.Count != 3 {
		t.Error("first pop didn't have count of 3")
	}

	seq = heap.Pop(h).(*Sequence)
	if seq.Count != 1 {
		t.Error("first pop didn't have count of 1")
	}

	if len(h) != 0 {
		t.Error("heap was not empty")
	}
}

func seqEqual(a, b *Sequence) bool {
	if a.Count != b.Count {
		return false
	}

	if len(a.Words) != len(b.Words) {
		return false
	}

	for i := range a.Words {
		if a.Words[i] != b.Words[i] {
			return false
		}
	}

	return true
}

func seqsEqual(a, b []*Sequence) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if !seqEqual(a[i], b[i]) {
			return false
		}
	}

	return true
}

func TestProcess(t *testing.T) {
	for _, v := range []struct {
		r      io.Reader
		expect []*Sequence
	}{{
		r: bytes.NewReader(nil),
	}, {
		r: strings.NewReader(""),
	}, {
		r: strings.NewReader("a"),
	}, {
		r: strings.NewReader("a b c"),
		expect: []*Sequence{{
			Words: []string{"a", "b", "c"},
			Count: 1,
		}},
	}, {
		r: strings.NewReader("a b c d"),
		expect: []*Sequence{{
			Words: []string{"a", "b", "c"},
			Count: 1,
		}, {
			Words: []string{"b", "c", "d"},
			Count: 1,
		}},
	}, {
		r: strings.NewReader("a b c a b c"),
		expect: []*Sequence{{
			Words: []string{"a", "b", "c"},
			Count: 2,
		}, {
			Words: []string{"b", "c", "a"},
			Count: 1,
		}, {
			Words: []string{"c", "a", "b"},
			Count: 1,
		}},
	}, {
		r: strings.NewReader("a b\nc"),
		expect: []*Sequence{{
			Words: []string{"a", "b", "c"},
			Count: 1,
		}},
	}, {
		r: strings.NewReader("a b (c)"),
		expect: []*Sequence{{
			Words: []string{"a", "b", "c"},
			Count: 1,
		}},
	}, {
		r: strings.NewReader("a b, c"),
		expect: []*Sequence{{
			Words: []string{"a", "b", "c"},
			Count: 1,
		}},
	}, {
		r: strings.NewReader("a B c"),
		expect: []*Sequence{{
			Words: []string{"a", "b", "c"},
			Count: 1,
		}},
	}} {
		seqs, err := Process(v.r, 3, 100)
		if err != nil {
			t.Fatal(err)
		}

		if !seqsEqual(v.expect, seqs) {
			t.Error("sequences not equal")
		}
	}
}
