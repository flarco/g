package csv

import (
	"strings"
	"testing"
	"reflect"
)

func TestMultiCharDelimiter(t *testing.T) {
	tests := []struct {
		Name   string
		Input  string
		Comma  string
		Output [][]string
		Error  error
	}{
		{
			Name:   "SimpleMultiChar",
			Input:  "a::b::c\n1::2::3\n",
			Comma:  "::",
			Output: [][]string{{"a", "b", "c"}, {"1", "2", "3"}},
		},
		{
			Name:   "MultiCharWithQuotes",
			Input:  `"a::b"::c::d` + "\n",
			Comma:  "::",
			Output: [][]string{{"a::b", "c", "d"}},
		},
		{
			Name:   "ThreeCharDelimiter",
			Input:  "field1|||field2|||field3\nval1|||val2|||val3\n",
			Comma:  "|||",
			Output: [][]string{{"field1", "field2", "field3"}, {"val1", "val2", "val3"}},
		},
		{
			Name:   "DelimiterWithSpecialChars",
			Input:  "a<=>b<=>c\n1<=>2<=>3\n",
			Comma:  "<=>",
			Output: [][]string{{"a", "b", "c"}, {"1", "2", "3"}},
		},
		{
			Name:   "EmptyFieldsMultiChar",
			Input:  "a::::c\n",
			Comma:  "::",
			Output: [][]string{{"a", "", "c"}},
		},
		{
			Name:   "QuotedMultiCharDelimiter",
			Input:  `"field with :: delimiter"::second::third` + "\n",
			Comma:  "::",
			Output: [][]string{{"field with :: delimiter", "second", "third"}},
		},
		{
			Name:  "InvalidMultiCharWithNewline",
			Input: "test",
			Comma: ":\n",
			Error: errInvalidDelim,
		},
		{
			Name:  "InvalidMultiCharWithQuote",
			Input: "test",
			Comma: `:"`,
			Error: errInvalidDelim,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			r := NewReader(strings.NewReader(tt.Input))
			r.Comma = tt.Comma
			
			out, err := r.ReadAll()
			if err != tt.Error {
				t.Errorf("ReadAll() error = %v, want %v", err, tt.Error)
			}
			if tt.Error == nil && !reflect.DeepEqual(out, tt.Output) {
				t.Errorf("ReadAll() output = %v, want %v", out, tt.Output)
			}
		})
	}
}

func TestMultiCharDelimiterWriter(t *testing.T) {
	tests := []struct {
		Name   string
		Input  [][]string
		Comma  string
		Output string
	}{
		{
			Name:   "SimpleMultiChar",
			Input:  [][]string{{"a", "b", "c"}, {"1", "2", "3"}},
			Comma:  "::",
			Output: "a::b::c\n1::2::3\n",
		},
		{
			Name:   "MultiCharWithQuotedField",
			Input:  [][]string{{"a::b", "c", "d"}},
			Comma:  "::",
			Output: `"a::b"::c::d` + "\n",
		},
		{
			Name:   "ThreeCharDelimiter",
			Input:  [][]string{{"field1", "field2", "field3"}, {"val1", "val2", "val3"}},
			Comma:  "|||",
			Output: "field1|||field2|||field3\nval1|||val2|||val3\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			var b strings.Builder
			w := NewWriter(&b)
			w.Comma = tt.Comma
			
			err := w.WriteAll(tt.Input)
			if err != nil {
				t.Fatalf("WriteAll() error = %v", err)
			}
			
			if got := b.String(); got != tt.Output {
				t.Errorf("WriteAll() = %q, want %q", got, tt.Output)
			}
		})
	}
}