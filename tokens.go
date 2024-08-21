// Package g provides tokenization and token manipulation utilities.

package g

import (
	"regexp"
	"strings"
	"unicode"
)

// Token represents a group of characters in a tokenized text.
type Token struct {
	Text             string   `json:"text"`             // The actual text content of the token
	Index            int      `json:"index"`            // The index of the token in the token list
	IsComment        bool     `json:"-"`                // Indicates if the token is a comment
	ParenthesisLevel int      `json:"parenthesisLevel"` // The nesting level of parentheses at this token
	Position         Position `json:"position"`         // The starting position of the token in the original text

	previous *Token // Pointer to the previous token
	next     *Token // Pointer to the next token

	body *TokenBody // Pointer to the TokenBody this token belongs to
}

// Position represents the line and column number of a token in the original text.
type Position struct {
	Line   int `json:"line"`   // The line number (1-based)
	Column int `json:"column"` // The column number (0-based)
}

// IsEmpty returns true if the token is empty or has an invalid index.
func (t Token) IsEmpty() bool {
	return t.Text == "" || t.Index == -1
}

// IsNotEmpty returns true if the token is not empty and has a valid index.
func (t Token) IsNotEmpty() bool {
	return !t.IsEmpty()
}

// IsWhitespace returns true if the token consists only of whitespace characters.
func (t Token) IsWhitespace() bool {
	return isWhite(t.Text)
}

// IsNumeric returns true if the token consists only of numeric characters.
func (t Token) IsNumeric() bool {
	return isNumeric(t.Text)
}

// IsWord returns true if the token consists of word characters (alphanumeric or underscore).
func (t Token) IsWord() bool {
	return isWord(t.Text)
}

// IsOperand returns true if the token is an operand (e.g., +, -, *, /).
func (t Token) IsOperand() bool {
	return isOperand(t.Text)
}

// IsAlphaNumericID returns true if all chars are alphanumeric or ID quotes.
func (t Token) IsAlphaNumericID() bool {
	if strings.TrimSpace(t.Text) == "" {
		return false
	}
	pattern := `[^_"` + "`" + `a-zA-Z0-9]`
	nonAlphaNumericRegex := *regexp.MustCompile(pattern)
	return len(t.Text) > 0 && !nonAlphaNumericRegex.Match([]byte(strings.TrimSpace(t.Text)))
}

// IsSingleSQLExpression returns true if the token represents a single SQL expression.
func (t Token) IsSingleSQLExpression() bool {
	pattern := `[^_"` + "`" + `a-zA-Z0-9:()]`
	nonAlphaNumericRegex := *regexp.MustCompile(pattern)
	return len(t.Text) > 0 && !nonAlphaNumericRegex.Match([]byte(strings.TrimSpace(t.Text)))
}

// Previous returns the previous `n` token relative to the provided token. `n` defaults to 1.
func (t Token) Previous(n ...int) (pt Token) {
	N := 1
	if len(n) > 0 {
		N = n[0]
	}
	pt.Index = -1
	pt.body = t.body
	p := t.Index - N
	if p >= 0 && p < len(t.body.Tokens) {
		pt = t.body.Tokens[p]
	}
	return pt
}

// Next returns the next `n` token relative to the provided token. `n` defaults to 1.
func (t Token) Next(n ...int) (nt Token) {
	N := 1
	if len(n) > 0 {
		N = n[0]
	}
	nt.Index = -1
	nt.body = t.body
	p := t.Index + N
	if p >= 0 && t.body != nil && p < len(t.body.Tokens) {
		nt = t.body.Tokens[p]
	}
	return nt
}

// FindNext advances until text is matched.
func (t Token) FindNext(s string) (nt Token) {
	nt = t
	for nt.HasNext() {
		nt = nt.Next()
		if !nt.HasNext() {
			break
		} else if nt.EqualsFold(s) {
			return
		}
	}
	return Token{Index: -1}
}

// FindNextWithinLevel advances within the specified max parenthesis level until text is matched.
func (t Token) FindNextWithinLevel(s string, maxParLevel int) (nt Token) {
	nt = t
	for nt.HasNext() {
		nt = nt.Next()
		if !nt.HasNext() {
			break
		} else if nt.ParenthesisLevel <= maxParLevel && nt.EqualsFold(s) {
			return
		}
	}
	return Token{Index: -1}
}

// IsAlone returns `true` if surrounded by white space.
func (t Token) IsAlone() bool {
	return t.Previous().IsWhitespace() && t.Next().IsWhitespace()
}

// NextNonWhitespace returns the next non-whitespace token.
func (t Token) NextNonWhitespace() (nt Token) {
	nt = t
	for nt.HasNext() {
		nt = nt.Next()
		if !nt.IsWhitespace() {
			return
		}
	}
	return Token{Index: -1}
}

// NextWord returns the next word token.
func (t Token) NextWord() (nt Token) {
	nt = t
	for nt.HasNext() {
		nt = nt.Next()
		if nt.IsWord() {
			return
		}
	}
	return Token{Index: -1}
}

// NextWordOrID returns the next word or identifier token.
func (t Token) NextWordOrID() (nt Token) {
	nt = t
	for nt.HasNext() {
		nt = nt.Next()
		if nt.IsWord() {
			return
		}
	}
	return Token{Index: -1}
}

// PreviousNonWhitespace returns the previous non-whitespace token.
func (t Token) PreviousNonWhitespace() (pt Token) {
	pt = t
	for pt.HasNext() {
		pt = pt.Previous()
		if !pt.IsWhitespace() {
			return
		}
	}
	return Token{Index: -1}
}

// HasPrevious returns true if there is a previous token.
func (t Token) HasPrevious() bool {
	return t.Previous().Index != -1
}

// HasNext returns true if there is a next token.
func (t Token) HasNext() bool {
	return t.Next().Index != -1
}

// EqualsFold compares the token text with the given string, ignoring case.
func (t Token) EqualsFold(s string) bool {
	return strings.EqualFold(t.Text, s)
}

// InFold checks if the token text matches any of the given values, ignoring case.
func (t Token) InFold(vals ...string) bool {
	for _, val := range vals {
		if strings.EqualFold(t.Text, val) {
			return true
		}
	}
	return false
}

// Select returns a slice of tokens starting from the current token and including
// 'delta' number of tokens (positive for forward, negative for backward).
func (t Token) Select(delta int) (nts Tokens) {
	counter := 0
	nts = Tokens{t}
	if delta == 0 {
		return nts.Recreate()
	} else if delta > 0 {
		for {
			if counter == delta || !t.HasNext() {
				break
			}
			counter++
			t = t.Next()
			nts = append(nts, t)
		}
	} else if delta < 0 {
		for {
			if counter == delta || !t.HasPrevious() {
				break
			}
			counter--
			t = t.Previous()
			nts = append(Tokens{t}, nts...)
		}
	}
	return nts.Recreate()
}

// SelectNonWhitespace returns a slice of non-whitespace tokens starting from the current token
// and including 'delta' number of tokens (positive for forward, negative for backward).
func (t Token) SelectNonWhitespace(delta int) (nts Tokens) {
	counter := 0
	if !t.IsWhitespace() {
		nts = Tokens{t}
	}
	if delta == 0 {
		return nts.Recreate()
	} else if delta > 0 {
		for {
			if counter == delta || !t.HasNext() {
				break
			}
			counter++
			t = t.NextNonWhitespace()
			if t.Index != -1 {
				nts = append(nts, t)
			}
		}
	} else if delta < 0 {
		for {
			if counter == delta || !t.HasPrevious() {
				break
			}
			counter--
			t = t.PreviousNonWhitespace()
			if t.Index != -1 {
				nts = append(Tokens{t}, nts...)
			}
		}
	}
	return nts.Recreate()
}

// Tokens represents a slice of Token.
type Tokens []Token

// TokenGroups represents a slice of Tokens.
type TokenGroups []Tokens

// First returns the first token in the Tokens slice.
func (ts Tokens) First() Token {
	if len(ts) > 0 {
		return ts[0]
	}
	return Token{Index: -1}
}

// Last returns the last token in the Tokens slice.
func (ts Tokens) Last() Token {
	if len(ts) > 0 {
		return ts[len(ts)-1]
	}
	return Token{Index: -1}
}

// Join concatenates all token texts into a single string.
func (ts Tokens) Join() string {
	builder := strings.Builder{}
	for _, t := range ts {
		builder.WriteString(t.Text)
	}
	return builder.String()
}

// IsEmpty returns true if the Tokens slice is empty.
func (ts Tokens) IsEmpty() bool {
	return len(ts) == 0
}

// TrimComments removes comment tokens from the Tokens slice.
func (ts Tokens) TrimComments() (nwTs Tokens) {
	for _, t := range ts {
		if !t.IsComment {
			nwTs = append(nwTs, t)
		}
	}
	return nwTs.Recreate()
}

// TrimParenthesis removes enclosing first/last parenthesis tokens.
func (ts Tokens) TrimParenthesis() (nwTs Tokens) {
	doTrim := func(ts Tokens) (newTs Tokens) {
		for i, t := range ts {
			if In(i, 0, len(ts)-1) && t.InFold("(", ")") {
				continue // typically will be first or last token
			}
			newTs = append(newTs, t)
		}
		return newTs.Recreate()
	}

	nwTs = ts
	for nwTs.First().EqualsFold("(") && nwTs.Last().EqualsFold(")") {
		nwTs = doTrim(nwTs)
	}

	return
}

// TrimWhiteSpace removes whitespace tokens from the start and end of the Tokens slice.
func (ts Tokens) TrimWhiteSpace() (nwTs Tokens) {
	// from start
	foundNonWhiteSpace := -1
	for i, t := range ts {
		if !t.IsWhitespace() {
			foundNonWhiteSpace = i
		}
		if foundNonWhiteSpace > -1 {
			nwTs = append(nwTs, t)
		}
	}

	// from end
	foundNonWhiteSpace = -1
	for i := range nwTs {
		j := len(nwTs) - i - 1
		if !nwTs[j].IsWhitespace() {
			foundNonWhiteSpace = j
			break
		}
	}

	return nwTs[:foundNonWhiteSpace+1].Recreate()
}

// Recreate rebuilds the tokens, removing the external body pointers.
func (ts Tokens) Recreate() (nwTs Tokens) {
	body := TokenBody{}
	for _, tok := range ts {
		body.AddToken(tok)
	}
	return body.Tokens
}

// StringSlice returns a slice of token texts.
func (ts Tokens) StringSlice() []string {
	ss := make([]string, len(ts))
	for i := range ts {
		ss[i] = ts[i].Text
	}
	return ss
}

// Body returns the TokenBody associated with the Tokens.
func (ts Tokens) Body() *TokenBody {
	return ts.First().body
}

// TokenBody represents a collection of tokens and their base parenthesis level.
type TokenBody struct {
	Tokens               Tokens
	baseParenthesisLevel int
}

// AddToken adds a new token to the TokenBody.
func (tb *TokenBody) AddToken(t Token) Token {
	t.previous = nil
	t.next = nil
	if len(tb.Tokens) > 0 {
		t.previous = &tb.Tokens[len(tb.Tokens)-1]
		t.previous.next = &t
	}
	t.Index = len(tb.Tokens)
	t.body = tb
	tb.Tokens = append(tb.Tokens, t)
	return t
}

var (
	nonWordRegex = *regexp.MustCompile(`[^_a-zA-Z0-9]`)
)

// isOperand checks if a string consists only of operand characters.
func isOperand(s string) bool {
	if len(s) == 0 {
		return false
	}

	for _, c := range s {
		switch c {
		case '.', ',', '+', '-', '*', '/', '=', '<', '>', '!', '~':
		case ';':
		default:
			return false
		}
	}

	return true
}

// isWhite checks if a string consists only of whitespace characters.
func isWhite(s string) bool {
	if len(s) == 0 {
		return false
	}

	for _, c := range s {
		if !unicode.IsSpace(c) {
			return false
		}
	}
	return true
}

// isNonWord checks if a string contains any non-word characters.
func isNonWord(s string) bool {
	return len(s) > 0 && nonWordRegex.Match([]byte(s))
}

func isWord(s string) bool {
	return !isNonWord(s)
}

func isNumeric(s string) bool {
	// return len(s) > 0 && !nonNumericRegex.Match([]byte(s))

	if len(s) == 0 {
		return false
	}

	isNumeric := true
	for _, c := range s {
		if !unicode.IsDigit(c) {
			isNumeric = false
			break
		}
	}
	return isNumeric
}

type TokenizeOptions struct{}

// TokenizeWithMapIDs map of char index to line-column ID
func Tokenize(text string, options *TokenizeOptions) (body TokenBody) {
	// wordChars := CharsToMap("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890_")

	var token, char, pChar, nChar string
	var inTickQ, inSingleQ, inDoubleQ, escaping bool
	var inCommentLine, inCommentMulti, newLine bool
	var i, parenthesisLevel, startLineNumber, startColNumber, lineNumber, colNumber int

	body = TokenBody{}

	inQuote := func() bool { return inSingleQ || inDoubleQ || inTickQ }
	inComment := func() bool { return inCommentLine || inCommentMulti }
	firstKeyword := ""

	reset := func() {
		token = ""
		startLineNumber = lineNumber
		startColNumber = colNumber - 1
	}
	addTokenAndReset := func() {
		if token == "" {
			return
		}
		t := Token{
			Text:             token,
			ParenthesisLevel: parenthesisLevel,
			Position:         Position{startLineNumber, startColNumber},
		}
		body.AddToken(t)
		reset()

		// detect baseParenthesisLevel
		if firstKeyword == "" {
			if !t.IsComment && !t.IsWhitespace() && !t.IsOperand() && t.Text != "(" {
				firstKeyword = t.Text
			} else if t.Text == "(" {
				body.baseParenthesisLevel = parenthesisLevel
			}
		}
	}
	append := func() {
		token = token + char
	}
	appendAndAdd := func() {
		append()
		addTokenAndReset()
	}
	addResetAndAppend := func() {
		addTokenAndReset()
		append()
	}

	for i = range text {
		char = string(text[i])
		// previous
		if i > 0 {
			pChar = string(text[i-1])
		}

		// next
		nChar = ""
		if i+1 < len(text) {
			nChar = string(text[i+1])
		}

		// line & column numbers
		if char == "\n" {
			newLine = true
			lineNumber++
			colNumber = 0
		} else if newLine {
			newLine = false
		} else {
			colNumber++
		}

		// comments
		{
			if !inQuote() && !inComment() && char == "-" && nChar == "-" {
				addResetAndAppend()
				inCommentLine = true
				continue
			} else if inCommentLine && newLine {
				appendAndAdd()
				inCommentLine = false
				continue
			} else if !inQuote() && !inComment() && char == "/" && nChar == "*" {
				addTokenAndReset()
				inCommentMulti = true
				append()
				continue
			} else if inCommentMulti && pChar == "*" && char == "/" {
				appendAndAdd()
				inCommentMulti = false
				continue
			} else if inComment() {
				append()
				continue // no need to process comment text
			}
		}

		// parenthesis
		{
			switch {
			case !inQuote() && char == "(":
				addTokenAndReset()
				parenthesisLevel++
				appendAndAdd()
				startColNumber++
				continue
			case !inQuote() && char == ")":
				addTokenAndReset()
				appendAndAdd()
				startColNumber++
				parenthesisLevel--
				continue
			}
		}

		// string & identifier quotes
		{
			if !inQuote() && char == "'" {
				inSingleQ = true
				if isWhite(token) || isOperand(token) {
					addResetAndAppend()
					continue
				}
			} else if inSingleQ && char == "'" && !escaping {
				appendAndAdd()
				inSingleQ = false
				continue
			} else if inSingleQ && char == `\` && !escaping {
				escaping = true
			} else if inSingleQ && escaping {
				escaping = false
			}

			if !inQuote() && char == `"` {
				inDoubleQ = true
				if isWhite(token) || isOperand(token) {
					addResetAndAppend()
					continue
				}
			} else if inDoubleQ && char == `"` {
				inDoubleQ = false
				appendAndAdd()
				continue
			}

			if !inQuote() && char == "`" {
				inTickQ = true
				if isWhite(token) || isOperand(token) {
					addResetAndAppend()
					continue
				}
			} else if inTickQ && char == "`" {
				inTickQ = false
				if nChar != "." {
					appendAndAdd()
					continue
				}
			}

		}

		if inQuote() {
			append()
		} else if isWhite(char) != isWhite(pChar) {
			addResetAndAppend()
		} else if isOperand(char) != isOperand(pChar) {
			addResetAndAppend()
		} else {
			append()
		}
	}

	if len(token) > 0 {
		addTokenAndReset()
	}

	return
}
