// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csv

import (
	"bufio"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// A Writer writes records using CSV encoding.
//
// As returned by NewWriter, a Writer writes records terminated by a
// newline and uses ',' as the field delimiter. The exported fields can be
// changed to customize the details before the first call to Write or WriteAll.
//
// Comma is the field delimiter.
//
// If UseCRLF is true, the Writer ends each output line with \r\n instead of \n.
//
// The writes of individual records are buffered.
// After all data has been written, the client should call the
// Flush method to guarantee all data has been forwarded to
// the underlying io.Writer.  Any errors that occurred should
// be checked by calling the Error method.
type Writer struct {
	Comma   string // Field delimiter (set to ',' by NewWriter)
	UseCRLF bool   // True to use \r\n as the line terminator
	w       *bufio.Writer
	bytes   int
}

// NewWriter returns a new Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		Comma: ",",
		w:     bufio.NewWriterSize(w, 40960),
		bytes: 0,
	}
}

// NewWriterSize returns a new Writer that writes to w, along with buffer size.
func NewWriterSize(w io.Writer, size int) *Writer {
	return &Writer{
		Comma: ",",
		w:     bufio.NewWriterSize(w, size),
		bytes: 0,
	}
}

// Write writes a single CSV record to w along with any necessary quoting.
// A record is a slice of strings with each string being one field.
// Writes are buffered, so Flush must eventually be called to ensure
// that the record is written to the underlying io.Writer.
// it returns the total number of bytes written.
func (w *Writer) Write(record []string) (tbw int, err error) {
	defer func() { w.bytes = w.bytes + tbw }()

	if !validDelim(w.Comma) {
		return tbw, errInvalidDelim
	}

	for n, field := range record {
		if n > 0 {
			bw, err := w.w.WriteString(w.Comma)
			tbw = tbw + bw
			if err != nil {
				return tbw, err
			}
		}

		// If we don't have to have a quoted field then just
		// write out the field and continue to the next field.
		if !w.fieldNeedsQuotes(field) {
			bw, err := w.w.WriteString(field)
			tbw = tbw + bw
			if err != nil {
				return tbw, err
			}
			continue
		}

		if err := w.w.WriteByte('"'); err != nil {
			return tbw, err
		}
		tbw++
		for len(field) > 0 {
			// Search for special characters.
			i := strings.IndexAny(field, "\"\r\n")
			if i < 0 {
				i = len(field)
			}

			// Copy verbatim everything before the special character.
			bw, err := w.w.WriteString(field[:i])
			tbw = tbw + bw
			if err != nil {
				return tbw, err
			}
			field = field[i:]

			// Encode the special character.
			if len(field) > 0 {
				var err error
				var bw int
				switch field[0] {
				case '"':
					bw, err = w.w.WriteString(`""`)
				case '\r':
					if !w.UseCRLF {
						err = w.w.WriteByte('\r')
						bw++
					}
				case '\n':
					if w.UseCRLF {
						bw, err = w.w.WriteString("\r\n")
					} else {
						err = w.w.WriteByte('\n')
						bw++
					}
				}
				field = field[1:]
				tbw = tbw + bw
				if err != nil {
					return tbw, err
				}
			}
		}
		if err := w.w.WriteByte('"'); err != nil {
			return tbw, err
		}
		tbw++
	}
	var bw int
	if w.UseCRLF {
		bw, err = w.w.WriteString("\r\n")
	} else {
		err = w.w.WriteByte('\n')
		bw++
	}
	tbw = tbw + bw
	return tbw, err
}

// Bytes returns the number of bytes written
func (w *Writer) Bytes() int {
	return w.bytes
}

// Flush writes any buffered data to the underlying io.Writer.
// To check if an error occurred during the Flush, call Error.
func (w *Writer) Flush() {
	w.w.Flush()
}

// Error reports any error that has occurred during a previous Write or Flush.
func (w *Writer) Error() error {
	_, err := w.w.Write(nil)
	return err
}

// WriteAll writes multiple CSV records to w using Write and then calls Flush,
// returning any error from the Flush.
func (w *Writer) WriteAll(records [][]string) error {
	for _, record := range records {
		_, err := w.Write(record)
		if err != nil {
			return err
		}
	}
	return w.w.Flush()
}

// fieldNeedsQuotes reports whether our field must be enclosed in quotes.
// Fields with a Comma, fields with a quote or newline, and
// fields which start with a space must be enclosed in quotes.
// We used to quote empty strings, but we do not anymore (as of Go 1.4).
// The two representations should be equivalent, but Postgres distinguishes
// quoted vs non-quoted empty string during database imports, and it has
// an option to force the quoted behavior for non-quoted CSV but it has
// no option to force the non-quoted behavior for quoted CSV, making
// CSV with quoted empty strings strictly less useful.
// Not quoting the empty string also makes this package match the behavior
// of Microsoft Excel and Google Drive.
// For Postgres, quote the data terminating string `\.`.
func (w *Writer) fieldNeedsQuotes(field string) bool {
	if field == "" {
		return false
	}

	if field == `\.` {
		return true
	}

	if strings.Contains(field, w.Comma) || strings.ContainsAny(field, "\"\r\n") {
		return true
	}

	r1, _ := utf8.DecodeRuneInString(field)
	return unicode.IsSpace(r1)
}
