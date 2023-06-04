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
	testLogHook := func(t string, a ...interface{}) {
		println("hook run for -> " + F(t, a...))
	}
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
	SetLogHook(NewLogHook(DebugLevel, testLogHook))
	Warn("number of cpus %d", runtime.NumCPU())
	LogError(fmt.Errorf("new error"), "hello")
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
	token := RandString(AplhanumericRunes, 32)
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
	println("-----------------------")
	err1 := Error(err)
	println(err1.Error())
	println(ErrMsg(err1))
	println("-----------------------")
	err2 := Error(err, "additional\ndetails %d", 4)
	P(ArgsErrMsg("additional\ndetails"))
	println(err2.Error())
	println(ErrMsg(err2))
	println("-----------------------")
	err2a := Error("additional\ndetails %d", 3)
	println(err2a.Error())
	println("-----------------------")
	err2b := Error(err2a, "moredetails %d", 5)
	println(err2b.Error())
	println("-----------------------")
	err3 := Error(err2, "additional details on top")
	LogFatal(err3)
	println("-----------------------")
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
