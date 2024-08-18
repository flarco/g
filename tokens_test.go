package g

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokens(t *testing.T) {

	expr := `((t = 'One There') or t = "ToT T") and (func(ok) = 3 )`
	tb := Tokenize(expr, nil)
	// Warn(Pretty(tb))
	assert.Len(t, tb.Tokens, 31)

	expr = `t in ('one', 'two')`
	tb = Tokenize(expr, nil)
	assert.Len(t, tb.Tokens, 10)

	expr = `t='val'`
	tb = Tokenize(expr, nil)
	assert.Len(t, tb.Tokens, 3)

	expr = `t ~ 'val'`
	tb = Tokenize(expr, nil)
	assert.Len(t, tb.Tokens, 5)

	expr = `t !~ "val ff"`
	tb = Tokenize(expr, nil)
	assert.Len(t, tb.Tokens, 5)

	expr = `a(v-  b ) <= t'`
	tb = Tokenize(expr, nil)
	assert.Len(t, tb.Tokens, 12)
}

func TestTokenMethods(t *testing.T) {
	input := "SELECT id, name FROM users WHERE id > 5"
	body := Tokenize(input, nil)

	t.Run("IsEmpty", func(t *testing.T) {
		assert.False(t, body.Tokens[0].IsEmpty(), "First token should not be empty")
		assert.True(t, Token{}.IsEmpty(), "Empty token should be empty")
	})

	t.Run("IsWhitespace", func(t *testing.T) {
		assert.False(t, body.Tokens[0].IsWhitespace(), "SELECT is not whitespace")
		assert.True(t, body.Tokens[1].IsWhitespace(), "Space after SELECT is whitespace")
	})

	t.Run("IsWord", func(t *testing.T) {
		assert.True(t, body.Tokens[0].IsWord(), "SELECT is a word")
		assert.False(t, body.Tokens[1].IsWord(), "Space is not a word")
	})

	t.Run("IsOperand", func(t *testing.T) {
		assert.False(t, body.Tokens[0].IsOperand(), "SELECT is not an operand")
		assert.True(t, body.Tokens[15].IsOperand(), "%s is an operand", body.Tokens[15].Text)
	})

	t.Run("IsAlphaNumericID", func(t *testing.T) {
		assert.True(t, body.Tokens[0].IsAlphaNumericID(), "SELECT is alphanumeric")
		assert.False(t, body.Tokens[1].IsAlphaNumericID(), "Space is not alphanumeric")
	})

	t.Run("Previous and Next", func(t *testing.T) {
		selectToken := body.Tokens[0]
		assert.Equal(t, "SELECT", selectToken.Text)

		nextToken := selectToken.Next()
		assert.Equal(t, " ", nextToken.Text)

		previousToken := nextToken.Previous()
		assert.Equal(t, "SELECT", previousToken.Text)
	})

	t.Run("FindNext", func(t *testing.T) {
		selectToken := body.Tokens[0]
		fromToken := selectToken.FindNext("FROM")
		assert.Equal(t, "FROM", fromToken.Text)
	})

	t.Run("NextNonWhitespace", func(t *testing.T) {
		selectToken := body.Tokens[0]
		idToken := selectToken.NextNonWhitespace()
		assert.Equal(t, "id", idToken.Text)
	})

	t.Run("TrimWhiteSpace", func(t *testing.T) {
		trimmed := body.Tokens.TrimWhiteSpace()
		assert.Equal(t, "SELECT", trimmed[0].Text)
		assert.Equal(t, "5", trimmed[len(trimmed)-1].Text)
	})
}
