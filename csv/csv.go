package csv

import (
	"bufio"
	"io"
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
	StateRead     = iota
	StateRowEnded = iota
)

type CsvReader struct {
	reader     *bufio.Reader
	state      int
	csv        *Csv
	lineBuffer []byte
	lastRow    []string

	line   uint64
	column int
	token  Token
	cell   Cell
	row    Row
}

type Row []Cell

type Cell []Token

type Token struct {
	start, end int
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
		line, _, err := cr.reader.ReadLine()
		if err != nil {
			return row, err
		}
		row, ok, err = cr.readLine(append(line, '\n'))
		if err != nil {
			return row, err
		}
		if ok {
			// row is complete
			break
		}
	}
	cr.lastRow = row
	return
}

func (cr *CsvReader) startToken(startColumn int) {
	cr.token = Token{start: startColumn}
}

func (cr *CsvReader) endToken(endColumn int) {
	cr.token.end = endColumn
	if cr.token.start != -1 && cr.token.end > cr.token.start {
		cr.cell = append(cr.cell, cr.token)
	}
	cr.token = Token{start: -1, end: -1}
}

func (cr *CsvReader) endCell() {
	if len(cr.cell) > 0 {
		cr.row = append(cr.row, cr.cell)
	}
	cr.cell = make(Cell, 0, 1)
}

func (cr *CsvReader) readLine(line []byte) (row []string, ok bool, err error) {

	if cr.state == StateRead || cr.state == StateRowEnded {
		cr.state = StateRead
		cr.lineBuffer = line
		cr.column = -1
		cr.cell = make(Cell, 0, 1)
		cr.row = make(Row, 0, len(cr.lastRow))
		cr.startToken(0)
	} else {
		cr.lineBuffer = append(cr.lineBuffer, line...)
	}

	for i, char := range line {
		cr.column++

		// if quote is escaped, continue, handled next loop
		if char == cr.csv.options.Escape && cr.state == StateInQuote && len(line) > i+1 && line[i+1] == cr.csv.options.Quote {
			continue
		}

		switch char {
		case cr.csv.options.Quote:
			if cr.state == StateInQuote {
				if i > 0 && line[i-1] == cr.csv.options.Escape {
					// if is escaped, new token, same cell
					cr.endToken(cr.column - 1)
					cr.startToken(cr.column)
				} else {
					// new cell
					cr.endToken(cr.column)
					cr.endCell()
					cr.state = StateRead
				}
			} else {
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
	lineBuffer := string(cr.lineBuffer)

	row = make([]string, len(cr.row))
	for i, cell = range cr.row {
		for _, token = range cell {
			row[i] = row[i] + lineBuffer[token.start:token.end]
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
