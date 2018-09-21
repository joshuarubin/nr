package main

// Copyright 2018 Joshua Rubin <joshua@rubixconsulting.com>
// Released under the MIT license

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"text/tabwriter"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/unicode"
	"jrubin.io/nr/wordseq"
)

type config struct {
	Encoding     string
	SequenceSize int
	TopN         int
}

func initFlags(c *config) *flag.FlagSet {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	fs.Usage = func() {
		_, _ = fmt.Fprintf(
			fs.Output(),
			`Usage of %s:

	%s file1.txt file2.txt ...
	cat file1.txt | %s

	A filename argument of '-' indicates that stdin should be read.
	If no filenames are given, input is assumed to come from stdin.

flags:
`,
			os.Args[0],
			os.Args[0],
			os.Args[0],
		)
		fs.PrintDefaults()
	}

	fs.StringVar(
		&c.Encoding,
		"encoding",
		"",
		"file encoding of all files, including stdin, valid values defined at https://www.w3.org/TR/encoding/",
	)

	fs.IntVar(
		&c.SequenceSize,
		"sequence-size",
		3,
		"number of words per sequence",
	)

	fs.IntVar(
		&c.TopN,
		"n",
		100,
		"only show the top n sequences with the highest frequency count",
	)

	_ = fs.Parse(os.Args[1:]) // #nosec

	return fs
}

func main() {
	var c config
	fs := initFlags(&c)

	if err := run(c, fs.Args()...); err != nil {
		log.Fatalf("%+v", err)
	}
}

func run(c config, args ...string) error {
	// build a list of all the things to read from

	var readers []io.Reader
	for _, fn := range args {
		if fn == "-" {
			readers = append(readers, os.Stdin)
		}

		f, err := os.Open(fn)
		if err != nil {
			return err
		}
		defer f.Close()
		readers = append(readers, f)
	}

	if len(readers) == 0 {
		readers = append(readers, os.Stdin)
	}

	// concatenate the readers
	reader := io.MultiReader(readers...)

	// ensure that the encoding is converted to utf-8
	var enc encoding.Encoding

	if c.Encoding == "" {
		// try to determine the encoding
		buf := make([]byte, 1024)
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		buf = buf[:n]

		// reset the reader so nothing is lost
		reader = io.MultiReader(bytes.NewReader(buf), reader)

		var name string
		var certain bool
		enc, name, certain = charset.DetermineEncoding(buf, "")
		if certain {
			log.Printf("detected %s encoding", name)
		} else {
			log.Printf("could not determine encoding, presuming utf-8")
			enc = encoding.Nop
		}
	} else {
		var err error
		if enc, err = htmlindex.Get(c.Encoding); err != nil {
			return err
		}
	}

	if enc != unicode.UTF8 {
		reader = enc.NewDecoder().Reader(reader)
	}

	// read all the content
	seqs, err := wordseq.Process(reader, c.SequenceSize, c.TopN)
	if err != nil {
		return err
	}

	// write out the results
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)

	for _, seq := range seqs {
		fmt.Fprintf(w, "%d\t %v\n", seq.Count, seq.Words)
	}

	return w.Flush()
}
