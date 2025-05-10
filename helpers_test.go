package g

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type cfgTest struct {
	Prop1 string `json:"prop1"`
}

func TestEncryptDecryptInBytes(t *testing.T) {
	message := "my message :// !"
	ct1 := cfgTest{message}
	ct2 := cfgTest{}
	passphrase := "heloaf aa gag"

	ct1.Prop1 = string(EncryptInBytes([]byte(ct1.Prop1), passphrase))
	ct2.Prop1 = string(DecryptInByte([]byte(ct1.Prop1), passphrase))

	assert.Equal(t, message, ct2.Prop1)
}

func TestEncryptDecrypt(t *testing.T) {
	var err error
	message := "my message :// !"
	ct1 := cfgTest{message}
	ct2 := cfgTest{}
	// the key should be 16, 24 or 32 byte
	key := RandString(TokenRunes, 32)
	println(key)

	ct1.Prop1, err = Encrypt(ct1.Prop1, key)
	assert.NoError(t, err)
	cfgBytes, err := json.Marshal(ct1)
	assert.NoError(t, err)

	err = json.Unmarshal(cfgBytes, &ct2)
	assert.NoError(t, err)

	assert.Equal(t, ct1.Prop1, ct2.Prop1)

	ct2.Prop1, err = Decrypt(ct2.Prop1, key)
	assert.NoError(t, err)

	assert.Equal(t, message, ct2.Prop1)
}

func TestLogging(t *testing.T) {
	// log.Info().Msg("hello world")
	// err := Error(fmt.Errorf("new error"), "This occurred just cause")
	// LogError(err)
	// LogError(fmt.Errorf("new error"), "This occurred just cause")
	// log.Err(err).Msg("This occurred just cause")
	mapInterf := map[string]interface{}{
		"output_type": "hello",
		"job_id":      1111,
	}
	println(fmt.Sprintf("%T", mapInterf))
	Debug("hello world %s %s", "fritz", mapInterf, "larco")

	mapInterf = M("job_id", 555, "output_type", "goodbye")

	Log("hello world", mapInterf)

	mapInterf = M("job_id", 555, "output_type")
	Debug("hello world", mapInterf)
	Debug("hello world", mapInterf)
	Trace("hello world")
	Warn("number of cpus %d", runtime.NumCPU())
	Warn("number of cpus %d", runtime.NumCPU())
	LogError(fmt.Errorf("new error"), M("user_name", "fritz"))
	// Error(nil)
	// LogFatal(Error(fmt.Errorf("new error")), "hello")

}

func TestMap(t *testing.T) {
	m := map[string]int64{"one": 2, "two": 121}
	println(m["hello"]) // should be 0
	assert.EqualValues(t, 121, AsMap(m)["two"])

	m1 := map[int]float32{1: 1, 2: 2.2}
	assert.EqualValues(t, m1[2], AsMap(m1)["2"])
}

func TestGenToken(t *testing.T) {
	token := RandString(AlphaNumericRunes, 32)
	hash, _ := Hash(token + token)
	println(token)
	println(hash)
}

func TestRand(t *testing.T) {
	val := RandInt(30)
	assert.Greater(t, val, 0)

	d := time.Duration(val) * time.Minute
	assert.EqualValues(t, val, d.Minutes())

}

func TestHash(t *testing.T) {
	hash, err := Hash("hello")
	assert.Nil(t, err)
	assert.NotEmpty(t, hash)
	println(hash)
}

func TestCompareVersion(t *testing.T) {
	isNew, err := CompareVersions("v0.0.5", "v0.0.40")
	assert.True(t, isNew)
	assert.NoError(t, err)
	isNew, err = CompareVersions("v0.0.5", "v0.0.12")
	assert.True(t, isNew)
	assert.NoError(t, err)
	isNew, err = CompareVersions("v1.0.5", "v0.0.40")
	assert.False(t, isNew)
	assert.NoError(t, err)
	isNew, err = CompareVersions("v1.0.5", "v0.9.40")
	assert.False(t, isNew)
	assert.NoError(t, err)
	_, err = CompareVersions("v0.0f.5", "v0.0.40")
	assert.Error(t, err)
}

func TestVerify(t *testing.T) {
	hash, err := Hash("hello")
	assert.Nil(t, err)
	assert.NotEmpty(t, hash)

	ok, err := VerifyHash("hello", hash)
	assert.Nil(t, err)
	assert.True(t, ok)
}
func TestError(t *testing.T) {
	err := fmt.Errorf("my new error first")
	println(err.Error())
	println(ErrMsg(err))
	println("0 -----------------------")
	err1 := Error(err)
	println(err1.Error())
	println(ErrMsg(err1))
	println("1 -----------------------")
	err2 := Error(err, "additional\ndetails %d", 4)
	P(ArgsErrMsg("additional\ndetails"))
	println(err2.Error())
	println(ErrMsg(err2))
	println("2 -----------------------")
	err2a := Error("additional\ndetails %d", 3)
	println(err2a.Error())
	println("3 -----------------------")
	err2b := Error(err2a, "moredetails %d", 5)
	println(err2b.Error())
	println("4 -----------------------")
	err3 := Error(err2, "additional details on top")
	LogError(err3)
	Warn(ErrMsgSimple(err2))
	println("5 -----------------------")
	PrintFatal(err3)
}

type wrapError struct {
	msg string
	err error
}

func (e *wrapError) Error() string {
	return e.msg
}

func (e *wrapError) Unwrap() error {
	return e.err
}

func TestError2(t *testing.T) {
	e0 := fmt.Errorf("my error failure")
	e1 := Error(e0, "level 1")
	e2 := Error(e1, "level 2")
	println(e1.Error())
	println()
	println(e2.Error())

	et1 := e1.(*ErrType)
	et2 := e2.(*ErrType)
	P(et1)
	P(et2)
	assert.Equal(t, et1.Err, et2.Err)
	println(et2.Debug())
}
func TestError3(t *testing.T) {
	e1 := Error("level 1")
	e2 := Error("level %d (%s)", 2, e1.Error())
	println(e1.Error())
	println()
	println(e2.Error())
}

func TestExists(t *testing.T) {
	assert.True(t, PathExists("/root"))
	assert.False(t, PathExists("/roodadat"))
}

func TestMatches(t *testing.T) {
	m := Matches("oracle://{username}:{password}@{tns}", "{([a-zA-Z]+)}")
	assert.Len(t, m, 3)

	g := MatchesGroup("oracle://{username}:{password}@{tns}", "{([a-zA-Z]+)}", 0)
	assert.Equal(t, g[2], "tns")
}

// go test -bench=. -benchmem -run '^BenchmarkIsIdentifier'
// go test -bench=. -run BenchmarkIsIdentifier
func BenchmarkToMap(b *testing.B) {
	m := map[string]interface{}{"1": 1}
	for i := 0; i < b.N; i++ {
		ToMap(m)
	}
}

// go test -bench=. -run '^Benchmark'
func BenchmarkAsMap(b *testing.B) {
	m := map[string]interface{}{"1": 1}
	for i := 0; i < b.N; i++ {
		AsMap(m)
	}
}

// go test -bench=. -run '^BenchmarkMux'
func BenchmarkMux(b *testing.B) {
	c := NewContext(context.Background())
	for i := 0; i < b.N; i++ {
		c.Lock()
		c.Unlock()
	}
}

func TestLoggerColor(t *testing.T) {
	println(Colorize(ColorRed, "colorRed"))
	println(Colorize(ColorGreen, "colorGreen"))
	println(Colorize(ColorYellow, "colorYellow"))
	println(Colorize(ColorBlue, "colorBlue"))
	println(Colorize(ColorMagenta, "colorMagenta"))
	println(Colorize(ColorCyan, "colorCyan"))
	println(Colorize(ColorWhite, "colorWhite"))
	println(Colorize(ColorBold, "ColorBold"))
	println(Colorize(ColorDarkGray, "ColorDarkGray"))
}

func TestMarshalOrdered(t *testing.T) {
	// Test with non-map values
	str, err := MarshalOrdered("hello")
	assert.NoError(t, err)
	assert.Equal(t, `"hello"`, str)

	num, err := MarshalOrdered(42)
	assert.NoError(t, err)
	assert.Equal(t, `42`, num)

	// Test with simple maps in different orders
	map1 := map[string]interface{}{
		"a": 1,
		"c": 3,
		"b": 2,
	}

	map2 := map[string]interface{}{
		"c": 3,
		"b": 2,
		"a": 1,
	}

	str1, err := MarshalOrdered(map1)
	assert.NoError(t, err)
	str2, err := MarshalOrdered(map2)
	assert.NoError(t, err)
	assert.Equal(t, str1, str2)
	assert.Equal(t, `{"a":1,"b":2,"c":3}`, str1)

	// Test with nested maps
	nestedMap1 := map[string]interface{}{
		"x": map[string]interface{}{
			"z": 1,
			"y": 2,
		},
		"a": 3,
	}

	nestedMap2 := map[string]interface{}{
		"a": 3,
		"x": map[string]interface{}{
			"y": 2,
			"z": 1,
		},
	}

	nestedStr1, err := MarshalOrdered(nestedMap1)
	assert.NoError(t, err)
	nestedStr2, err := MarshalOrdered(nestedMap2)
	assert.NoError(t, err)
	assert.Equal(t, nestedStr1, nestedStr2)
	assert.Equal(t, `{"a":3,"x":{"y":2,"z":1}}`, nestedStr1)

	// Test with array containing maps
	arrMap1 := map[string]interface{}{
		"arr": []interface{}{
			map[string]interface{}{"b": 2, "a": 1},
			map[string]interface{}{"d": 4, "c": 3},
		},
	}

	arrMap2 := map[string]interface{}{
		"arr": []interface{}{
			map[string]interface{}{"a": 1, "b": 2},
			map[string]interface{}{"c": 3, "d": 4},
		},
	}

	arrStr1, err := MarshalOrdered(arrMap1)
	assert.NoError(t, err)
	arrStr2, err := MarshalOrdered(arrMap2)
	assert.NoError(t, err)
	assert.Equal(t, arrStr1, arrStr2)
	assert.Equal(t, `{"arr":[{"a":1,"b":2},{"c":3,"d":4}]}`, arrStr1)

	// Test with custom Map type
	customMap1 := Map{
		"foo": "bar",
		"baz": 123,
	}

	customMap2 := Map{
		"baz": 123,
		"foo": "bar",
	}

	customStr1, err := MarshalOrdered(customMap1)
	assert.NoError(t, err)
	customStr2, err := MarshalOrdered(customMap2)
	assert.NoError(t, err)
	assert.Equal(t, customStr1, customStr2)
	assert.Equal(t, `{"baz":123,"foo":"bar"}`, customStr1)

	// Test deterministic ordering from JSON payload
	jsonPayload := `{
		"metadata": {
			"tags": ["tag3", "tag1", "tag2"],
			"created": "2023-06-01",
			"nested": {
				"z": 3,
				"y": 2,
				"x": 1,
				"array": [
					{"c": 3, "a": 1, "b": 2},
					{"f": 6, "e": 5, "d": 4}
				]
			}
		},
		"config": {
			"timeout": 500,
			"enabled": true
		},
		"id": 42,
		"values": [5, 4, 3, 2, 1]
	}`

	// Create a baseline ordered string
	var baseMap map[string]interface{}
	err = json.Unmarshal([]byte(jsonPayload), &baseMap)
	assert.NoError(t, err)
	baseOrdered, err := MarshalOrdered(baseMap)
	assert.NoError(t, err)

	// Verify the exact expected value with ordered keys
	expectedJSON := `{"config":{"enabled":true,"timeout":500},"id":42,"metadata":{"created":"2023-06-01","nested":{"array":[{"a":1,"b":2,"c":3},{"d":4,"e":5,"f":6}],"x":1,"y":2,"z":3},"tags":["tag3","tag1","tag2"]},"values":[5,4,3,2,1]}`
	assert.Equal(t, expectedJSON, baseOrdered, "The ordered JSON doesn't match the expected value")

	// Test multiple iterations with different unmarshaling order to ensure determinism
	for i := 0; i < 100; i++ {
		var iterMap map[string]interface{}
		err = json.Unmarshal([]byte(jsonPayload), &iterMap)
		assert.NoError(t, err)

		// Simulate different insertion order by manipulating the map
		// (Go maps have random iteration order)
		tempMap := map[string]interface{}{}
		for k, v := range iterMap {
			tempMap[k] = v
		}

		iterOrdered, err := MarshalOrdered(tempMap)
		assert.NoError(t, err)
		assert.Equal(t, baseOrdered, iterOrdered, "Iteration %d failed to produce deterministic output", i)
	}
}

// BenchmarkMarshalOrdered benchmarks the MarshalOrdered function
// go test -bench=BenchmarkMarshalOrdered -benchmem
// go test -bench=BenchmarkMarshalOrdered -benchmem -benchtime=5s
func BenchmarkMarshalOrdered(b *testing.B) {
	// Small flat map
	smallMap := map[string]interface{}{
		"id":     1,
		"name":   "item",
		"value":  42.5,
		"active": true,
	}

	// Medium map with some nesting
	mediumMap := map[string]interface{}{
		"id": 1,
		"metadata": map[string]interface{}{
			"created": "2023-06-01",
			"updated": "2023-06-02",
			"tags":    []string{"tag1", "tag2", "tag3"},
		},
		"values": []int{1, 2, 3, 4, 5},
		"config": map[string]interface{}{
			"enabled": true,
			"timeout": 500,
		},
	}

	// Large complex map with deeper nesting
	largeMap := map[string]interface{}{
		"id": "abc123",
		"user": map[string]interface{}{
			"id":    42,
			"name":  "John Doe",
			"email": "john@example.com",
			"preferences": map[string]interface{}{
				"theme":         "dark",
				"notifications": true,
				"language":      "en",
			},
		},
		"items": []interface{}{
			map[string]interface{}{"id": 1, "name": "Item 1"},
			map[string]interface{}{"id": 2, "name": "Item 2"},
			map[string]interface{}{"id": 3, "name": "Item 3", "metadata": map[string]interface{}{
				"color": "blue",
				"size":  "medium",
			}},
		},
		"settings": map[string]interface{}{
			"display": map[string]interface{}{
				"width":  1920,
				"height": 1080,
				"mode":   "fullscreen",
			},
			"audio": map[string]interface{}{
				"volume":  75,
				"muted":   false,
				"devices": []string{"speaker", "headphones"},
			},
			"network": map[string]interface{}{
				"proxy": map[string]interface{}{
					"enabled": false,
					"address": "proxy.example.com",
					"port":    8080,
				},
			},
		},
	}

	b.Run("SmallMap-Standard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(smallMap)
		}
	})

	b.Run("SmallMap-Ordered", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = MarshalOrdered(smallMap)
		}
	})

	b.Run("MediumMap-Standard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(mediumMap)
		}
	})

	b.Run("MediumMap-Ordered", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = MarshalOrdered(mediumMap)
		}
	})

	b.Run("LargeMap-Standard", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(largeMap)
		}
	})

	b.Run("LargeMap-Ordered", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = MarshalOrdered(largeMap)
		}
	})
}
