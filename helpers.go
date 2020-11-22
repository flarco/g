package g

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
)

const (
	// AlphaRunes are alphabetic chars
	AlphaRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// AlphaRunesLower are lowercase alphabetic chars
	AlphaRunesLower = "abcdefghijklmnopqrstuvwxyz"
	// AlphaRunesUpper are uppercase alphabetic chars
	AlphaRunesUpper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// NumericRunes are numeric chars
	NumericRunes = "1234567890"
	// AplhanumericRunes are alphanumeric chars
	AplhanumericRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	// TokenRunes are alphanumeric+ chars for tokens
	TokenRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890_."
)

type (
	// Map is map[string]interface{}
	Map map[string]interface{}
)

// IsTask returns true is is TASK
func IsTask() bool {
	return os.Getenv("G_LOGGING") == "TASK"
}

// GetType : return the type of an interface
func GetType(myvar interface{}) string {
	t := reflect.TypeOf(myvar)
	if t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	}
	return t.Name()
}

// F : fmt.Sprintf
func F(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// M : return map[string]interface from args
func M(args ...interface{}) map[string]interface{} {
	mapInterf := map[string]interface{}{}
	key := ""
	for i, val := range args {
		if i%2 == 0 {
			key = cast.ToString(val)
		} else {
			mapInterf[key] = val
		}
	}
	return mapInterf
}

// ArrMapInt create a map from an array of integers
func ArrMapInt(args []int) map[int]interface{} {
	mapInterf := map[int]interface{}{}
	for _, val := range args {
		mapInterf[val] = val
	}
	return mapInterf
}

// ArrMapString create a map from an array of strings
func ArrMapString(args []string, toLower ...bool) map[string]interface{} {
	lower := false
	if len(toLower) > 0 && toLower[0] {
		lower = true
	}

	mapInterf := map[string]interface{}{}
	for _, val := range args {
		if lower {
			mapInterf[strings.ToLower(val)] = val
		} else {
			mapInterf[val] = val
		}
	}
	return mapInterf
}

// mapMerge copies key values from `mS` into `mT`
func mapMerge(mT map[string]interface{}, mS map[string]interface{}) map[string]interface{} {
	for key, val := range mS {
		mT[key] = val
	}
	return mT
}

// R : Replacer
// R("File {file} had error {error}", "file", file, "error", err)
func R(format string, args ...string) string {
	args2 := make([]string, len(args))
	for i, v := range args {
		if i%2 == 0 {
			args2[i] = fmt.Sprintf("{%v}", v)
		} else {
			args2[i] = fmt.Sprint(v)
		}
	}
	r := strings.NewReplacer(args2...)
	return r.Replace(format)
}

// Rm is like R, for replacing with a map
func Rm(format string, m map[string]interface{}) string {
	if m == nil {
		return format
	}

	args, i := make([]string, len(m)*2), 0
	for k, v := range m {
		args[i] = "{" + k + "}"
		args[i+1] = cast.ToString(v)
		i += 2
	}
	return strings.NewReplacer(args...).Replace(format)
}

// PrintT prints the type of object
func PrintT(v interface{}) {
	if IsDebugLow() {
		args := addCaller([]interface{}{})
		doLog(LogErr.Debug(), F("%T", v), args)
	}
}

// PrintRows prints the rows of object
func PrintRows(rows [][]interface{}) {
	for _, row := range rows {
		P(row)
	}
}

// Now : Get unix epoch time in milli
func Now() int64 {
	return int64(time.Now().UnixNano() / 1000000)
}

// NowFileStr : Get millisecond time in file string format
func NowFileStr() string {
	return time.Now().Format("2006-01-02T150405.000")
}

func uintStr(val string) uint {
	val64, _ := strconv.ParseUint(val, 10, 32)
	return uint(val64)
}

// Tee prints stream of text of reader
func Tee(reader io.Reader, limit int) io.Reader {
	pipeR, pipeW := io.Pipe()

	cnt := 0
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			cnt++
			if cnt > limit {
				break
			}
			bytes := scanner.Bytes()
			nl := []byte("\n")
			fmt.Println(string(bytes))
			pipeW.Write(append(bytes, nl...))
		}
		pipeW.Close()
	}()

	return pipeR
}

// KVArrToMap parse a Key-Value array in the form of
// `"Prop1=Value1", "Prop2=Value2", ...`
// and return a map
func KVArrToMap(props ...string) map[string]string {
	properties := map[string]string{}
	for _, propStr := range props {
		arr := strings.Split(propStr, "=")
		if len(arr) == 1 && arr[0] != "" {
			properties[arr[0]] = ""
		} else if len(arr) == 2 {
			properties[arr[0]] = arr[1]
		} else if len(arr) > 2 {
			val := strings.Join(arr[1:], "=")
			properties[arr[0]] = val
		}
	}
	return properties
}

// MapToKVArr transforms a map into a key-value array
// such as: `"Prop1=Value1", "Prop2=Value2", ...`
func MapToKVArr(properties map[string]string) []string {
	props := []string{}
	for k, v := range properties {
		props = append(props, F("%s=%s", k, v))
	}
	return props
}

// MapKeys returns the keys of a map
func MapKeys(m map[string]interface{}) []string {
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Must panics on error
func Must(e error) {
	if e != nil {
		panic(NewError(3, e))
	}
}

// GetPort asks the kernel for a free open port that is ready to use.
func GetPort(hostPort string) (int, error) {
	if hostPort == "" {
		hostPort = "localhost:0"
	}
	addr, err := net.ResolveTCPAddr("tcp", hostPort)
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// MJ returns the JSON string of a map
func MJ(args ...interface{}) string {
	return string(MarshalMap(M(args...)))
}

// ToMap converst an interface to a map
func ToMap(i interface{}) Map {
	m := M()
	jBytes, _ := json.Marshal(i)
	json.Unmarshal(jBytes, &m)
	return m
}

// Marshal marshals an interface into json
func Marshal(i interface{}) string {
	jBytes, _ := json.Marshal(i)
	return string(jBytes)
}

// MarshalMap marshals a map into json
func MarshalMap(m map[string]interface{}) []byte {
	jBytes, _ := json.Marshal(m)
	return jBytes
}

// Unmarshal unmarshals into an objPtr
func Unmarshal(s string, objPtr interface{}) error {
	err := json.Unmarshal([]byte(s), objPtr)
	if err != nil {
		err = Error(err, "could not unmarshal")
	}
	return err
}

// UnmarshalMap unmarshals into a map of interface
func UnmarshalMap(s string) (map[string]interface{}, error) {
	m := M()
	err := json.Unmarshal([]byte(s), &m)
	if err != nil {
		err = Error(err, "could not unmarshal into map")
		return m, err
	}
	return m, nil
}

// UnmarshalArray unmarshals into a array of interface
func UnmarshalArray(s string) ([]interface{}, error) {
	a := []interface{}{}
	err := json.Unmarshal([]byte(s), &a)
	if err != nil {
		err = Error(err, "could not unmarshal into array")
		return a, err
	}
	return a, nil
}

// NewTsID creates a new timestamp ID
func NewTsID(prefix ...string) string {
	p := ""
	if len(prefix) > 0 {
		p = prefix[0] + "."
	}
	tsMilli := int64(cast.ToFloat64(time.Now().UnixNano()) / 1000000.0)
	return F("%s%d.%s", p, tsMilli, RandString(AplhanumericRunes, 3))
}

// ArrI returns an array of interface
func ArrI(items ...interface{}) []interface{} {
	return items
}

// ArrStr returns an array of strings
func ArrStr(items ...string) []string {
	return items
}

// ArrContains returns true if array of strings contains
func ArrContains(items []string, subItem string) bool {
	_, ok := ArrMapString(items)[subItem]
	return ok
}

// StructField is a field of a struct
type StructField struct {
	Field reflect.StructField
	Value reflect.Value
	JKey  string
}

// StructFields returns the fields of a struct
func StructFields(obj interface{}) (fields []StructField) {
	var t reflect.Type
	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		t = reflect.Indirect(value).Type()
	} else {
		t = reflect.TypeOf(obj)
	}
	for i := 0; i < t.NumField(); i++ {
		var valueField reflect.Value
		if value.Kind() == reflect.Ptr {
			valueField = value.Elem().Field(i)
		} else {
			valueField = value.Field(i)
		}
		sField := t.Field(i)
		jKey := strings.Split(sField.Tag.Get("json"), ",")[0]
		fields = append(fields, StructField{sField, valueField, jKey})
	}
	return fields
}

// CloneValue clones a pointer to another
func CloneValue(source interface{}, destin interface{}) {
	x := reflect.ValueOf(source)
	if x.Kind() == reflect.Ptr {
		starX := x.Elem()
		y := reflect.New(starX.Type())
		starY := y.Elem()
		starY.Set(starX)
		reflect.ValueOf(destin).Elem().Set(y.Elem())
	} else {
		destin = x.Interface()
	}
}

// PathExists returns true if path exists
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
