package csv

import (
	"fmt"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/flarco/g"
	"github.com/stretchr/testify/assert"
)

func TestCsvReader1(t *testing.T) {
	in := `first_name,last_name,username
"Rob","Pike",rob
Ken,Thompson,ken
"Robert","Griesemer","gri"
"Another" , Person , "aPerson"""
`
	c := NewCsv()

	r := c.NewReader(strings.NewReader(in))

	rows := [][]string{}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		rows = append(rows, record)
		fmt.Println(g.Marshal(record))
		println()
	}

	if assert.Equal(t, 4, len(rows)) {
		assert.Equal(t, `["first_name","last_name","username"]`, g.Marshal(rows[0]))
		assert.Equal(t, `["Rob","Pike","rob"]`, g.Marshal(rows[1]))
		assert.Equal(t, `["Ken","Thompson","ken"]`, g.Marshal(rows[2]))
		assert.Equal(t, `["Robert","Griesemer","gri"]`, g.Marshal(rows[3]))
	}

}

func TestCsvReader2(t *testing.T) {
	in := `col1|header|col3
column1_data|"column2 data with \"space\""|column3_data
"column1_data"|"column2 data with \"space\""|column3_data`

	c := NewCsv(CsvOptions{Delimiter: '|', Escape: '\\'})

	r := c.NewReader(strings.NewReader(in))

	rows := [][]string{}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		rows = append(rows, record)
		fmt.Println(g.Marshal(record))
		println()
	}

	if assert.Equal(t, 3, len(rows)) {
		assert.Equal(t, `["col1","header","col3"]`, g.Marshal(rows[0]))
		assert.Equal(t, `["column1_data","column2 data with \"space\"","column3_data"]`, g.Marshal(rows[1]))
		assert.Equal(t, `["column1_data","column2 data with \"space\"","column3_data"]`, g.Marshal(rows[2]))
	}

}

func TestCsvReader3(t *testing.T) {
	in := `id,first_name,last_name,email,target,create_dt,rating,code
1,"AOCG,""
883",Lumox,ilumox0@unc.edu,true,2019-02-11 11:23:00.000
2,Terrell,"EKOZ,989",tmee1@example.com,true,2019-08-19 17:02:09.000,89.983,2
`

	c := NewCsv(CsvOptions{Delimiter: ',', Escape: '"'})

	r := c.NewReader(strings.NewReader(in))

	rows := [][]string{}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		rows = append(rows, record)

		fmt.Println(g.Marshal(record))
		println()
	}

	if assert.Equal(t, 3, len(rows)) {
		assert.Equal(t, `["id","first_name","last_name","email","target","create_dt","rating","code"]`, g.Marshal(rows[0]))
		assert.Equal(t, `["1","AOCG,\"\n883","Lumox","ilumox0@unc.edu","true","2019-02-11 11:23:00.000"]`, g.Marshal(rows[1]))
		assert.Equal(t, `["2","Terrell","EKOZ,989","tmee1@example.com","true","2019-08-19 17:02:09.000","89.983","2"]`, g.Marshal(rows[2]))
	}

}

// benchmarkReadNew measures reading the provided CSV rows data.
// initReader, if non-nil, modifies the Reader before it's used.
func benchmarkReadNew(b *testing.B, opts CsvOptions, rows string) {
	b.ReportAllocs()
	r := NewCsv(opts).NewReader(&nTimes{s: rows, n: b.N})

	for {
		_, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadNew(b *testing.B) {
	benchmarkReadNew(b, CsvOptions{}, benchmarkCSVData)
}

func BenchmarkReadNewWithFieldsPerRecord(b *testing.B) {
	benchmarkReadNew(b, CsvOptions{}, benchmarkCSVData)
}

func BenchmarkReadNewWithoutFieldsPerRecord(b *testing.B) {
	benchmarkReadNew(b, CsvOptions{}, benchmarkCSVData)
}

func BenchmarkReadNewLargeFields(b *testing.B) {
	benchmarkReadNew(b, CsvOptions{}, strings.Repeat(`xxxxxxxxxxxxxxxx,yyyyyyyyyyyyyyyy,zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv
xxxxxxxxxxxxxxxxxxxxxxxx,yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy,zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvv
,,zzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx,yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy,zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv
`, 3))
}

func BenchmarkReadNewReuseRecord(b *testing.B) {
	benchmarkReadNew(b, CsvOptions{}, benchmarkCSVData)
}

func BenchmarkReadNewReuseRecordWithFieldsPerRecord(b *testing.B) {
	benchmarkReadNew(b, CsvOptions{}, benchmarkCSVData)
}

func BenchmarkReadNewReuseRecordWithoutFieldsPerRecord(b *testing.B) {
	benchmarkReadNew(b, CsvOptions{}, benchmarkCSVData)
}

func BenchmarkReadNewReuseRecordLargeFields(b *testing.B) {
	benchmarkReadNew(b, CsvOptions{}, strings.Repeat(`xxxxxxxxxxxxxxxx,yyyyyyyyyyyyyyyy,zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv
xxxxxxxxxxxxxxxxxxxxxxxx,yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy,zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvv
,,zzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx,yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy,zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz,wwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww,vvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvvv
`, 3))
}
