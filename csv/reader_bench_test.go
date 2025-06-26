package csv

import (
	"bytes"
	"strings"
	"testing"
)

var benchmarkData = []struct {
	name string
	data string
}{
	{"Simple", "field1,field2,field3\nvalue1,value2,value3\n"},
	{"Quoted", `"field1","field2","field3"` + "\n" + `"value1","value2","value3"` + "\n"},
	{"QuotedWithCommas", `"field,1","field,2","field,3"` + "\n" + `"value,1","value,2","value,3"` + "\n"},
	{"LargeFields", strings.Repeat("a", 100) + "," + strings.Repeat("b", 100) + "," + strings.Repeat("c", 100) + "\n"},
	{"ManyFields", strings.Repeat("field,", 50) + "field\n" + strings.Repeat("value,", 50) + "value\n"},
	{"ManyRows", func() string {
		var sb strings.Builder
		for i := 0; i < 100; i++ {
			sb.WriteString("field1,field2,field3\n")
		}
		return sb.String()
	}(),
	},
}

func BenchmarkReader(b *testing.B) {
	for _, bm := range benchmarkData {
		b.Run(bm.name, func(b *testing.B) {
			data := []byte(bm.data)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				r := NewReader(bytes.NewReader(data))
				_, err := r.ReadAll()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkReaderSingleComma(b *testing.B) {
	data := []byte(strings.Repeat("a,b,c,d,e,f,g,h,i,j\n", 1000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewReader(bytes.NewReader(data))
		_, err := r.ReadAll()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReaderCustomDelimiter(b *testing.B) {
	data := []byte(strings.Repeat("a|b|c|d|e|f|g|h|i|j\n", 1000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewReader(bytes.NewReader(data))
		r.Comma = "|"
		_, err := r.ReadAll()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReaderQuotedFields(b *testing.B) {
	data := []byte(strings.Repeat(`"a","b","c","d","e","f","g","h","i","j"`+"\n", 1000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewReader(bytes.NewReader(data))
		_, err := r.ReadAll()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReaderMixedQuoted(b *testing.B) {
	data := []byte(strings.Repeat(`a,"b,c",d,"e,f","g","h,i,j",k,l,m,n`+"\n", 1000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewReader(bytes.NewReader(data))
		_, err := r.ReadAll()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReaderMultiCharDelimiter(b *testing.B) {
	data := []byte(strings.Repeat("a::b::c::d::e::f::g::h::i::j\n", 1000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewReader(bytes.NewReader(data))
		r.Comma = "::"
		_, err := r.ReadAll()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReaderThreeCharDelimiter(b *testing.B) {
	data := []byte(strings.Repeat("a|||b|||c|||d|||e|||f|||g|||h|||i|||j\n", 1000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewReader(bytes.NewReader(data))
		r.Comma = "|||"
		_, err := r.ReadAll()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReaderMultiCharQuoted(b *testing.B) {
	data := []byte(strings.Repeat(`"a"::"b"::"c"::"d"::"e"::"f"::"g"::"h"::"i"::"j"`+"\n", 1000))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := NewReader(bytes.NewReader(data))
		r.Comma = "::"
		_, err := r.ReadAll()
		if err != nil {
			b.Fatal(err)
		}
	}
}