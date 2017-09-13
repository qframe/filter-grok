package qfilter_grok

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"github.com/vjeantet/grok"
	"github.com/zpatrick/go-config"

	"github.com/qframe/types/plugin"
	"github.com/qframe/types/qchannel"
	"github.com/qframe/types/messages"
	"github.com/qframe/types/metrics"
	"github.com/deckarep/golang-set"

	"path/filepath"
	"os"
)

const (
	version = "0.1.12"
	pluginTyp = "filter"
	pluginPkg = "grok"
	defPatternDir = "/etc/grok-patterns"
)

type Plugin struct {
	*qtypes_plugin.Plugin
	mu 				sync.Mutex
	grok    		*grok.Grok
	expectMetric	bool
	expectJson 		bool
	isBuffered 		bool
	pattern 		string
}

func (p *Plugin) GetOverwriteKeys() []string {
	inStr, err := p.CfgString("overwrite-keys")
	if err != nil {
		inStr = ""
	}
	return strings.Split(inStr, ",")
}

func New(qChan qtypes_qchannel.QChan, cfg *config.Config, name string) (p Plugin, err error) {
	p = Plugin{
		Plugin: qtypes_plugin.NewNamedPlugin(qChan, cfg, pluginTyp, pluginPkg,  name, version),
	}
	return p, err
}

func (p *Plugin) Match(str string) (map[string]string, bool) {
	match := true
	val, _ := p.grok.Parse(p.pattern, str)
	keys := reflect.ValueOf(val).MapKeys()
	if len(keys) == 0 {
		match = false
	}
	if match {
		p.Log("trace", fmt.Sprintf("Pattern '%s' matched '%s'", p.pattern, str))
	} else {
		p.Log("trace", fmt.Sprintf("Pattern '%s' DID NOT match '%s'", p.pattern, str))
	}
	return val, match
}

func (p *Plugin) GetPattern() string {
	return p.pattern
}

func (p *Plugin) InitGrok() {
	p.grok, _ = grok.New()
	var err error
	p.pattern, err = p.CfgString("pattern")
	if err != nil {
		p.Log("fatal", "Could not find pattern in config")
	}
	pFileSet := mapset.NewSet()
	pDir, err := p.CfgString("pattern-dir")
	if err == nil && pDir != "" {
		err := filepath.Walk(pDir, func(path string, f os.FileInfo, err error) error {
			pFileSet.Add(path)
			return nil
		})
		if err != nil {
			p.Log("error", err.Error())
		}
	}
	pFiles, err := p.CfgString("pattern-files")
	for _, f := range strings.Split(pFiles, ",") {
		if f == "" {
			continue
		}
		pFileSet.Add(f)
	}
	for f := range pFileSet.Iterator().C {
		p.Log("trace", fmt.Sprintf("Iterate %s", f))
		err := p.grok.AddPatternsFromPath(f.(string))
		if err != nil {
			p.Log("error", err.Error())
		} else {
			p.Log("debug", fmt.Sprintf("Added pattern-file '%s'", f))
		}
	}
}


func (p *Plugin) sendMetric(t time.Time, tags, kv map[string]string) {
	key, kOk := kv["key"]
	if !kOk {
		p.Log("trace", "Expect to match metric, but group 'key' could not be found in pattern")
		return
	}
	valueStr, vOk := kv["value"]
	if !vOk {
		p.Log("trace", "Expect to match metric, but group 'value' could not be found in pattern")
		return
	}
	val, err := strconv.ParseFloat(valueStr,10)
	if err != nil {
		p.Log("trace", fmt.Sprintf("String '%s' could not be parsed to become a float", valueStr))
		return
	}
	tagsStr, tOk := kv["tags"]
	dimensions := tags
	if tOk {
		for _, t := range strings.Split(tagsStr, " ") {
			sl := strings.Split(t, "=")
			if len(sl) != 2 {
				p. Log("trace", fmt.Sprintf("Failed to parse tag '%s' into key=val", t))
				return
			}
			dimensions[sl[0]] = sl[1]
		}
	}
	m := qtypes_metrics.NewExt(p.Name, key, qtypes_metrics.Gauge, val, dimensions, t, p.isBuffered)
	p.Log("debug", fmt.Sprintf("Send metric: Key:%s Val:%.1f Dims:%v", m.Name, m.Value, m.Dimensions))
	p.QChan.Data.Send(m)
}

// Lock locks the plugins' mutex.
func (p *Plugin) Lock() {
	p.mu.Lock()
}

// Unlock unlocks the plugins' mutex.
func (p *Plugin) Unlock() {
	p.mu.Unlock()
}

// Run fetches everything from the Data channel and flushes it to stdout.
func (p *Plugin) Run() {
	p.Log("notice", fmt.Sprintf("Start grok filter v%s", p.Version))
	p.InitGrok()
	p.expectMetric = p.CfgBoolOr("expect-metric", false)
	p.expectJson = p.CfgBoolOr("expect-json", false)
	p.isBuffered = p.CfgBoolOr("buffer-metric", false)
	bg := p.QChan.Data.Join()
	msgKey := p.CfgStringOr("overwrite-message-key", "")
	for {
		val := bg.Recv()
		switch val.(type) {
		case qtypes_messages.ContainerMessage:
			cm := val.(qtypes_messages.ContainerMessage)
			if cm.StopProcessing(p.Plugin, false) {
				continue
			}
			cm.AppendSource(p.Name)
			var kv map[string]string
			kv, cm.SourceSuccess = p.Match(cm.Message.Message)
			if cm.SourceSuccess {
				p.Log("debug", fmt.Sprintf("Matched pattern '%s'", p.pattern))
				if p.expectMetric {
					tags := cm.Tags
					tags["container_id"] = cm.Container.ID
					tags["image"] = cm.Container.Config.Image
					tags["image_config"] = cm.Container.Image
					tags["container_name"] = strings.TrimLeft(cm.Container.Name,"/")
					p.sendMetric(cm.Time, tags, kv)
					continue
				}
				if p.expectJson {
					p.Lock()
					cm.Message.ParseJsonMap(p.Plugin, kv)
					p.Unlock()
				} else {
					for k,v := range kv {
						p.Log("debug", fmt.Sprintf("    %15s: %s", k, v))
						cm.Tags[k] = v
						if msgKey == k {
							cm.Message.Message = v
						}
					}
				}
			} else {
				p.Log("debug", fmt.Sprintf("No match of '%s' for message '%s'", p.pattern, cm.Message))
			}
			p.QChan.Data.Send(cm)
		case qtypes_messages.Message:
			qm := val.(qtypes_messages.Message)
			if qm.StopProcessing(p.Plugin, false) {
				continue
			}
			qm.AppendSource(p.Name)
			var kv map[string]string
			kv, qm.SourceSuccess = p.Match(qm.Message)
			if qm.SourceSuccess {
				p.Log("debug", fmt.Sprintf("Matched pattern '%s'", p.pattern))
				if p.expectMetric {
					p.sendMetric(qm.Time, qm.Tags, kv)
					continue
				}
				if p.expectJson {
					p.Lock()
					qm.ParseJsonMap(p.Plugin, kv)
					p.Unlock()
				}
				for k,v := range kv {
					p.Log("debug", fmt.Sprintf("    %15s: %s", k,v ))
					qm.Tags[k] = v
					if msgKey == k {
						qm.Message = v
					}
				}
			} else {
				p.Log("debug", fmt.Sprintf("No match of '%s' for message '%s'", p.pattern, qm.Message))
			}
			p.QChan.Data.Send(qm)
		default:
			p.Log("trace", fmt.Sprintf("No match for type '%s'", reflect.TypeOf(val)))
		}
	}
}
