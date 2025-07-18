package g

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v2"
)

var (
	json           = jsoniter.ConfigCompatibleWithStandardLibrary
	nonWordPattern = regexp.MustCompile(`\W+`)
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
	// AlphaNumericRunes are alphanumeric chars
	AlphaNumericRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	// TokenRunes are alphanumeric+ chars for tokens
	TokenRunes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890_."
)

type (
	// Map is map[string]interface{}
	Map map[string]interface{}

	// Strings is an array of strings
	Strings []string
)

// JSONScanner scans value into Jsonb, implements sql.Scanner interface
func JSONScanner(destPtr, value interface{}) error {
	var bytes []byte

	switch v := value.(type) {
	case []byte:
		bytes = value.([]byte)
	case string:
		bytes = []byte(value.(string))
	default:
		_ = v
		return Error("Failed to unmarshal JSONB value")
	}

	err := json.Unmarshal(bytes, destPtr)
	if err != nil {
		return Error(err, "Could not unmarshal bytes")
	}
	return nil
}

// JSONValuer return json value, implement driver.Valuer interface
func JSONValuer(val interface{}, defVal string) (driver.Value, error) {
	if val == nil {
		return []byte(defVal), nil
	}
	jBytes, err := json.Marshal(val)
	if err != nil {
		return nil, Error(err, "could not marshal")
	}
	return jBytes, nil
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (j *Map) Scan(value interface{}) error {
	return JSONScanner(j, value)
}

// Value return json value, implement driver.Valuer interface
func (j Map) Value() (driver.Value, error) {
	return JSONValuer(j, "{}")
}

// ToMapString returns the value as a Map of strings
func ToMapString(j map[string]interface{}) map[string]string {
	m := map[string]string{}
	for k, v := range j {
		m[k] = cast.ToString(v)
	}
	return m
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

// R : Replacer with optional spaces around keys
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

// Rm is like R, for replacing with a map. replaces  {var}
func Rm(format string, m map[string]interface{}) string {
	if len(m) == 0 {
		return format
	}

	var err error
	args, i := make([]string, len(m)*4), 0

	for k, v := range m {
		args[i] = "{" + k + "}"
		args[i+1], err = cast.ToStringE(v)
		if err != nil {
			args[i+1] = Marshal(v)
		}
		i += 2
	}

	return strings.NewReplacer(args...).Replace(format)
}

// Rme is like Rm, for replacing with a map. replaces ${var} and {var}
func Rme(format string, m map[string]interface{}) string {
	if len(m) == 0 {
		return format
	}

	var err error
	args, i := make([]string, len(m)*4), 0

	for k, v := range m {
		args[i] = "${" + k + "}"
		args[i+1], err = cast.ToStringE(v)
		if err != nil {
			args[i+1] = Marshal(v)
		}
		i += 2
	}

	for k, v := range m {
		args[i] = "{" + k + "}"
		args[i+1], err = cast.ToStringE(v)
		if err != nil {
			args[i+1] = Marshal(v)
		}
		i += 2
	}

	return strings.NewReplacer(args...).Replace(format)
}

// Rmd is like Rm, for replacing with a map. replaces ${var}
func Rmd(format string, m map[string]interface{}) string {
	if len(m) == 0 {
		return format
	}

	var err error
	args, i := make([]string, len(m)*4), 0

	for k, v := range m {
		args[i] = "${" + k + "}"
		args[i+1], err = cast.ToStringE(v)
		if err != nil {
			args[i+1] = Marshal(v)
		}
		i += 2
	}

	return strings.NewReplacer(args...).Replace(format)
}

// Match is a regex match
type Match struct {
	Full  string
	Group []string
}

// MatchesGroup returns an array of a group value index
func MatchesGroup(whole, pattern string, i int) (a []string) {
	matches := Matches(whole, pattern)
	a = make([]string, len(matches))
	for j, m := range matches {
		if i < len(m.Group) {
			a[j] = m.Group[i]
		}
	}
	return
}

// Matches returns potential regex matches
func Matches(whole, pattern string) (matches []Match) {
	regex := *regexp.MustCompile(pattern)
	result := regex.FindAllStringSubmatch(whole, -1)
	matches = make([]Match, len(result))
	for i, arr := range result {
		matches[i] = Match{Full: arr[0], Group: arr}
		if len(arr) > 1 {
			matches[i].Group = arr[1:]
		}
	}
	return
}

func WildCardMatch(whole string, pattens []string) bool {
	whole = strings.TrimSpace(strings.ToLower(whole))
	for _, pattern := range pattens {
		pattern = strings.TrimSpace(strings.ToLower(pattern))
		if strings.HasSuffix(pattern, "*") &&
			strings.HasPrefix(whole, strings.TrimSuffix(pattern, "*")) {
			return true
		}
		if strings.HasPrefix(pattern, "*") &&
			strings.HasSuffix(whole, strings.TrimPrefix(pattern, "*")) {
			return true
		}

		patternTrimmed := strings.TrimSuffix(strings.TrimPrefix(pattern, "*"), "*")
		if strings.HasSuffix(pattern, "*") &&
			strings.HasPrefix(pattern, "*") &&
			strings.Contains(whole, patternTrimmed) {
			return true
		}
		if patternArr := strings.Split(pattern, "*"); len(patternArr) == 2 {
			if strings.HasPrefix(whole, patternArr[0]) && strings.HasSuffix(whole, patternArr[1]) {
				return true
			}
		}
		if whole == pattern {
			return true
		}
	}
	return false
}

// ReplaceNonWord replaces characters not: [^a-zA-Z0-9_]
func ReplaceNonWord(in, replaceWith string) (out string) {
	return string(nonWordPattern.ReplaceAll([]byte(in), []byte(replaceWith)))
}

// PrintT prints the type of object
func PrintT(v interface{}) {
	if IsDebugLow() {
		args := addCaller([]interface{}{})
		doLog(ZLogErr.Debug(), F("%T", v), args)
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

// ToMap convert an interface to a map via JSON
func ToMap(i interface{}) map[string]any {
	m := M()
	jBytes, _ := json.Marshal(i)
	json.Unmarshal(jBytes, &m)
	return m
}

// AsMap converts a map to a map via cast
func AsMap(value interface{}, toLowerKey ...bool) map[string]interface{} {
	m0 := M()

	lowerKey := false
	if len(toLowerKey) > 0 && toLowerKey[0] {
		lowerKey = true
	}

	switch m1 := value.(type) {
	case map[string]interface{}:
		m0 = m1
	case map[string]string:
		for k, v := range m1 {
			m0[k] = v
		}
	case Map:
		m0 = m1
	default:
		v := reflect.ValueOf(value)
		if v.Kind() == reflect.Map {
			iter := v.MapRange()
			for iter.Next() {
				key := iter.Key()
				val := iter.Value()
				m0[cast.ToString(key.Interface())] = val.Interface()
			}
		}
	}

	m2 := M()
	if lowerKey {
		for k, v := range m0 {
			m2[strings.ToLower(k)] = v
		}
	} else {
		m2 = m0
	}

	return m2
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
	return json.Unmarshal([]byte(s), objPtr)
}

// UnmarshalYAML unmarshals into an objPtr
func UnmarshalYAML(s string, objPtr interface{}) error {
	return yaml.Unmarshal([]byte(s), objPtr)
}

// UnmarshalMap unmarshals into a map of interface
func UnmarshalMap(s string) (map[string]interface{}, error) {
	m := M()
	err := json.Unmarshal([]byte(s), &m)
	if err != nil {
		return m, err
	}
	return m, nil
}

// UnmarshalMap unmarshals into a map of interface
func UnmarshalYAMLMap(s string) (map[string]interface{}, error) {
	m := M()
	err := yaml.Unmarshal([]byte(s), &m)
	if err != nil {
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
	return F("%s%d.%s", p, tsMilli, RandString(AlphaNumericRunes, 3))
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

// PathExists returns true if path exists
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// PathSize returns the total size of the file or folder at path
// it calculates the total size of all nested files recursively
func PathSize(path string) (size int64, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, Error(err, "could not stat path")
	}

	if !info.IsDir() {
		return info.Size(), nil
	}

	err = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, Error(err, "error walking path")
	}

	return size, nil
}

// In returns true if `item` matches a value in `potMatches`
func In[T comparable](item T, potMatches ...T) bool {
	for _, m := range potMatches {
		if item == m {
			return true
		}
	}
	return false
}

// HasPrefix returns true if `item` has prefix of a value in `potPrefixes`
func HasPrefix(item string, potPrefixes ...string) bool {
	for _, p := range potPrefixes {
		if strings.HasPrefix(item, p) {
			return true
		}
	}
	return false
}

// ChunkBy seperates into chunks
func ChunkBy(items []string, chunkSize int) (chunks [][]string) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}
	return append(chunks, items)
}

// Join joins the array of strings
func (ss Strings) Join(sep string) string {
	return strings.Join(ss, sep)
}

// Print prints one string entry per line
func (ss Strings) Print(sep string) {
	println(ss.Join(sep))
}

// ExecuteTemplate executes the templates passed
func ExecuteTemplate(text string, values map[string]interface{}) (out string, err error) {

	var output bytes.Buffer
	t, err := template.New("t1").Parse(text)
	if err != nil {
		err = Error(err, "error parsing template")
		return
	}

	err = t.Execute(&output, values)
	if err != nil {
		err = Error(err, "error execute template")
		return
	}

	return output.String(), nil
}

func MD5(text ...string) string {
	hash := md5.Sum([]byte(strings.Join(text, "")))
	return hex.EncodeToString(hash[:])
}

// JSONConvert converts from an interface to another via JSON
func JSONConvert(source interface{}, destination interface{}) (err error) {
	b, err := JSONMarshal(source)
	if err != nil {
		return
	}

	err = JSONUnmarshal(b, destination)
	if err != nil {
		return
	}
	return
}

// JSONMarshal does not escape html as the original marshaller does,
// which escapes <, >, & etc. into unicode such as \u003e
func JSONMarshal(t interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(t)
	return bytes.TrimRight(buffer.Bytes(), "\n"), err
}

// JSONUnmarshalToMap
func JSONUnmarshalToMap(b []byte) (map[string]interface{}, error) {
	m := M()
	err := json.Unmarshal(b, &m)
	if err != nil {
		err = Error(err, "could not unmarshal")
	}
	return m, err
}

// JSONUnmarshal
func JSONUnmarshal(b []byte, p interface{}) error {
	return json.Unmarshal(b, p)
}

func IsMatched(filters []string, name string) bool {
	name = strings.ToLower(name)
	for _, filter := range filters {
		filter = strings.ToLower(filter)
		if filter == "" {
			continue
		}
		if strings.HasSuffix(filter, "*") &&
			strings.HasPrefix(name, strings.TrimSuffix(filter, "*")) {
			return true
		}
		if strings.HasPrefix(filter, "*") &&
			strings.HasSuffix(name, strings.TrimPrefix(filter, "*")) {
			return true
		}
		if strings.HasSuffix(filter, "*") &&
			strings.HasPrefix(filter, "*") &&
			strings.Contains(name, strings.TrimPrefix(strings.TrimSuffix(filter, "*"), "*")) {
			return true
		}
		if filter == name {
			return true
		}
	}
	return false
}

// Ptr creates a pointer to the passed value
func Ptr[T any](t T) *T {
	return &t
}
func PtrVal[T any](t *T) T {
	if t != nil {
		return *t
	}
	return *new(T)
}

// String returns a pointer to the string value passed in.
func String(v string) *string {
	return &v
}

// StringVal returns a the value of the string
func StringVal(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}

// Int returns a pointer to the int value passed in.
func Int(v int) *int {
	return &v
}

// IntVal returns the value of the int
func IntVal(v *int) int {
	if v != nil {
		return *v
	}
	return 0
}

// UInt returns a pointer to the uint value passed in.
func UInt(v uint) *uint {
	return &v
}

// UIntVal returns the value of the uint
func UIntVal(v *uint) uint {
	if v != nil {
		return *v
	}
	return 0
}

// Int64 returns a pointer to the int64 value passed in.
func Int64(v int64) *int64 {
	return &v
}

// Int64Val returns the value of the int64
func Int64Val(v *int64) int64 {
	if v != nil {
		return *v
	}
	return 0
}

// UInt64 returns a pointer to the uint64 value passed in.
func UInt64(v uint64) *uint64 {
	return &v
}

// UInt64Val returns the value of the uint64
func UInt64Val(v *uint64) uint64 {
	if v != nil {
		return *v
	}
	return 0
}

// Bool returns a pointer to the bool value passed in.
func Bool(v bool) *bool {
	return &v
}

// BoolVal returns the value of the bool
func BoolVal(v *bool) bool {
	if v != nil {
		return *v
	}
	return false
}

// Time returns a pointer to the time value passed in.
func Time(v time.Time) *time.Time {
	return &v
}

// TimeVal returns the value of the time
func TimeVal(v *time.Time) time.Time {
	if v != nil {
		return *v
	}
	return time.Time{}
}

// CompareVersions uses integers for each part to compare
// when comparing strings, 'v0.0.40' > 'v0.0.5' = False
// when it should be True.
func CompareVersions(current, latest string) (isNew bool, err error) {
	current = strings.Replace(current, "v", "", 1)
	latest = strings.Replace(latest, "v", "", 1)

	currentArr := strings.Split(current, ".")
	latestArr := strings.Split(latest, ".")

	if len(currentArr) != len(latestArr) {
		return false, Error("incompatible version structures. `%s` vs `%s`", current, latest)
	}

	for i := 0; i < len(currentArr); i++ {
		currentVal, err := cast.ToIntE(currentArr[i])
		if err != nil {
			return false, Error(err, "unable to convert parts to integer: %s", current)
		}

		latestVal, err := cast.ToIntE(latestArr[i])
		if err != nil {
			return false, Error(err, "unable to convert parts to integer: %s", latest)
		}

		switch {
		case latestVal == currentVal:
			continue
		case latestVal > currentVal:
			return true, nil
		case latestVal < currentVal:
			return false, nil
		}
	}

	return false, nil
}

func DurationString(duration time.Duration) (d string) {
	secs := cast.ToInt(math.Floor(duration.Seconds()))
	mins := secs / 60
	hours := mins / 60
	days := hours / 24
	years := days / 365

	if secs < 60 {
		return F("%ds", secs)
	}

	if mins < 60 {
		return F("%dm %ds", mins, secs%60)
	}

	if hours < 24 {
		return F("%dh %dm", hours, mins%60)
	}

	if days < 365 {
		return F("%dd %dh", days, hours%24)
	}

	return F("%dy %dd", years, days%365)
}

// the SHA-1 hash of the very first commit in a Git repository
func GetRootCommit(dirPath string) (rootCommit string) {
	// get first sha
	cmd := exec.Command("git", "rev-list", "--max-parents=0", "HEAD")
	cmd.Dir = dirPath
	out, err := cmd.Output()
	if err == nil {
		// in case of multiple root commits, take first line
		rootCommit = strings.TrimSpace(strings.Split(strings.TrimSpace(string(out)), "\n")[0])
	}
	return
}

// Finds the root folder where a  parent file or folder matches name
func GetRootFolder(startDir, name string) (string, error) {
	dir := startDir

	for {
		// Check if the directory / file exists in the current directory
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			// Found the root directory, return the current directory
			return dir, nil
		}

		// Move to the parent directory
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// Reached the root of the filesystem without finding
			return "", Error("%s file/directory not found", name)
		}

		dir = parentDir
	}
}

func ExtractTarGz(filePath, outFolder string) (err error) {
	gzipStream, err := os.Open(filePath)
	if err != nil {
		return Error(err, "could not open file")
	}
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Fatal("ExtractTarGz: NewReader failed")
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return Error(err, "could not extract archive")
		}

		outPath := path.Join(outFolder, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(outPath, 0755); err != nil {
				return Error(err, "could not create directory => %s", outPath)
			}
		case tar.TypeReg:
			outFile, err := os.Create(outPath)
			if err != nil {
				return Error(err, "could not create file => %s", outPath)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return Error(err, "could not write to file => %s", outPath)
			}
			outFile.Close()

		default:
			return Error("unknown type: %s in %s", header.Typeflag, header.Name)
		}
	}

	return nil
}

// os.Getenv with a default value
func Getenv(key string, def ...string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	if len(def) > 0 {
		return def[0]
	}
	return ""
}

// IsNil properly returns true for map or array nil values
// have encountered where map[string]interface {}(nil) != nil after unmarshal
func IsNil(val any) bool {
	if val == nil {
		return true
	}

	switch vt := val.(type) {
	case map[string]any:
		if vt == nil {
			return true
		}
	case map[any]any:
		if vt == nil {
			return true
		}
	case []any:
		if len(vt) == 0 {
			return true
		}
	}
	return false
}

// MarshalOrdered marshals an interface into json with sorted keys for deterministic output
// BenchmarkMarshalOrdered/SmallMap-Standard-10         	 1000000	      1045 ns/op	     525 B/op	      12 allocs/op
// BenchmarkMarshalOrdered/SmallMap-Ordered-10          	 1000000	      1116 ns/op	     296 B/op	      17 allocs/op
// BenchmarkMarshalOrdered/MediumMap-Standard-10        	  596228	      3219 ns/op	    1386 B/op	      30 allocs/op
// BenchmarkMarshalOrdered/MediumMap-Ordered-10         	  578047	      2630 ns/op	    1128 B/op	      41 allocs/op
// BenchmarkMarshalOrdered/LargeMap-Standard-10         	  115510	      9226 ns/op	    5303 B/op	     113 allocs/op
// BenchmarkMarshalOrdered/LargeMap-Ordered-10          	  158793	      8507 ns/op	    4739 B/op	     154 allocs/op
func MarshalOrdered(val any) (string, error) {
	switch v := val.(type) {
	case map[string]interface{}:
		return marshalOrderedMap(v)
	case Map:
		return marshalOrderedMap(map[string]interface{}(v))
	default:
		// If not a map type, use regular marshaling
		bytes, err := json.Marshal(val)
		if err != nil {
			return "", Error(err, "failed to marshal value")
		}
		return string(bytes), nil
	}
}

func marshalOrderedMap(m map[string]interface{}) (string, error) {
	// Get all keys and sort them
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build ordered map with nested ordering
	var b strings.Builder
	b.WriteString("{")

	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}

		// Marshal the key
		keyBytes, err := json.Marshal(k)
		if err != nil {
			return "", Error(err, "failed to marshal key")
		}
		b.Write(keyBytes)
		b.WriteString(":")

		// Handle the value based on its type
		switch vv := m[k].(type) {
		case map[string]interface{}:
			// If value is a map, recursively order it
			ordered, err := marshalOrderedMap(vv)
			if err != nil {
				return "", err
			}
			b.WriteString(ordered)
		case Map:
			// If value is a Map type, convert and order it
			ordered, err := marshalOrderedMap(map[string]interface{}(vv))
			if err != nil {
				return "", err
			}
			b.WriteString(ordered)
		case []interface{}:
			// For arrays, need to check for nested maps
			arrayBytes, err := marshalOrderedArray(vv)
			if err != nil {
				return "", err
			}
			b.WriteString(arrayBytes)
		default:
			// Regular value, use standard marshaling
			valueBytes, err := json.Marshal(vv)
			if err != nil {
				return "", Error(err, "failed to marshal value")
			}
			b.Write(valueBytes)
		}
	}

	b.WriteString("}")
	return b.String(), nil
}

func marshalOrderedArray(arr []interface{}) (string, error) {
	var b strings.Builder
	b.WriteString("[")

	for i, v := range arr {
		if i > 0 {
			b.WriteString(",")
		}

		// Handle the value based on its type
		switch vv := v.(type) {
		case map[string]interface{}:
			// If value is a map, recursively order it
			ordered, err := marshalOrderedMap(vv)
			if err != nil {
				return "", err
			}
			b.WriteString(ordered)
		case Map:
			// If value is a Map type, convert and order it
			ordered, err := marshalOrderedMap(map[string]interface{}(vv))
			if err != nil {
				return "", err
			}
			b.WriteString(ordered)
		case []interface{}:
			// For nested arrays, recurse
			nestedArrayBytes, err := marshalOrderedArray(vv)
			if err != nil {
				return "", err
			}
			b.WriteString(nestedArrayBytes)
		default:
			// Regular value, use standard marshaling
			valueBytes, err := json.Marshal(vv)
			if err != nil {
				return "", Error(err, "failed to marshal array value")
			}
			b.Write(valueBytes)
		}
	}

	b.WriteString("]")
	return b.String(), nil
}
