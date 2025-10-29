package mackerelplugin

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCalcDiff(t *testing.T) {
	var mp MackerelPlugin

	val1 := 10.0
	val2 := 0.0
	now := time.Now()
	last := time.Unix(now.Unix()-10, 0)

	diff, err := mp.calcDiff(val1, now, val2, last)
	if diff != 60 {
		t.Errorf("calcDiff: %f should be %f", diff, 60.0)
	}
	if err != nil {
		t.Error("calcDiff causes an error")
	}
}

func TestCalcDiffWithReset(t *testing.T) {
	var mp MackerelPlugin

	val := 10.0
	now := time.Now()
	lastval := 12345.0
	last := time.Unix(now.Unix()-60, 0)

	diff, err := mp.calcDiff(val, now, lastval, last)
	if err == nil {
		t.Errorf("calcDiffUint32 with counter reset should cause an error: %f", diff)
	}
}

func TestCalcDiffWithUInt32WithReset(t *testing.T) {
	var mp MackerelPlugin

	val := uint32(10)
	now := time.Now()
	lastval := uint32(12345)
	last := time.Unix(now.Unix()-60, 0)

	diff, err := mp.calcDiffUint32(val, now, lastval, last, 10)
	if err != nil {
	} else {
		t.Errorf("calcDiffUint32 with counter reset should cause an error: %f", diff)
	}
}

func TestCalcDiffWithUInt32Overflow(t *testing.T) {
	var mp MackerelPlugin

	val := uint32(10)
	now := time.Now()
	lastval := math.MaxUint32 - uint32(10)
	last := time.Unix(now.Unix()-60, 0)

	diff, err := mp.calcDiffUint32(val, now, lastval, last, 10)
	if diff != 21.0 {
		t.Errorf("calcDiff: last: %d, now: %d, %f should be %f", val, lastval, diff, 21.0)
	}
	if err != nil {
		t.Error("calcDiff causes an error")
	}
}

func TestCalcDiffWithUInt64WithReset(t *testing.T) {
	var mp MackerelPlugin

	val := uint64(10)
	now := time.Now()
	lastval := uint64(12345)
	last := time.Unix(now.Unix()-60, 0)

	diff, err := mp.calcDiffUint64(val, now, lastval, last, 10)
	if err != nil {
	} else {
		t.Errorf("calcDiffUint64 with counter reset should cause an error: %f", diff)
	}
}

func TestCalcDiffWithUInt64Overflow(t *testing.T) {
	var mp MackerelPlugin

	val := uint64(10)
	now := time.Now()
	lastval := math.MaxUint64 - uint64(10)
	last := time.Unix(now.Unix()-60, 0)

	diff, err := mp.calcDiffUint64(val, now, lastval, last, 10)
	if diff != 21.0 {
		t.Errorf("calcDiff: last: %d, now: %d, %f should be %f", val, lastval, diff, 21.0)
	}
	if err != nil {
		t.Error("calcDiff causes an error")
	}
}

func TestPrintValueUint32(t *testing.T) {
	var mp MackerelPlugin
	s := new(bytes.Buffer)
	var now = time.Unix(1437227240, 0)
	mp.printValue(s, "test", uint32(10), now)

	expected := []byte("test\t10\t1437227240\n")

	if !bytes.Equal(expected, s.Bytes()) {
		t.Fatalf("not matched, expected: %s, got: %s", expected, s)
	}
}

func TestPrintValueUint64(t *testing.T) {
	var mp MackerelPlugin
	s := new(bytes.Buffer)
	var now = time.Unix(1437227240, 0)
	mp.printValue(s, "test", uint64(10), now)

	expected := []byte("test\t10\t1437227240\n")

	if !bytes.Equal(expected, s.Bytes()) {
		t.Fatalf("not matched, expected: %s, got: %s", expected, s)
	}
}

func TestPrintValueFloat64(t *testing.T) {
	var mp MackerelPlugin
	s := new(bytes.Buffer)
	var now = time.Unix(1437227240, 0)
	mp.printValue(s, "test", float64(10.0), now)

	expected := []byte("test\t10.000000\t1437227240\n")

	if !bytes.Equal(expected, s.Bytes()) {
		t.Fatalf("not matched, expected: %s, got: %s", expected, s)
	}
}

type emptyPlugin struct {
}

func (*emptyPlugin) FetchMetrics() (map[string]interface{}, error) {
	return nil, nil
}

func (*emptyPlugin) GraphDefinition() map[string]Graphs {
	return nil
}

func boolPtr(b bool) *bool {
	return &b
}

func TestFetchLastValues_stateFileNotFound(t *testing.T) {
	var mp MackerelPlugin
	mp.Plugin = &emptyPlugin{}
	mp.Tempfile = "state_file_should_not_exist.json"
	mp.diff = boolPtr(true)
	m, err := mp.FetchLastValues()
	if err != nil {
		t.Fatalf("FetchLastValues: %v", err)
	}
	if !m.Timestamp.IsZero() {
		t.Errorf("Timestamp = %v; want 0001-01-01", m.Timestamp)
	}
}

func TestFetchLastValues_readStateSameTime(t *testing.T) {
	var mp MackerelPlugin
	mp.Plugin = &emptyPlugin{}
	f, err := os.CreateTemp("", "mackerel-plugin-helper.")
	if err != nil {
		t.Fatal(err)
	}
	file := f.Name()
	defer os.Remove(file)
	mp.Tempfile = file
	mp.diff = boolPtr(true)
	metricValues := MetricValues{
		Values:    make(map[string]interface{}),
		Timestamp: time.Now(),
	}
	err = mp.saveValues(metricValues)
	if err != nil {
		t.Fatal(err)
	}

	_, err = mp.fetchLastValuesSafe(metricValues.Timestamp)
	if err != errStateUpdated {
		t.Errorf("FetchLastValues: %v; want %v", err, errStateUpdated)
	}
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func TestOutput(t *testing.T) {
	var tests = []func() []string{
		tcFormatValues,
		tcFormatValuesAbsoluteName,
		tcFormatValuesAbsoluteNameButNoPrefix,
		tcFormatValuesWithCounterReset,
		tcFormatFloatValuesWithCounterReset,
		tcFormatValuesWithOverflow,
		tcFormatValuesWithOverflowAndTooHighDifference,
		tcFormatValuesWithOverflowAndNoLastDiff,
		tcFormatValuesWithWildcard,
		tcFormatValuesWithWildcardAndAbsoluteName,
		tcFormatValuesWithWildcardAndNoDiff,
		tcFormatValuesWithWildcardAstarisk,
		tcOutputDefinitions,
		tcPluginWithPrefixOutputDefinitions,
		tcPluginWithPrefixOutputValues,
		tcPluginWithPrefixOutputValues2,
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("TestOutput %s", getFunctionName(tc)), func(t *testing.T) {
			orig := os.Stdout
			t.Cleanup(func() { os.Stdout = orig })

			r, w, _ := os.Pipe()
			os.Stdout = w

			wants := tc()
			w.Close()
			actual, _ := io.ReadAll(r)

			expected := strings.Join(wants, "\n")
			if len(wants) > 0 {
				expected += "\n"
			}

			if string(actual) != expected {
				t.Error("Failure!")
			}
		})
	}
}

func tcFormatValues() []string {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64"}
	now := time.Unix(1437227240, 0)
	metricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": uint64(1000)},
		Timestamp: now,
	}
	lastMetricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": uint64(500), ".last_diff.cmd_get": 300.0},
		Timestamp: now.Add(-time.Duration(60) * time.Second),
	}
	mp.formatValues(prefix, metric, metricValues, lastMetricValues)

	return []string{"foo.cmd_get	500.000000	1437227240"}
}

func tcFormatValuesAbsoluteName() []string {
	var mp MackerelPlugin
	prefixA := "foo"
	metricA := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64", AbsoluteName: true}
	prefixB := "bar"
	metricB := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64", AbsoluteName: true}
	now := time.Unix(1437227240, 0)
	metricValues := MetricValues{
		Values:    map[string]interface{}{"foo.cmd_get": uint64(1000), "bar.cmd_get": uint64(1234)},
		Timestamp: now,
	}
	lastMetricValues := MetricValues{
		Values:    map[string]interface{}{"foo.cmd_get": uint64(500), ".last_diff.foo.cmd_get": 300.0, "bar.cmd_get": uint64(600), ".last_diff.bar.cmd_get": 400.0},
		Timestamp: now.Add(-time.Duration(60) * time.Second),
	}
	mp.formatValues(prefixA, metricA, metricValues, lastMetricValues)
	mp.formatValues(prefixB, metricB, metricValues, lastMetricValues)

	return []string{
		"foo.cmd_get	500.000000	1437227240",
		"bar.cmd_get	634.000000	1437227240",
	}
}

func tcFormatValuesAbsoluteNameButNoPrefix() []string {
	var mp MackerelPlugin
	prefix := ""
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64", AbsoluteName: true}
	now := time.Unix(1437227240, 0)
	metricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": uint64(1000)},
		Timestamp: now,
	}
	lastMetricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": uint64(500), ".last_diff.cmd_get": 300.0},
		Timestamp: now.Add(-time.Duration(60) * time.Second),
	}
	mp.formatValues(prefix, metric, metricValues, lastMetricValues)

	return []string{
		"cmd_get	500.000000	1437227240",
	}
}

func tcFormatValuesWithCounterReset() []string {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64"}
	now := time.Unix(1437227240, 0)
	metricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": uint64(10)},
		Timestamp: now,
	}
	lastMetricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": uint64(500), ".last_diff.cmd_get": 300.0},
		Timestamp: now.Add(-time.Duration(60) * time.Second),
	}
	mp.formatValues(prefix, metric, metricValues, lastMetricValues)

	return nil
}

func tcFormatFloatValuesWithCounterReset() []string {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "float"}
	now := time.Unix(1437227240, 0)
	metricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": 10.0},
		Timestamp: now,
	}
	lastMetricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": 500.0, ".last_diff.cmd_get": 300.0},
		Timestamp: now.Add(-time.Duration(60) * time.Second),
	}
	mp.formatValues(prefix, metric, metricValues, lastMetricValues)

	return nil
}

func tcFormatValuesWithOverflow() []string {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64"}
	now := time.Unix(1437227240, 0)
	metricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": uint64(500)},
		Timestamp: now,
	}
	lastMetricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": uint64(math.MaxUint64 - 100), ".last_diff.cmd_get": float64(100.0)},
		Timestamp: now.Add(-time.Duration(60) * time.Second),
	}
	mp.formatValues(prefix, metric, metricValues, lastMetricValues)

	return []string{
		"foo.cmd_get	601.000000	1437227240",
	}
}

func tcFormatValuesWithOverflowAndTooHighDifference() []string {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64"}
	now := time.Unix(1437227240, 0)
	metricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": uint64(500)},
		Timestamp: now,
	}
	lastMetricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": uint64(math.MaxUint64 - 100), ".last_diff.cmd_get": float64(10.0)},
		Timestamp: now.Add(-time.Duration(60) * time.Second),
	}
	mp.formatValues(prefix, metric, metricValues, lastMetricValues)

	return nil
}

func tcFormatValuesWithOverflowAndNoLastDiff() []string {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64"}
	now := time.Unix(1437227240, 0)
	metricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": uint64(500)},
		Timestamp: now,
	}
	lastMetricValues := MetricValues{
		Values:    map[string]interface{}{"cmd_get": uint64(math.MaxUint64 - 100)},
		Timestamp: now.Add(-time.Duration(60) * time.Second),
	}
	mp.formatValues(prefix, metric, metricValues, lastMetricValues)

	return nil
}

func tcFormatValuesWithWildcard() []string {
	var mp MackerelPlugin
	prefix := "foo.#"
	metric := Metrics{Name: "bar", Label: "Get", Diff: true, Type: "uint64"}
	now := time.Unix(1437227240, 0)
	metricValues := MetricValues{
		Values:    map[string]interface{}{"foo.1.bar": uint64(1000), "foo.2.bar": uint64(2000)},
		Timestamp: now,
	}
	lastMetricValues := MetricValues{
		Values:    map[string]interface{}{"foo.1.bar": uint64(500), ".last_diff.foo.1.bar": float64(2.0)},
		Timestamp: now.Add(-time.Duration(60) * time.Second),
	}
	mp.formatValuesWithWildcard(prefix, metric, metricValues, lastMetricValues)

	return []string{
		"foo.1.bar	500.000000	1437227240",
	}
}

func tcFormatValuesWithWildcardAndAbsoluteName() []string {
	// AbsoluteName should be ignored with WildCard
	var mp MackerelPlugin
	prefix := "foo.#"
	metric := Metrics{Name: "bar", Label: "Get", Diff: true, Type: "uint64", AbsoluteName: true}
	now := time.Unix(1437227240, 0)
	metricValues := MetricValues{
		Values:    map[string]interface{}{"foo.1.bar": uint64(1000), "foo.2.bar": uint64(2000)},
		Timestamp: now,
	}
	lastMetricValues := MetricValues{
		Values:    map[string]interface{}{"foo.1.bar": uint64(500), ".last_diff.foo.1.bar": float64(2.0)},
		Timestamp: now.Add(-time.Duration(60) * time.Second),
	}
	mp.formatValuesWithWildcard(prefix, metric, metricValues, lastMetricValues)

	return []string{
		"foo.1.bar	500.000000	1437227240",
	}
}

func tcFormatValuesWithWildcardAndNoDiff() []string {
	var mp MackerelPlugin
	prefix := "foo.#"
	metric := Metrics{Name: "bar", Label: "Get", Diff: false}
	now := time.Unix(1437227240, 0)
	metricValues := MetricValues{
		Values:    map[string]interface{}{"foo.1.bar": float64(1000)},
		Timestamp: now,
	}
	lastMetricValues := MetricValues{
		Values:    map[string]interface{}{"foo.1.bar": float64(500), ".last_diff.foo.1.bar": float64(2.0)},
		Timestamp: now.Add(-time.Duration(60) * time.Second),
	}
	mp.formatValuesWithWildcard(prefix, metric, metricValues, lastMetricValues)

	return []string{
		"foo.1.bar	1000.000000	1437227240",
	}
}

func tcFormatValuesWithWildcardAstarisk() []string {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "*", Label: "Get", Diff: true, Type: "uint64"}
	now := time.Unix(1437227240, 0)
	metricValues := MetricValues{
		Values:    map[string]interface{}{"foo.1": uint64(1000), "foo.2": uint64(2000)},
		Timestamp: now,
	}
	lastMetricValues := MetricValues{
		Values:    map[string]interface{}{"foo.1": uint64(500), ".last_diff.foo.1": float64(2.0)},
		Timestamp: now.Add(-time.Duration(60) * time.Second),
	}
	mp.formatValuesWithWildcard(prefix, metric, metricValues, lastMetricValues)

	return []string{
		"foo.1	500.000000	1437227240",
	}
}

// an example implementation
type MemcachedPlugin struct {
}

var graphdef = map[string]Graphs{
	"memcached.cmd": {
		Label: "Memcached Command",
		Unit:  "integer",
		Metrics: []Metrics{
			{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64"},
		},
	},
}

func (m MemcachedPlugin) GraphDefinition() map[string]Graphs {
	return graphdef
}

func (m MemcachedPlugin) FetchMetrics() (map[string]interface{}, error) {
	var stat map[string]interface{}
	return stat, nil
}

func tcOutputDefinitions() []string {
	var mp MemcachedPlugin
	helper := NewMackerelPlugin(mp)
	helper.OutputDefinitions()

	return []string{
		"# mackerel-agent-plugin",
		`{"graphs":{"memcached.cmd":{"label":"Memcached Command","unit":"integer","metrics":[{"name":"cmd_get","label":"Get","stacked":false}]}}}`,
	}
}

func TestToUint32(t *testing.T) {
	if ret := toUint32(uint32(100)); ret != uint32(100) {
		t.Errorf("toUint32(uint32) returns incorrect value: %v expected to be %v", ret, uint32(100))
	}

	if ret := toUint32(uint64(100)); ret != uint32(100) {
		t.Errorf("toUint32(uint64) returns incorrect value: %v expected to be %v", ret, uint32(100))
	}

	if ret := toUint32(float64(100)); ret != uint32(100) {
		t.Errorf("toUint32(float64) returns incorrect value: %v expected to be %v", ret, uint32(100))
	}

	if ret := toUint32("100"); ret != uint32(100) {
		t.Errorf("toUint32(string) returns incorrect value: %v expected to be %v", ret, uint32(100))
	}
}

func TestToUint64(t *testing.T) {
	if ret := toUint64(uint32(100)); ret != uint64(100) {
		t.Errorf("toUint64(uint32) returns incorrect value: %v expected to be %v", ret, uint64(100))
	}

	if ret := toUint64(uint64(100)); ret != uint64(100) {
		t.Errorf("toUint64(uint64) returns incorrect value: %v expected to be %v", ret, uint64(100))
	}

	if ret := toUint64(float64(100)); ret != uint64(100) {
		t.Errorf("toUint64(float64) returns incorrect value: %v expected to be %v", ret, uint64(100))
	}

	if ret := toUint64("100"); ret != uint64(100) {
		t.Errorf("toUint64(string) returns incorrect value: %v expected to be %v", ret, uint64(100))
	}
}

func TestToFloat64(t *testing.T) {
	if ret := toFloat64(uint32(100)); ret != float64(100) {
		t.Errorf("toFloat64(uint32) returns incorrect value: %v expected to be %v", ret, float64(100))
	}

	if ret := toFloat64(uint64(100)); ret != float64(100) {
		t.Errorf("toFloat64(uint64) returns incorrect value: %v expected to be %v", ret, float64(100))
	}

	if ret := toFloat64(float64(100)); ret != float64(100) {
		t.Errorf("toFloat64(float64) returns incorrect value: %v expected to be %v", ret, float64(100))
	}

	if ret := toFloat64("100"); ret != float64(100) {
		t.Errorf("toFloat64(string) returns incorrect value: %v expected to be %v", ret, float64(100))
	}
}

type testP struct{}

func (t testP) FetchMetrics() (map[string]interface{}, error) {
	ret := make(map[string]interface{})
	ret["bar"] = 15.0
	ret["baz"] = 18.0
	return ret, nil
}

func (t testP) GraphDefinition() map[string]Graphs {
	return map[string]Graphs{
		"": {
			Unit: "integer",
			Metrics: []Metrics{
				{Name: "bar"},
			},
		},
		"fuga": {
			Unit: "float",
			Metrics: []Metrics{
				{Name: "baz"},
			},
		},
	}
}

func (t testP) MetricKeyPrefix() string {
	return "testP"
}

func TestDefaultTempfile(t *testing.T) {
	mp := &MackerelPlugin{}
	filename := filepath.Base(os.Args[0])
	expect := filepath.Join(os.TempDir(), fmt.Sprintf(
		"mackerel-plugin-%s-%x",
		filename,
		sha1.Sum([]byte(strings.Join(os.Args[1:], " "))),
	))
	if mp.tempfilename() != expect {
		t.Errorf("mp.tempfilename() should be %s, but: %s", expect, mp.tempfilename())
	}

	pPrefix := NewMackerelPlugin(testP{})
	expectForPrefix := filepath.Join(os.TempDir(), fmt.Sprintf(
		"mackerel-plugin-testP-%x",
		sha1.Sum([]byte(strings.Join(os.Args[1:], " "))),
	))
	if pPrefix.tempfilename() != expectForPrefix {
		t.Errorf("pPrefix.tempfilename() should be %s, but: %s", expectForPrefix, pPrefix.tempfilename())
	}
}

func TestTempfilenameFromExecutableFilePath(t *testing.T) {
	mp := &MackerelPlugin{}

	wd, _ := os.Getwd()
	// not PluginWithPrefix, regular filename
	expect1 := filepath.Join(os.TempDir(), "mackerel-plugin-foobar-da39a3ee5e6b4b0d3255bfef95601890afd80709")
	filename1 := mp.generateTempfilePath([]string{filepath.Join(wd, "foobar")})
	if filename1 != expect1 {
		t.Errorf("p.generateTempfilePath() should be %s, but: %s", expect1, filename1)
	}

	// not PluginWithPrefix, contains some characters to be sanitized
	expect2 := filepath.Join(os.TempDir(), "mackerel-plugin-some_sanitized_name_1.2-da39a3ee5e6b4b0d3255bfef95601890afd80709")
	filename2 := mp.generateTempfilePath([]string{filepath.Join(wd, "some sanitized:name+1.2")})
	if filename2 != expect2 {
		t.Errorf("p.generateTempfilePath() should be %s, but: %s", expect2, filename2)
	}

	// not PluginWithPrefix, begins with "mackerel-plugin-"
	expect3 := filepath.Join(os.TempDir(), "mackerel-plugin-trimmed-da39a3ee5e6b4b0d3255bfef95601890afd80709")
	filename3 := mp.generateTempfilePath([]string{filepath.Join(wd, "mackerel-plugin-trimmed")})
	if filename3 != expect3 {
		t.Errorf("p.generateTempfilePath() should be %s, but: %s", expect3, filename3)
	}

	// PluginWithPrefix ignores current filename
	pPrefix := NewMackerelPlugin(testP{})
	expectForPrefix := filepath.Join(os.TempDir(), "mackerel-plugin-testP-da39a3ee5e6b4b0d3255bfef95601890afd80709")
	filenameForPrefix := pPrefix.generateTempfilePath([]string{filepath.Join(wd, "foo")})
	if filenameForPrefix != expectForPrefix {
		t.Errorf("pPrefix.generateTempfilePath() should be %s, but: %s", expectForPrefix, filenameForPrefix)
	}

	// Generate sha1 using command-line options, and use it for filename
	expect5 := filepath.Join(os.TempDir(), "mackerel-plugin-mysql-9045504f8fadd7ddcc8962ec1d9fc70e3f7ba627")
	filename5 := mp.generateTempfilePath([]string{filepath.Join(wd, "mackerel-plugin-mysql"), "-host", "hostname1", "-port", "3306"})
	if filename5 != expect5 {
		t.Errorf("p.generateTempfilePath() should be %s, but: %s", expect5, filename5)
	}
}

func TestSetTempfileWithBasename(t *testing.T) {
	var p MackerelPlugin

	expect1 := filepath.Join(os.TempDir(), "my-super-tempfile")
	p.SetTempfileByBasename("my-super-tempfile")
	if p.Tempfile != expect1 {
		t.Errorf("p.SetTempfileByBasename() should set %s, but: %s", expect1, p.Tempfile)
	}

	t.Setenv("MACKEREL_PLUGIN_WORKDIR", "/tmp/somewhere")

	expect2 := filepath.FromSlash("/tmp/somewhere/my-great-tempfile")
	p.SetTempfileByBasename("my-great-tempfile")
	if p.Tempfile != expect2 {
		t.Errorf("p.SetTempfileByBasename() should set %s, but: %s", expect2, p.Tempfile)
	}
}

func tcPluginWithPrefixOutputDefinitions() []string {
	helper := NewMackerelPlugin(testP{})
	helper.OutputDefinitions()

	return []string{
		"# mackerel-agent-plugin",
		`{"graphs":{"testP":{"label":"TestP","unit":"integer","metrics":[{"name":"bar","label":"Bar","stacked":false}]},"testP.fuga":{"label":"TestP Fuga","unit":"float","metrics":[{"name":"baz","label":"Baz","stacked":false}]}}}`,
	}
}

func tcPluginWithPrefixOutputValues() []string {
	helper := NewMackerelPlugin(testP{})
	stat, _ := helper.FetchMetrics()
	key := ""
	metric := helper.GraphDefinition()[key].Metrics[0]
	var lastStat map[string]interface{}
	now := time.Unix(1437227240, 0)
	lastTime := time.Unix(0, 0)
	helper.formatValues(key, metric, MetricValues{Values: stat, Timestamp: now}, MetricValues{Values: lastStat, Timestamp: lastTime})

	return []string{
		"testP.bar	15.000000	1437227240",
	}
}

func tcPluginWithPrefixOutputValues2() []string {
	helper := NewMackerelPlugin(testP{})
	stat, _ := helper.FetchMetrics()
	key := "fuga"
	metric := helper.GraphDefinition()[key].Metrics[0]
	var lastStat map[string]interface{}
	now := time.Unix(1437227240, 0)
	lastTime := time.Unix(0, 0)
	helper.formatValues(key, metric, MetricValues{Values: stat, Timestamp: now}, MetricValues{Values: lastStat, Timestamp: lastTime})

	return []string{
		"testP.fuga.baz	18.000000	1437227240",
	}
}

type testPHasDiff struct{}

func (t testPHasDiff) FetchMetrics() (map[string]interface{}, error) {
	return nil, nil
}

func (t testPHasDiff) GraphDefinition() map[string]Graphs {
	return map[string]Graphs{
		"hoge": {
			Metrics: []Metrics{
				{Name: "hoge1", Label: "hoge1", Diff: true},
			},
		},
	}
}

type testPHasntDiff struct{}

func (t testPHasntDiff) FetchMetrics() (map[string]interface{}, error) {
	return nil, nil
}

func (t testPHasntDiff) GraphDefinition() map[string]Graphs {
	return map[string]Graphs{
		"hoge": {
			Metrics: []Metrics{
				{Name: "hoge1", Label: "hoge1"},
			},
		},
	}
}

func TestPluginHasDiff(t *testing.T) {
	pHasDiff := NewMackerelPlugin(testPHasDiff{})
	if !pHasDiff.hasDiff() {
		t.Errorf("something went wrong")
	}

	pHasntDiff := NewMackerelPlugin(testPHasntDiff{})
	if pHasntDiff.hasDiff() {
		t.Errorf("something went wrong")
	}
}

func TestSaveStateIfContainsInvalidNumbers(t *testing.T) {
	p := NewMackerelPlugin(testPHasDiff{})
	f := createTempState(t)
	defer f.Close()
	p.Tempfile = f.Name()

	stats := map[string]interface{}{
		"key1": 3.0,
		"key2": math.Inf(1),
		"key3": math.Inf(-1),
		"key4": math.NaN(),
	}
	const lastTime = 1624848982

	now := time.Unix(lastTime, 0)
	values := MetricValues{
		Values:    stats,
		Timestamp: now,
	}
	if err := p.saveValues(values); err != nil {
		t.Errorf("saveValues: %v", err)
	}
	values, err := p.FetchLastValues()
	if err != nil {
		t.Fatal("FetchLastValues:", err)
	}
	want := MetricValues{
		Values: map[string]interface{}{
			"_lastTime": float64(lastTime),
			"key1":      3.0,
		},
		Timestamp: now,
	}
	if !reflect.DeepEqual(values, want) {
		t.Errorf("saveValues stores only valid numbers: got %v; want %v", values, want)
	}
}

func createTempState(t testing.TB) *os.File {
	t.Helper()
	f, err := os.CreateTemp("", "mackerel-plugin.")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(f.Name()); err != nil {
			t.Fatal(err)
		}
	})
	return f
}
