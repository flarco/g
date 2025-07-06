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

func TestReplace(t *testing.T) {
	// Basic replacement
	result := R("Hello {name}, you are {age} years old {other}", "name", "John", "age", "30")
	assert.Equal(t, "Hello John, you are 30 years old {other}", result)

	// Replacement with spaces around keys
	result = R("Hello { name }, you are {  age  } years old", "name", "John", "age", "30")
	assert.Equal(t, "Hello John, you are 30 years old", result)

	// Mixed spacing
	result = R("File {file} had error { error }", "file", "test.txt", "error", "not found")
	assert.Equal(t, "File test.txt had error not found", result)

	// Multiple occurrences of same key
	result = R("User {user} logged in at {time}. User {user} has permissions.", "user", "admin", "time", "10:30")
	assert.Equal(t, "User admin logged in at 10:30. User admin has permissions.", result)

	// No matches
	result = R("No placeholders here", "name", "John", "age", "30")
	assert.Equal(t, "No placeholders here", result)
	result = R("${c}", "c", "1")
	assert.Equal(t, "$1", result)

	// Empty format string
	result = R("", "name", "John")
	assert.Equal(t, "", result)

	// Empty args
	result = R("Hello {name}")
	assert.Equal(t, "Hello {name}", result)

	// Odd number of args (should handle gracefully)
	result = R("Hello {name}", "name", "John", "age")
	assert.Equal(t, "Hello John", result)

	// Special characters in keys
	result = R("Value is {my-key}", "my-key", "test")
	assert.Equal(t, "Value is test", result)

	// Keys with dots
	result = R("Config {app.name} version {app.version }", "app.name", "MyApp", "app.version", "1.0")
	assert.Equal(t, "Config MyApp version 1.0", result)

	// Different types of whitespace
	result = R("Hello {\tname\n} and {  age  }", "name", "John", "age", "30")
	assert.Equal(t, "Hello John and 30", result)

	// Keys that don't exist
	result = R("Hello {name}", "user", "John")
	assert.Equal(t, "Hello {name}", result)

	// Empty key
	result = R("Hello {}", "", "John")
	assert.Equal(t, "Hello John", result)

	// Nested braces (will match inner pattern)
	result = R("Hello {{name}}", "name", "John")
	assert.Equal(t, "Hello {John}", result)

	// Single brace (should not match)
	result = R("Hello {name and age}", "name", "John")
	assert.Equal(t, "Hello {name and age}", result)
}

func TestReplacem(t *testing.T) {
	// Basic replacement
	m := map[string]any{"name": "John", "age": 30}
	result := Rm("Hello {name}, you are {age} years old", m)
	assert.Equal(t, "Hello John, you are 30 years old", result)

	// Replacement with spaces around keys
	result = Rm("Hello { name }, you are {  age  } years old { other}", m)
	assert.Equal(t, "Hello John, you are 30 years old { other}", result)

	// Mixed spacing
	m2 := map[string]any{"file": "test.txt", "error": "not found"}
	result = Rm("File {file} had error { error }", m2)
	assert.Equal(t, "File test.txt had error not found", result)

	// Multiple occurrences of same key
	m3 := map[string]any{"user": "admin", "time": "10:30"}
	result = Rm("User {user} logged in at {time}. User {user} has permissions.", m3)
	assert.Equal(t, "User admin logged in at 10:30. User admin has permissions.", result)

	// No matches
	result = Rm("No placeholders here", m)
	assert.Equal(t, "No placeholders here", result)

	// Empty format string
	result = Rm("", m)
	assert.Equal(t, "", result)

	// Empty map
	result = Rm("Hello {name}", map[string]any{})
	assert.Equal(t, "Hello {name}", result)

	// Nil map
	result = Rm("Hello {name}", nil)
	assert.Equal(t, "Hello {name}", result)

	// Different value types
	m4 := map[string]any{
		"str":   "hello",
		"int":   42,
		"float": 3.14,
		"bool":  true,
		"nil":   nil,
		"slice": []int{1, 2, 3},
		"map":   map[string]int{"a": 1},
	}
	result = Rm("str:{str} int:{int} float:{float} bool:{bool} nil:{nil} slice:{slice} map:{map}", m4)
	expected := "str:hello int:42 float:3.14 bool:true nil: slice:[1,2,3] map:{\"a\":1}"
	assert.Equal(t, expected, result)

	// Special characters in keys
	m5 := map[string]any{"my-key": "test", "app.name": "MyApp"}
	result = Rm("Value is {my-key} and app is {app.name}", m5)
	assert.Equal(t, "Value is test and app is MyApp", result)

	// Different types of whitespace
	result = Rm("Hello {\tname\n} and {  age  }", m)
	assert.Equal(t, "Hello John and 30", result)

	// Keys that don't exist in map
	result = Rm("Hello {name} and {missing}", map[string]any{"name": "John"})
	assert.Equal(t, "Hello John and {missing}", result)

	// Empty key
	result = Rm("Hello {}", map[string]any{"": "World"})
	assert.Equal(t, "Hello World", result)

	// Nested braces (will match inner pattern)
	result = Rm("Hello {{name}}", m)
	assert.Equal(t, "Hello {John}", result)

	// Single brace (should not match)
	result = Rm("Hello {name and age}", m)
	assert.Equal(t, "Hello {name and age}", result)

	// Complex object that needs JSON marshaling
	complexObj := struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{ID: 1, Name: "test"}
	m6 := map[string]any{"object": complexObj}
	result = Rm("Object: {object}", m6)
	assert.Contains(t, result, `"id":1`)
	assert.Contains(t, result, `"name":"test"`)

	// Test with cast.ToStringE failure scenario (should fall back to Marshal)
	m7 := map[string]any{"complex": make(chan int)} // channels can't be converted to string
	result = Rm("Value: {complex}", m7)
	assert.Contains(t, result, "Value: ") // channels marshal to empty/null in JSON
}

func TestReplaceEdgeCases(t *testing.T) {
	// Test regex special characters in keys
	result := R("Value {key[0]} and {key+1}", "key[0]", "first", "key+1", "second")
	assert.Equal(t, "Value first and second", result)

	// Test with regex metacharacters
	result = R("Pattern {.*} and {^start$}", ".*", "wildcard", "^start$", "anchor")
	assert.Equal(t, "Pattern wildcard and anchor", result)

	// Test parentheses in keys
	result = R("Function {func()} called", "func()", "myFunction")
	assert.Equal(t, "Function myFunction called", result)
}

func TestReplacemEdgeCases(t *testing.T) {
	// Test regex special characters in keys
	m := map[string]any{"key[0]": "first", "key+1": "second"}
	result := Rm("Value {key[0]} and {key+1}", m)
	assert.Equal(t, "Value first and second", result)

	// Test with regex metacharacters
	m2 := map[string]any{".*": "wildcard", "^start$": "anchor"}
	result = Rm("Pattern {.*} and {^start$}", m2)
	assert.Equal(t, "Pattern wildcard and anchor", result)

	// Test parentheses in keys
	m3 := map[string]any{"func()": "myFunction"}
	result = Rm("Function {func()} called", m3)
	assert.Equal(t, "Function myFunction called", result)

	// Test backslashes in keys
	m4 := map[string]any{"path\\file": "test.txt"}
	result = Rm("File at {path\\file}", m4)
	assert.Equal(t, "File at test.txt", result)
}

func TestReplaceAndRmCompatibility(t *testing.T) {
	// Test that R and Rm produce the same results for equivalent inputs
	format := "Hello {name}, you are {age} years old"

	// Using R
	resultR := R(format, "name", "John", "age", "30")

	// Using Rm
	m := map[string]any{"name": "John", "age": "30"}
	resultRm := Rm(format, m)

	assert.Equal(t, resultR, resultRm)

	// Test with spacing
	formatWithSpaces := "Hello { name }, you are {  age  } years old"
	resultRSpaced := R(formatWithSpaces, "name", "John", "age", "30")
	resultRmSpaced := Rm(formatWithSpaces, m)

	assert.Equal(t, resultRSpaced, resultRmSpaced)
	assert.Equal(t, "Hello John, you are 30 years old", resultRSpaced)
}
