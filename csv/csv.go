package csv

import (
	"bufio"
	"errors"
	"io"
	"strings"

	"github.com/flarco/g"
)

type Csv struct {
	options CsvOptions
}

type CsvOptions struct {
	Delimiter byte
	Quote     byte
	NewLine   byte
	Header    bool
	Escape    byte
}

const (
	StateInQuote  = iota
	StateEscaping = iota
	StateRead     = iota
	StateRowEnded = iota
)

type CsvReader struct {
	reader     *bufio.Reader
	state      int
	csv        *Csv
	lineBuffer strings.Builder
	numFields  int
	line       uint64
	column     int
	token      Token
	cell       Cell
	row        Row
}

type Row []Cell

type Cell []Token

type Token struct {
	Start, End int
}

func NewCsv(options ...CsvOptions) *Csv {
	opts := CsvOptions{}
	if len(options) > 0 {
		opts = options[0]
	}
	if opts.Quote == 0 {
		opts.Quote = '"'
	}
	if opts.Delimiter == 0 {
		opts.Delimiter = ','
	}
	if opts.Escape == 0 {
		opts.Escape = '"'
	}
	if opts.NewLine == 0 {
		opts.NewLine = '\n'
	}
	return &Csv{options: opts}
}

func (c *Csv) NewReader(r io.Reader) *CsvReader {

	cr := &CsvReader{
		reader: bufio.NewReader(r),
		state:  StateRead,
		csv:    c,
		cell:   make(Cell, 0, 1),
	}
	return cr
}

func (cr *CsvReader) Read() (row []string, err error) {
	var ok bool
	for {
		line, hasMore, err := cr.reader.ReadLine()

		if err == io.EOF {
			if cr.state == StateInQuote {
				return row, errors.New(g.F("unterminated quoted field: line %d", cr.line))
			}
			return row, err
		} else if err != nil {
			return row, err
		}
		if !hasMore {
			line = append(line, '\n')
		}
		row, ok, err = cr.readLine(line)
		// debug

		// println()
		// g.Info(string(line))
		// g.Warn("buffer: %s", string(cr.lineBuffer))
		// g.Warn("row: %s", g.Marshal(cr.Row()))
		// g.Warn("cells: %s", g.Marshal(cr.row))
		// g.Warn("token: %s", g.Marshal(cr.token))
		// g.Warn("inQuote: %t    hasMore: %t", cr.state == StateInQuote, hasMore)
		if err != nil {
			return row, err
		}
		if ok {
			// row is complete
			break
		}
	}

	if len(row) > cr.numFields {
		cr.numFields = len(row)
	}

	return
}

func (cr *CsvReader) startToken(startColumn int) {
	cr.token = Token{Start: startColumn}
}

func (cr *CsvReader) endToken(endColumn int) {
	cr.token.End = endColumn
	if cr.token.Start != -1 && cr.token.End >= cr.token.Start {
		cr.cell = append(cr.cell, cr.token)
	}
	cr.token = Token{Start: -1, End: -1}
}

func (cr *CsvReader) endCell() {
	if len(cr.cell) > 0 {
		cr.row = append(cr.row, cr.cell)
	}
	cr.cell = make(Cell, 0, 1)
}

func (cr *CsvReader) readLine(line []byte) (row []string, ok bool, err error) {
	// var s strings.Builder
	if cr.state == StateRead || cr.state == StateRowEnded {
		cr.state = StateRead
		cr.lineBuffer.Reset()
		cr.lineBuffer.Write(line)
		cr.column = -1
		cr.cell = make(Cell, 0, 1)
		cr.row = cr.row[:0] // reset
		if cap(cr.row) < cr.numFields {
			cr.row = make(Row, 0, cr.numFields)
		}
		cr.startToken(0)
	} else {
		cr.lineBuffer.Write(line)
	}

	for i, char := range line {
		cr.column++

		// if quote is escaped, continue, handled next loop
		if cr.state == StateInQuote &&
			char == cr.csv.options.Escape &&
			len(line) > i+1 &&
			line[i+1] == cr.csv.options.Quote {
			cr.state = StateEscaping
			continue
		}

		switch char {
		case cr.csv.options.Quote:
			if cr.state == StateEscaping {
				// if is escaped, new token, same cell
				cr.endToken(cr.column - 1)
				cr.startToken(cr.column)
				cr.state = StateInQuote
			} else if cr.state == StateInQuote {
				// new cell
				cr.endToken(cr.column)
				cr.endCell()
				cr.state = StateRead
			} else if i == 0 || line[i-1] == cr.csv.options.Delimiter {
				cr.state = StateInQuote
				cr.endToken(cr.column)
				cr.startToken(cr.column + 1)
			}
		case cr.csv.options.Delimiter:
			if cr.state != StateInQuote {
				cr.endToken(cr.column)
				cr.endCell()
				cr.startToken(cr.column + 1)
			}
		case cr.csv.options.NewLine:
			if cr.state != StateInQuote {
				cr.endToken(cr.column)
				cr.endCell()
			}
		}
	}
	cr.line++

	// debug
	// g.Info(strings.TrimSpace(string(line)))
	// g.Warn("    token >>> %#v", cr.token)
	// g.Warn("    cell >>> %#v", cr.cell)
	// g.Warn("    row >>> %#v", cr.row)
	// g.Warn("    StateInQuote >>> %#v", cr.state == StateInQuote)

	ok = cr.state != StateInQuote
	if ok {
		cr.endCell()
		row = cr.Row()
		cr.state = StateRowEnded
	}

	return
}

func (cr *CsvReader) Row() (row []string) {
	var i int
	var cell Cell
	var token Token

	// debug
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		g.Warn("panic for token => %#v", token)
	// 		g.Warn("cell => %#v", cell)
	// 		g.Warn("cr.lineBuffer => %s", string(cr.lineBuffer))
	// 	}
	// }()

	// converts once to reduce allocations
	lineBuffer := cr.lineBuffer.String()

	row = make([]string, len(cr.row))
	for i, cell = range cr.row {
		for _, token = range cell {
			row[i] = row[i] + lineBuffer[token.Start:token.End]
		}
	}
	return
}

func (c *Csv) NewWriter(w io.Writer) *Writer {
	cr := &Writer{
		Comma: rune(c.options.Delimiter),
		w:     bufio.NewWriterSize(w, 40960),
		bytes: 0,
	}
	return cr
}
