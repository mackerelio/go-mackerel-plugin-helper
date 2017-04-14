go-mackerel-plugin-helper [![Build Status](https://travis-ci.org/mackerelio/go-mackerel-plugin-helper.svg?branch=master)](https://travis-ci.org/mackerelio/go-mackerel-plugin)
==================

This package provides helper methods to create mackerel agent plugin easily.

How to use
==========

## Graph Definition

A plugin can specify `Graphs` and `Metrics`.
`Graphs` represents one graph and includes some `Metrics`s which represent each line.

`Graphs` includes followings:

- `Label`: Label for the graph
- `Unit`: Unit for lines, `integer`, `float` can be specified.
- `Metrics`: Array of `Metrics` which represents each line.

`Metics` includes followings:

- `Name`: Key of the line
- `Label`: Label of the line
- `Diff`: If `Diff` is true, differential is used as value.
- `Type`: 'float64', 'uint64' or 'uint32' can be specified. Default is `float64`
- `Stacked`: If `Stacked` is true, the line is stacked.
- `Scale`: Each value is multiplied by `Scale`.

```go
var graphdef = map[string](mackerelplugin.Graphs){
	"memcached.cmd": {
		Label: "Memcached Command",
		Unit:  "integer",
		Metrics: [](mackerelplugin.Metrics){
			{Name: "cmd_get", Label: "Get", Diff: true, Type: "uint64"},
			{Name: "cmd_set", Label: "Set", Diff: true, Type: "uint64"},
			{Name: "cmd_flush", Label: "Flush", Diff: true, Type: "uint64"},
			{Name: "cmd_touch", Label: "Touch", Diff: true, Type: "uint64"},
		},
	},
}
```

### Calculate Differential of Counter

Many status values of popular middle-wares are provided as counter.
But current Mackerel API can accept only absolute values, so differential values must be calculated beside agent plugins.

`Diff` of `Metrics` is a flag whether values must be treated as counter or not.
If this flag is set, this package calculate differential values automatically with current values and previous values, which are saved to a temporally file.

### Adjust Scale Value

Some status values such as `jstat` memory usage are provided as scaled values.
For example, `OGC` value are provided KB scale.

`Scale` of `Metrics` is a multiplier for adjustment of the scale values.

```go
var graphdef = map[string](mackerelplugin.Graphs){
    "jvm.old_space": {
        Label: "JVM Old Space memory",
        Unit:  "float",
        Metrics: [](mackerelplugin.Metrics){
            {Name: "OGCMX", Label: "Old max", Diff: false, Scale: 1024},
            {Name: "OGC", Label: "Old current", Diff: false, Scale: 1024},
            {Name: "OU", Label: "Old used", Diff: false, Scale: 1024},
        },
    },
}
```

### Deal with counter overflow

If `Type` of metrics is `uint64` or `uint32` and `Diff` is true, the helper check counter overflow.
When differential value is negative, overflow or counter reset may be occurred.
If the differential value is ten-times above last value, the helper judge this is counter reset, not counter overflow, then the helper set value is unknown. If not, the helper recognizes counter overflow occurred.

## Tempfile

`MackerelPlugin` interface has `Tempfile` field. The tempfile is used to calc differences in metrics with `Diff: true`.
If this field is kept empty, default Tempfile will be placed under `os.Getenv("MACKEREL_PLUGIN_WORKDIR")` or `os.Tempdir()` and its filename will be generated from plugin filename.
mackerel-agent's plugin should place its Tempfile under `os.Getenv("MACKEREL_PLUGIN_WORKDIR")` unless specified explicitly. If plugin wants to set default Tempfile filename by itself, use `MackerelPlugin.GenerateTempfilePathWithBase()`.

## Method

A plugin must implement this interface and the `main` method.

```go
type PluginWithPrefix interface {
	FetchMetrics() (map[string]interface{}, error)
	GraphDefinition() map[string]Graphs
	MetricKeyPrefix() string
}
```

```go
func main() {
	optHost := flag.String("host", "localhost", "Hostname")
	optPort := flag.String("port", "11211", "Port")
	optTempfile := flag.String("tempfile", "", "Temp file name")
    optMetricKeyPrefix := flag.String("metric-key-prefix", "memcached", "Metric Key Prefix")
	flag.Parse()

	var memcached MemcachedPlugin

	memcached.Target = fmt.Sprintf("%s:%s", *optHost, *optPort)
    memcached.prefix = *optMetricKeyPrefix
	helper := mackerelplugin.NewMackerelPlugin(memcached)
	helper.Tempfile = *optTempfile

	helper.Run()
}
```

### old `Plugin` interface

`Plugin` interface is old one. `PluginWithPrefix` interface is recommended now.
