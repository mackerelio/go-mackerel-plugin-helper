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

func (h *MackerelPlugin) fetchLastValues() (map[string]interface{}, time.Time, error) {
	if !h.hasDiff() {
		return nil, time.Unix(0, 0), nil
	}
	lastTime := time.Now()

	f, err := os.Open(h.tempfilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, lastTime, nil
		}
		return nil, lastTime, err
	}
	defer f.Close()

	stat := make(map[string]interface{})
	decoder := json.NewDecoder(f)
	err = decoder.Decode(&stat)
	switch stat["_lastTime"].(type) {
	case float64:
		lastTime = time.Unix(int64(stat["_lastTime"].(float64)), 0)
	case int64:
		lastTime = time.Unix(stat["_lastTime"].(int64), 0)
	}
	if err != nil {
		return stat, lastTime, err
	}
	return stat, lastTime, nil
}

func (h *MackerelPlugin) saveValues(values map[string]interface{}, now time.Time) error {
	if !h.hasDiff() {
		return nil
	}
	fname := h.tempfilePath()
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

func (h *MackerelPlugin) tempfilePath() string {
	if h.Tempfile == "" {
		h.Tempfile = h.generateTempfilePath(os.Args[0])
	}
	return h.Tempfile
}

func (h *MackerelPlugin) generateTempfilePath(path string) string {
	var prefix string
	if p, ok := h.Plugin.(PluginWithPrefix); ok {
		prefix = p.MetricKeyPrefix()
	} else {
		name := filepath.Base(path)
		var sanitizeReg = regexp.MustCompile(`[^A-Za-z0-9_.-]`)
		prefix = sanitizeReg.ReplaceAllString(name, "_")
	}
	filename := fmt.Sprintf("mackerel-plugin-%s", prefix)
	dir := os.Getenv("MACKEREL_PLUGIN_WORKDIR")
	if dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, filename)
}

func (h *MackerelPlugin) formatValues(prefix string, metric Metrics, stat *map[string]interface{}, lastStat *map[string]interface{}, now time.Time, lastTime time.Time) {
	name := metric.Name
	if metric.AbsoluteName && len(prefix) > 0 {
		name = prefix + "." + name
	}
	value, ok := (*stat)[name]
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
		_, ok := (*lastStat)[name]
		if ok {
			var lastDiff float64
			if (*lastStat)[".last_diff."+name] != nil {
				lastDiff = toFloat64((*lastStat)[".last_diff."+name])
			}
			var err error
			switch metric.Type {
			case "uint32":
				value, err = h.calcDiffUint32(toUint32(value), now, toUint32((*lastStat)[name]), lastTime, lastDiff)
			case "uint64":
				value, err = h.calcDiffUint64(toUint64(value), now, toUint64((*lastStat)[name]), lastTime, lastDiff)
			default:
				value, err = h.calcDiff(toFloat64(value), now, toFloat64((*lastStat)[name]), lastTime)
			}
			if err != nil {
				log.Println("OutputValues: ", err)
				return
			}
			(*stat)[".last_diff."+name] = value
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
	h.printValue(os.Stdout, strings.Join(metricNames, "."), value, now)
}

func (h *MackerelPlugin) formatValuesWithWildcard(prefix string, metric Metrics, stat *map[string]interface{}, lastStat *map[string]interface{}, now time.Time, lastTime time.Time) {
	regexpStr := `\A` + prefix + "." + metric.Name
	regexpStr = strings.Replace(regexpStr, ".", "\\.", -1)
	regexpStr = strings.Replace(regexpStr, "*", "[-a-zA-Z0-9_]+", -1)
	regexpStr = strings.Replace(regexpStr, "#", "[-a-zA-Z0-9_]+", -1)
	re, err := regexp.Compile(regexpStr)
	if err != nil {
		log.Fatalln("Failed to compile regexp: ", err)
	}
	for k := range *stat {
		if re.MatchString(k) {
			metricEach := metric
			metricEach.Name = k
			h.formatValues("", metricEach, stat, lastStat, now, lastTime)
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

	lastStat, lastTime, err := h.fetchLastValues()
	if err != nil {
		log.Println("fetchLastValues (ignore):", err)
	}

	for key, graph := range h.GraphDefinition() {
		for _, metric := range graph.Metrics {
			if strings.ContainsAny(key+metric.Name, "*#") {
				h.formatValuesWithWildcard(key, metric, &stat, &lastStat, now, lastTime)
			} else {
				h.formatValues(key, metric, &stat, &lastStat, now, lastTime)
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
