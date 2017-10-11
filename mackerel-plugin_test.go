package mackerelplugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
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

	if bytes.Compare(expected, s.Bytes()) != 0 {
		t.Fatalf("not matched, expected: %s, got: %s", expected, s)
	}
}

func TestPrintValueUint64(t *testing.T) {
	var mp MackerelPlugin
	s := new(bytes.Buffer)
	var now = time.Unix(1437227240, 0)
	mp.printValue(s, "test", uint64(10), now)

	expected := []byte("test\t10\t1437227240\n")

	if bytes.Compare(expected, s.Bytes()) != 0 {
		t.Fatalf("not matched, expected: %s, got: %s", expected, s)
	}
}

func TestPrintValueFloat64(t *testing.T) {
	var mp MackerelPlugin
	s := new(bytes.Buffer)
	var now = time.Unix(1437227240, 0)
	mp.printValue(s, "test", float64(10.0), now)

	expected := []byte("test\t10.000000\t1437227240\n")

	if bytes.Compare(expected, s.Bytes()) != 0 {
		t.Fatalf("not matched, expected: %s, got: %s", expected, s)
	}
}

func ExampleFormatValues() {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64"}
	stat := map[string]interface{}{"cmd_get": uint64(1000)}
	lastStat := map[string]interface{}{"cmd_get": uint64(500), ".last_diff.cmd_get": 300.0}
	now := time.Unix(1437227240, 0)
	lastTime := now.Add(-time.Duration(60) * time.Second)
	mp.formatValues(prefix, metric, &stat, &lastStat, now, &lastTime)

	// Output:
	// foo.cmd_get	500.000000	1437227240
}

func ExampleFormatValuesAbsoluteName() {
	var mp MackerelPlugin
	prefixA := "foo"
	metricA := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64", AbsoluteName: true}
	prefixB := "bar"
	metricB := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64", AbsoluteName: true}
	stat := map[string]interface{}{"foo.cmd_get": uint64(1000), "bar.cmd_get": uint64(1234)}
	lastStat := map[string]interface{}{"foo.cmd_get": uint64(500), ".last_diff.foo.cmd_get": 300.0, "bar.cmd_get": uint64(600), ".last_diff.bar.cmd_get": 400.0}
	now := time.Unix(1437227240, 0)
	lastTime := now.Add(-time.Duration(60) * time.Second)
	mp.formatValues(prefixA, metricA, &stat, &lastStat, now, &lastTime)
	mp.formatValues(prefixB, metricB, &stat, &lastStat, now, &lastTime)

	// Output:
	// foo.cmd_get	500.000000	1437227240
	// bar.cmd_get	634.000000	1437227240
}

func ExampleFormatValuesAbsoluteNameButNoPrefix() {
	var mp MackerelPlugin
	prefix := ""
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64", AbsoluteName: true}
	stat := map[string]interface{}{"cmd_get": uint64(1000)}
	lastStat := map[string]interface{}{"cmd_get": uint64(500), ".last_diff.cmd_get": 300.0}
	now := time.Unix(1437227240, 0)
	lastTime := now.Add(-time.Duration(60) * time.Second)
	mp.formatValues(prefix, metric, &stat, &lastStat, now, &lastTime)

	// Output:
	// cmd_get	500.000000	1437227240
}

func ExampleFormatValuesWithCounterReset() {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64"}
	stat := map[string]interface{}{"cmd_get": uint64(10)}
	lastStat := map[string]interface{}{"cmd_get": uint64(500), ".last_diff.cmd_get": 300.0}
	now := time.Unix(1437227240, 0)
	lastTime := now.Add(-time.Duration(60) * time.Second)
	mp.formatValues(prefix, metric, &stat, &lastStat, now, &lastTime)

	// Output:
}

func ExampleFormatFloatValuesWithCounterReset() {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "float"}
	stat := map[string]interface{}{"cmd_get": 10.0}
	lastStat := map[string]interface{}{"cmd_get": 500.0, ".last_diff.cmd_get": 300.0}
	now := time.Unix(1437227240, 0)
	lastTime := now.Add(-time.Duration(60) * time.Second)
	mp.formatValues(prefix, metric, &stat, &lastStat, now, &lastTime)

	// Output:
}

func ExampleFormatValuesWithOverflow() {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64"}
	stat := map[string]interface{}{"cmd_get": uint64(500)}
	lastStat := map[string]interface{}{"cmd_get": uint64(math.MaxUint64 - 100), ".last_diff.cmd_get": float64(100.0)}
	now := time.Unix(1437227240, 0)
	lastTime := now.Add(-time.Duration(60) * time.Second)
	mp.formatValues(prefix, metric, &stat, &lastStat, now, &lastTime)

	// Output:
	// foo.cmd_get	601.000000	1437227240
}

func ExampleFormatValuesWithOverflowAndTooHighDifference() {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64"}
	stat := map[string]interface{}{"cmd_get": uint64(500)}
	lastStat := map[string]interface{}{"cmd_get": uint64(math.MaxUint64 - 100), ".last_diff.cmd_get": float64(10.0)}
	now := time.Unix(1437227240, 0)
	lastTime := now.Add(-time.Duration(60) * time.Second)
	mp.formatValues(prefix, metric, &stat, &lastStat, now, &lastTime)

	// Output:
}

func ExampleFormatValuesWithOverflowAndNoLastDiff() {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64"}
	stat := map[string]interface{}{"cmd_get": uint64(500)}
	lastStat := map[string]interface{}{"cmd_get": uint64(math.MaxUint64 - 100)}
	now := time.Unix(1437227240, 0)
	lastTime := now.Add(-time.Duration(60) * time.Second)
	mp.formatValues(prefix, metric, &stat, &lastStat, now, &lastTime)

	// Output:
}

func ExampleFormatValuesWithWildcard() {
	var mp MackerelPlugin
	prefix := "foo.#"
	metric := Metrics{Name: "bar", Label: "Get", Diff: true, Type: "uint64"}
	stat := map[string]interface{}{"foo.1.bar": uint64(1000), "foo.2.bar": uint64(2000)}
	lastStat := map[string]interface{}{"foo.1.bar": uint64(500), ".last_diff.foo.1.bar": float64(2.0)}
	now := time.Unix(1437227240, 0)
	lastTime := now.Add(-time.Duration(60) * time.Second)
	mp.formatValuesWithWildcard(prefix, metric, &stat, &lastStat, now, &lastTime)

	// Output:
	// foo.1.bar	500.000000	1437227240
}

func ExampleFormatValuesWithWildcardAndAbsoluteName() {
	// AbsoluteName should be ignored with WildCard
	var mp MackerelPlugin
	prefix := "foo.#"
	metric := Metrics{Name: "bar", Label: "Get", Diff: true, Type: "uint64", AbsoluteName: true}
	stat := map[string]interface{}{"foo.1.bar": uint64(1000), "foo.2.bar": uint64(2000)}
	lastStat := map[string]interface{}{"foo.1.bar": uint64(500), ".last_diff.foo.1.bar": float64(2.0)}
	now := time.Unix(1437227240, 0)
	lastTime := now.Add(-time.Duration(60) * time.Second)
	mp.formatValuesWithWildcard(prefix, metric, &stat, &lastStat, now, &lastTime)

	// Output:
	// foo.1.bar	500.000000	1437227240
}

func ExampleFormatValuesWithWildcardAndNoDiff() {
	var mp MackerelPlugin
	prefix := "foo.#"
	metric := Metrics{Name: "bar", Label: "Get", Diff: false}
	stat := map[string]interface{}{"foo.1.bar": float64(1000)}
	lastStat := map[string]interface{}{"foo.1.bar": float64(500), ".last_diff.foo.1.bar": float64(2.0)}
	now := time.Unix(1437227240, 0)
	lastTime := now.Add(-time.Duration(60) * time.Second)
	mp.formatValuesWithWildcard(prefix, metric, &stat, &lastStat, now, &lastTime)

	// Output:
	// foo.1.bar	1000.000000	1437227240
}

func ExampleFormatValuesWithWildcardAstarisk() {
	var mp MackerelPlugin
	prefix := "foo"
	metric := Metrics{Name: "*", Label: "Get", Diff: true, Type: "uint64"}
	stat := map[string]interface{}{"foo.1": uint64(1000), "foo.2": uint64(2000)}
	lastStat := map[string]interface{}{"foo.1": uint64(500), ".last_diff.foo.1": float64(2.0)}
	now := time.Unix(1437227240, 0)
	lastTime := now.Add(-time.Duration(60) * time.Second)
	mp.formatValuesWithWildcard(prefix, metric, &stat, &lastStat, now, &lastTime)

	// Output:
	// foo.1	500.000000	1437227240
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

func ExampleOutputDefinitions() {
	var mp MemcachedPlugin
	helper := NewMackerelPlugin(mp)
	helper.OutputDefinitions()

	// Output:
	// # mackerel-agent-plugin
	// {"graphs":{"memcached.cmd":{"label":"Memcached Command","unit":"integer","metrics":[{"name":"cmd_get","label":"Get","stacked":false}]}}}
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
	var p MackerelPlugin
	filename := filepath.Base(os.Args[0])
	expect := filepath.Join(os.TempDir(), fmt.Sprintf("mackerel-plugin-%s", filename))
	if p.tempfilename() != expect {
		t.Errorf("p.tempfilename() should be %s, but: %s", expect, p.tempfilename())
	}

	pPrefix := NewMackerelPlugin(testP{})
	expectForPrefix := filepath.Join(os.TempDir(), "mackerel-plugin-testP")
	if pPrefix.tempfilename() != expectForPrefix {
		t.Errorf("pPrefix.tempfilename() should be %s, but: %s", expectForPrefix, pPrefix.tempfilename())
	}
}

func TestTempfilenameFromExecutableFilePath(t *testing.T) {
	var p MackerelPlugin

	wd, _ := os.Getwd()
	// not PluginWithPrefix, regular filename
	expect1 := filepath.Join(os.TempDir(), "mackerel-plugin-foobar")
	filename1 := p.generateTempfilePath(filepath.Join(wd, "foobar"))
	if filename1 != expect1 {
		t.Errorf("p.generateTempfilePath() should be %s, but: %s", expect1, filename1)
	}

	// not PluginWithPrefix, contains some characters to be sanitized
	expect2 := filepath.Join(os.TempDir(), "mackerel-plugin-some_sanitized_name_1.2")
	filename2 := p.generateTempfilePath(filepath.Join(wd, "some sanitized:name+1.2"))
	if filename2 != expect2 {
		t.Errorf("p.generateTempfilePath() should be %s, but: %s", expect2, filename2)
	}

	// not PluginWithPrefix, begins with "mackerel-plugin-"
	expect3 := filepath.Join(os.TempDir(), "mackerel-plugin-trimmed")
	filename3 := p.generateTempfilePath(filepath.Join(wd, "mackerel-plugin-trimmed"))
	if filename3 != expect3 {
		t.Errorf("p.generateTempfilePath() should be %s, but: %s", expect3, filename3)
	}

	// PluginWithPrefix ignores current filename
	pPrefix := NewMackerelPlugin(testP{})
	expectForPrefix := filepath.Join(os.TempDir(), "mackerel-plugin-testP")
	filenameForPrefix := pPrefix.generateTempfilePath(filepath.Join(wd, "foo"))
	if filenameForPrefix != expectForPrefix {
		t.Errorf("pPrefix.generateTempfilePath() should be %s, but: %s", expectForPrefix, filenameForPrefix)
	}
}

func TestSetTempfileWithBasename(t *testing.T) {
	var p MackerelPlugin

	expect1 := filepath.Join(os.TempDir(), "my-super-tempfile")
	p.SetTempfileByBasename("my-super-tempfile")
	if p.Tempfile != expect1 {
		t.Errorf("p.SetTempfileByBasename() should set %s, but: %s", expect1, p.Tempfile)
	}

	origDir := os.Getenv("MACKEREL_PLUGIN_WORKDIR")
	os.Setenv("MACKEREL_PLUGIN_WORKDIR", "/tmp/somewhere")
	defer os.Setenv("MACKEREL_PLUGIN_WORKDIR", origDir)

	expect2 := "/tmp/somewhere/my-great-tempfile"
	p.SetTempfileByBasename("my-great-tempfile")
	if p.Tempfile != expect2 {
		t.Errorf("p.SetTempfileByBasename() should set %s, but: %s", expect2, p.Tempfile)
	}
}

func ExamplePluginWithPrefixOutputDefinitions() {
	helper := NewMackerelPlugin(testP{})
	helper.OutputDefinitions()

	// Output:
	// # mackerel-agent-plugin
	// {"graphs":{"testP":{"label":"TestP","unit":"integer","metrics":[{"name":"bar","label":"Bar","stacked":false}]},"testP.fuga":{"label":"TestP Fuga","unit":"float","metrics":[{"name":"baz","label":"Baz","stacked":false}]}}}
}

func ExamplePluginWithPrefixOutputValues() {
	helper := NewMackerelPlugin(testP{})
	stat, _ := helper.FetchMetrics()
	key := ""
	metric := helper.GraphDefinition()[key].Metrics[0]
	var lastStat map[string]interface{}
	now := time.Unix(1437227240, 0)
	helper.formatValues(key, metric, &stat, &lastStat, now, nil)

	// Output:
	// testP.bar	15.000000	1437227240
}

func ExamplePluginWithPrefixOutputValues2() {
	helper := NewMackerelPlugin(testP{})
	stat, _ := helper.FetchMetrics()
	key := "fuga"
	metric := helper.GraphDefinition()[key].Metrics[0]
	var lastStat map[string]interface{}
	now := time.Unix(1437227240, 0)
	helper.formatValues(key, metric, &stat, &lastStat, now, nil)

	// Output:
	// testP.fuga.baz	18.000000	1437227240
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

func TestLoadLastValues(t *testing.T) {
	lastTime := time.Now().Add(-1 * time.Duration(1*time.Minute))
	stat := map[string]interface{}{
		"key1":      float64(3.2),
		"key2":      float64(4.3),
		"_lastTime": lastTime.Unix(),
	}

	tempfilePath := filepath.Join(os.TempDir(), "mackerel-plugin-test-tempfile")
	f, _ := os.Create(tempfilePath)
	json.NewEncoder(f).Encode(stat)
	f.Close()

	plugin := NewMackerelPlugin(testPHasDiff{})
	plugin.Tempfile = tempfilePath

	if err := plugin.LoadLastValues(); err != nil {
		t.Error("something went wrong")
	}

	if lastTime.Unix() != plugin.lastTime.Unix() {
		t.Errorf("lastTime unmatch: expected %s, but %s", lastTime.Unix(), plugin.lastTime.Unix())
	}

	if v, ok := plugin.LastStat["key1"]; !ok || v.(float64) != float64(3.2) {
		t.Error("saved stats does not match")
	}

	if err := plugin.LoadLastValues(); err != nil {
		t.Error("Calling LoadLastValues() multiple times should not raise error")
	}
}

func TestLoadLastValues_WithoutDiff(t *testing.T) {
	plugin := NewMackerelPlugin(testP{})

	if err := plugin.LoadLastValues(); err != nil {
		t.Error("something went wrong")
	}
}

func TestLoadLastValues_NotFound(t *testing.T) {
	tempfilePath := filepath.Join(os.TempDir(), "mackerel-plugin-test-tempfile-notfound")

	plugin := NewMackerelPlugin(testPHasDiff{})
	plugin.Tempfile = tempfilePath

	if err := plugin.LoadLastValues(); err != nil {
		t.Error("something went wrong")
	}

	if plugin.lastTime == nil {
		t.Errorf("lastTime should be set even if file not found")
	}
}

func TestLoadLastValues_ParseFailed(t *testing.T) {
	tempfilePath := filepath.Join(os.TempDir(), "mackerel-plugin-test-tempfile-broken-json")
	f, _ := os.Create(tempfilePath)
	f.WriteString(`{"this_is_broken:}`)
	f.Close()

	plugin := NewMackerelPlugin(testPHasDiff{})
	plugin.Tempfile = tempfilePath

	if err := plugin.LoadLastValues(); err == nil {
		t.Error("Error should be raised")
	}

	if plugin.lastTime == nil {
		t.Errorf("lastTime should be set even if load failed due to parse failure")
	}
}
