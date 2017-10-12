package mackerelplugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mackerelio/golib/pluginutil"
)

// Metrics represents definition of a metric
type Metrics struct {
	Name         string  `json:"name"`
	Label        string  `json:"label"`
	Diff         bool    `json:"-"`
	Type         string  `json:"-"`
	Stacked      bool    `json:"stacked"`
	Scale        float64 `json:"-"`
	AbsoluteName bool    `json:"-"`
}

// Graphs represents definition of a graph
type Graphs struct {
	Label   string    `json:"label"`
	Unit    string    `json:"unit"`
	Metrics []Metrics `json:"metrics"`
}

// MetricValues represents a collection of metric values and its timestamp
type MetricValues struct {
	Values    map[string]interface{}
	Timestamp time.Time
}

// Plugin is old interface of mackerel-plugin
type Plugin interface {
	FetchMetrics() (map[string]interface{}, error)
	GraphDefinition() map[string]Graphs
}

// PluginWithPrefix is recommended interface
type PluginWithPrefix interface {
	Plugin
	MetricKeyPrefix() string
}

// MackerelPlugin is for mackerel-agent-plugin
type MackerelPlugin struct {
	Plugin
	Tempfile string
	diff     *bool
}

// NewMackerelPlugin returns new MackerelPlugin struct
func NewMackerelPlugin(plugin Plugin) MackerelPlugin {
	mp := MackerelPlugin{Plugin: plugin}
	return mp
}

func (h *MackerelPlugin) hasDiff() bool {
	if h.diff == nil {
		diff := false
		h.diff = &diff
	DiffCheck:
		for _, graph := range h.GraphDefinition() {
			for _, metric := range graph.Metrics {
				if metric.Diff {
					*h.diff = true
					break DiffCheck
				}
			}
		}
	}
	return *h.diff
}

func (h *MackerelPlugin) printValue(w io.Writer, key string, value interface{}, now time.Time) {
	switch value.(type) {
	case uint32:
		fmt.Fprintf(w, "%s\t%d\t%d\n", key, value.(uint32), now.Unix())
	case uint64:
		fmt.Fprintf(w, "%s\t%d\t%d\n", key, value.(uint64), now.Unix())
	case float64:
		if math.IsNaN(value.(float64)) || math.IsInf(value.(float64), 0) {
			log.Printf("Invalid value: key = %s, value = %f\n", key, value)
		} else {
			fmt.Fprintf(w, "%s\t%f\t%d\n", key, value.(float64), now.Unix())
		}
	}
}

func (h *MackerelPlugin) FetchLastValues() (metricValues MetricValues, err error) {
	if !h.hasDiff() {
		return
	}
	metricValues.Timestamp = time.Now()

	f, err := os.Open(h.tempfilename())
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		return
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&metricValues.Values)
	switch metricValues.Values["_lastTime"].(type) {
	case float64:
		metricValues.Timestamp = time.Unix(int64(metricValues.Values["_lastTime"].(float64)), 0)
	case int64:
		metricValues.Timestamp = time.Unix(metricValues.Values["_lastTime"].(int64), 0)
	}
	if err != nil {
		return
	}
	return
}

func (h *MackerelPlugin) saveValues(values map[string]interface{}, now time.Time) error {
	if !h.hasDiff() {
		return nil
	}
	fname := h.tempfilename()
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()

	values["_lastTime"] = now.Unix()
	encoder := json.NewEncoder(f)
	err = encoder.Encode(values)
	if err != nil {
		return err
	}

	return nil
}

func (h *MackerelPlugin) calcDiff(value float64, now time.Time, lastValue float64, lastTime time.Time) (float64, error) {
	diffTime := now.Unix() - lastTime.Unix()
	if diffTime > 600 {
		return 0, errors.New("Too long duration")
	}

	diff := (value - lastValue) * 60 / float64(diffTime)

	if lastValue <= value {
		return diff, nil
	}
	return 0.0, errors.New("Counter seems to be reset.")
}

func (h *MackerelPlugin) calcDiffUint32(value uint32, now time.Time, lastValue uint32, lastTime time.Time, lastDiff float64) (float64, error) {
	diffTime := now.Unix() - lastTime.Unix()
	if diffTime > 600 {
		return 0, errors.New("Too long duration")
	}

	diff := float64((value-lastValue)*60) / float64(diffTime)

	if lastValue <= value || diff < lastDiff*10 {
		return diff, nil
	}
	return 0.0, errors.New("Counter seems to be reset.")

}

func (h *MackerelPlugin) calcDiffUint64(value uint64, now time.Time, lastValue uint64, lastTime time.Time, lastDiff float64) (float64, error) {
	diffTime := now.Unix() - lastTime.Unix()
	if diffTime > 600 {
		return 0, errors.New("Too long duration")
	}

	diff := float64((value-lastValue)*60) / float64(diffTime)

	if lastValue <= value || diff < lastDiff*10 {
		return diff, nil
	}
	return 0.0, errors.New("Counter seems to be reset.")
}

func (h *MackerelPlugin) tempfilename() string {
	if h.Tempfile == "" {
		h.Tempfile = h.generateTempfilePath(os.Args[0])
	}
	return h.Tempfile
}

var tempfileSanitizeReg = regexp.MustCompile(`[^A-Za-z0-9_.-]`)

// SetTempfileByBasename sets Tempfile under proper directory with specified basename.
func (h *MackerelPlugin) SetTempfileByBasename(base string) {
	h.Tempfile = filepath.Join(pluginutil.PluginWorkDir(), base)
}

func (h *MackerelPlugin) generateTempfilePath(path string) string {
	var prefix string
	if p, ok := h.Plugin.(PluginWithPrefix); ok {
		prefix = p.MetricKeyPrefix()
	} else {
		name := filepath.Base(path)
		prefix = strings.TrimPrefix(tempfileSanitizeReg.ReplaceAllString(name, "_"), "mackerel-plugin-")
	}
	filename := fmt.Sprintf("mackerel-plugin-%s", prefix)
	return filepath.Join(pluginutil.PluginWorkDir(), filename)
}

func (h *MackerelPlugin) formatValues(prefix string, metric Metrics, metricValues MetricValues, lastMetricValues MetricValues) {
	name := metric.Name
	if metric.AbsoluteName && len(prefix) > 0 {
		name = prefix + "." + name
	}
	value, ok := metricValues.Values[name]
	if !ok || value == nil {
		return
	}

	switch value.(type) {
	case string:
		switch metric.Type {
		case "uint32":
			value, _ = strconv.ParseUint(value.(string), 10, 32)
		case "uint64":
			value, _ = strconv.ParseUint(value.(string), 10, 64)
		default:
			value, _ = strconv.ParseFloat(value.(string), 64)
		}
	}

	if metric.Diff {
		_, ok := lastMetricValues.Values[name]
		if ok {
			var lastDiff float64
			if lastMetricValues.Values[".last_diff."+name] != nil {
				lastDiff = toFloat64(lastMetricValues.Values[".last_diff."+name])
			}
			var err error
			switch metric.Type {
			case "uint32":
				value, err = h.calcDiffUint32(toUint32(value), metricValues.Timestamp, toUint32(lastMetricValues.Values[name]), lastMetricValues.Timestamp, lastDiff)
			case "uint64":
				value, err = h.calcDiffUint64(toUint64(value), metricValues.Timestamp, toUint64(lastMetricValues.Values[name]), lastMetricValues.Timestamp, lastDiff)
			default:
				value, err = h.calcDiff(toFloat64(value), metricValues.Timestamp, toFloat64(lastMetricValues.Values[name]), lastMetricValues.Timestamp)
			}
			if err != nil {
				log.Println("OutputValues: ", err)
				return
			}
			metricValues.Values[".last_diff."+name] = value
		} else {
			log.Printf("%s does not exist at last fetch\n", name)
			return
		}
	}

	if metric.Scale != 0 {
		switch metric.Type {
		case "uint32":
			value = toUint32(value) * uint32(metric.Scale)
		case "uint64":
			value = toUint64(value) * uint64(metric.Scale)
		default:
			value = toFloat64(value) * metric.Scale
		}
	}

	metricNames := []string{}
	if p, ok := h.Plugin.(PluginWithPrefix); ok {
		metricNames = append(metricNames, p.MetricKeyPrefix())
	}
	if len(prefix) > 0 {
		metricNames = append(metricNames, prefix)
	}
	metricNames = append(metricNames, metric.Name)
	h.printValue(os.Stdout, strings.Join(metricNames, "."), value, metricValues.Timestamp)
}

func (h *MackerelPlugin) formatValuesWithWildcard(prefix string, metric Metrics, metricValues MetricValues, lastMetricValues MetricValues) {
	regexpStr := `\A` + prefix + "." + metric.Name
	regexpStr = strings.Replace(regexpStr, ".", "\\.", -1)
	regexpStr = strings.Replace(regexpStr, "*", "[-a-zA-Z0-9_]+", -1)
	regexpStr = strings.Replace(regexpStr, "#", "[-a-zA-Z0-9_]+", -1)
	re, err := regexp.Compile(regexpStr)
	if err != nil {
		log.Fatalln("Failed to compile regexp: ", err)
	}
	for k := range metricValues.Values {
		if re.MatchString(k) {
			metricEach := metric
			metricEach.Name = k
			h.formatValues("", metricEach, metricValues, lastMetricValues)
		}
	}
}

// Run the plugin
func (h *MackerelPlugin) Run() {
	if os.Getenv("MACKEREL_AGENT_PLUGIN_META") != "" {
		h.OutputDefinitions()
	} else {
		h.OutputValues()
	}
}

// OutputValues output the metrics
func (h *MackerelPlugin) OutputValues() {
	now := time.Now()
	stat, err := h.FetchMetrics()
	if err != nil {
		log.Fatalln("OutputValues: ", err)
	}

	lastMetricValues, err := h.FetchLastValues()
	if err != nil {
		log.Println("FetchLastValues (ignore):", err)
	}

	for key, graph := range h.GraphDefinition() {
		for _, metric := range graph.Metrics {
			if strings.ContainsAny(key+metric.Name, "*#") {
				h.formatValuesWithWildcard(key, metric, MetricValues{Values: stat, Timestamp: now}, lastMetricValues)
			} else {
				h.formatValues(key, metric, MetricValues{Values: stat, Timestamp: now}, lastMetricValues)
			}
		}
	}

	err = h.saveValues(stat, now)
	if err != nil {
		log.Fatalln("saveValues: ", err)
	}
}

// GraphDef represents graph definitions
type GraphDef struct {
	Graphs map[string]Graphs `json:"graphs"`
}

func title(s string) string {
	r := strings.NewReplacer(".", " ", "_", " ")
	return strings.Title(r.Replace(s))
}

// OutputDefinitions outputs graph definitions
func (h *MackerelPlugin) OutputDefinitions() {
	fmt.Println("# mackerel-agent-plugin")
	graphs := make(map[string]Graphs)
	for key, graph := range h.GraphDefinition() {
		g := graph
		k := key
		if p, ok := h.Plugin.(PluginWithPrefix); ok {
			prefix := p.MetricKeyPrefix()
			if k == "" {
				k = prefix
			} else {
				k = prefix + "." + k
			}
		}
		if g.Label == "" {
			g.Label = title(k)
		}
		metrics := []Metrics{}
		for _, v := range g.Metrics {
			if v.Label == "" {
				v.Label = title(v.Name)
			}
			metrics = append(metrics, v)
		}
		g.Metrics = metrics
		graphs[k] = g
	}
	var graphdef GraphDef
	graphdef.Graphs = graphs
	b, err := json.Marshal(graphdef)
	if err != nil {
		log.Fatalln("OutputDefinitions: ", err)
	}
	fmt.Println(string(b))
}

func toUint32(value interface{}) uint32 {
	var ret uint32
	switch value.(type) {
	case uint32:
		ret = value.(uint32)
	case uint64:
		ret = uint32(value.(uint64))
	case float64:
		ret = uint32(value.(float64))
	case string:
		v, err := strconv.ParseUint(value.(string), 10, 32)
		if err == nil {
			ret = uint32(v)
		}
	}
	return ret
}

func toUint64(value interface{}) uint64 {
	var ret uint64
	switch value.(type) {
	case uint32:
		ret = uint64(value.(uint32))
	case uint64:
		ret = value.(uint64)
	case float64:
		ret = uint64(value.(float64))
	case string:
		ret, _ = strconv.ParseUint(value.(string), 10, 64)
	}
	return ret
}

func toFloat64(value interface{}) float64 {
	var ret float64
	switch value.(type) {
	case uint32:
		ret = float64(value.(uint32))
	case uint64:
		ret = float64(value.(uint64))
	case float64:
		ret = value.(float64)
	case string:
		ret, _ = strconv.ParseFloat(value.(string), 64)
	}
	return ret
}
