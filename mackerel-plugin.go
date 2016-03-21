package mackerelplugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Metrics struct {
	Name    string  `json:"name"`
	Label   string  `json:"label"`
	Diff    bool    `json:"-"`
	Type    string  `json:"type"`
	Stacked bool    `json:"stacked"`
	Scale   float64 `json:"scale"`
}

type Graphs struct {
	Label   string    `json:"label"`
	Unit    string    `json:"unit"`
	Metrics []Metrics `json:"metrics"`
}

type Plugin interface {
	FetchMetrics() (map[string]interface{}, error)
	GraphDefinition() map[string]Graphs
}

type PluginWithPrefix interface {
	Plugin
	GetMetricKeyPrefix() string
}

type MackerelPlugin struct {
	Plugin
	Tempfile string
	diff     *bool
}

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

	f, err := os.Open(h.Tempfilename())
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
	f, err := os.Create(h.Tempfilename())
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

func (h *MackerelPlugin) Tempfilename() string {
	if h.Tempfile == "" {
		prefix := "default"
		if p, ok := h.Plugin.(PluginWithPrefix); ok {
			prefix = p.GetMetricKeyPrefix()
		}
		h.Tempfile = fmt.Sprintf("/tmp/mackerel-plugin-%s", prefix)
	}
	return h.Tempfile
}

func (h *MackerelPlugin) formatValues(prefix string, metric Metrics, stat *map[string]interface{}, lastStat *map[string]interface{}, now time.Time, lastTime time.Time) {
	value, ok := (*stat)[metric.Name]
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
		_, ok := (*lastStat)[metric.Name]
		if ok {
			var lastDiff float64
			if (*lastStat)[".last_diff."+metric.Name] != nil {
				lastDiff = toFloat64((*lastStat)[".last_diff."+metric.Name])
			}
			var err error
			switch metric.Type {
			case "uint32":
				value, err = h.calcDiffUint32(toUint32(value), now, toUint32((*lastStat)[metric.Name]), lastTime, lastDiff)
			case "uint64":
				value, err = h.calcDiffUint64(toUint64(value), now, toUint64((*lastStat)[metric.Name]), lastTime, lastDiff)
			default:
				value, err = h.calcDiff(toFloat64(value), now, toFloat64((*lastStat)[metric.Name]), lastTime)
			}
			if err != nil {
				log.Println("OutputValues: ", err)
				return
			} else {
				(*stat)[".last_diff."+metric.Name] = value
			}
		} else {
			log.Printf("%s does not exist at last fetch\n", metric.Name)
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
		metricNames = append(metricNames, p.GetMetricKeyPrefix())
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
	for k, _ := range *stat {
		if re.MatchString(k) {
			metricEach := metric
			metricEach.Name = k
			h.formatValues("", metricEach, stat, lastStat, now, lastTime)
		}
	}
}

func (h *MackerelPlugin) Run() {
	if os.Getenv("MACKEREL_AGENT_PLUGIN_META") != "" {
		h.OutputDefinitions()
	} else {
		h.OutputValues()
	}
}

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
		log.Fatalf("saveValues: ", err)
	}
}

type GraphDef struct {
	Graphs map[string]Graphs `json:"graphs"`
}

func (h *MackerelPlugin) OutputDefinitions() {
	fmt.Println("# mackerel-agent-plugin")
	graphs := make(map[string]Graphs)
	for key, graph := range h.GraphDefinition() {
		k := key
		if p, ok := h.Plugin.(PluginWithPrefix); ok {
			prefix := p.GetMetricKeyPrefix()
			if k == "" {
				k = prefix
			} else {
				k = prefix + "." + k
			}
		}
		graphs[k] = graph
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
