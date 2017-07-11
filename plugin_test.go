package qfilter_grok

import (
	"fmt"
	"testing"
	"time"
	"log"
	"github.com/zpatrick/go-config"

	"github.com/qnib/qframe-types"
	"github.com/stretchr/testify/assert"
	"reflect"
)


/******* Tests */
func TestPlugin_Match(t *testing.T) {
	cfgMap := map[string]string{
		"log.level": "error",
		"filter.grok.pattern": "test%{INT:number}",
		"filter.grok.inputs": "test",
		"filter.grok.pattern-dir": "./resources/patterns/",
	}
	cfg := config.NewConfig([]config.Provider{config.NewStatic(cfgMap)})
	qChan := qtypes.NewCfgQChan(cfg)
	// simple
	p, err := New(qChan, cfg, "grok")
	assert.NoError(t, err)
	p.InitGrok()
	got, ok := p.Match("test1")
	assert.True(t, ok, "test1 should match pattern")
	assert.Equal(t, map[string]string{"number": "1"}, got)
	cfgMap["filter.grok.pattern"] = "test%{INT:number} %{WORD:str}"
	cfg = config.NewConfig([]config.Provider{config.NewStatic(cfgMap)})
	p1, err := New(qChan, cfg, "grok")
	assert.NoError(t, err)
	p1.InitGrok()
	g1, ok1 := p1.Match("test1 sometext")
	assert.True(t, ok1, "test1 should match pattern")
	assert.Equal(t, map[string]string{"number": "1", "str": "sometext"}, g1)

}

/******* Benchmarks */
func Receive(qchan qtypes.QChan, source string, endCnt int) {
	bg := qchan.Data.Join()
	allCnt := 1
	cnt := 1
	for {
		select {
		case val := <-bg.Read:
			allCnt++
			switch val.(type) {
			case qtypes.Message:
				qm := val.(qtypes.Message)
				if qm.IsLastSource(source) {
					cnt++
				}
			default:
				fmt.Printf("received msg %d: type=%s\n", allCnt, reflect.TypeOf(val))

			}
		}
		if endCnt == cnt {
			qchan.Data.Send(cnt)
			break
		}
	}
}

func BenchmarkGrok(b *testing.B) {
	endCnt := b.N
	cfgMap := map[string]string{
		"log.level": "error",
		"filter.grok.pattern": "test%{INT:number}",
		"filter.grok.inputs": "test",
		"filter.grok.pattern-dir": "./resources/patterns/",
	}
	cfg := config.NewConfig([]config.Provider{config.NewStatic(cfgMap)})
	qChan := qtypes.NewCfgQChan(cfg)
	qChan.Broadcast()
	go Receive(qChan, "grok", endCnt)
	p, err := New(qChan, cfg, "grok")
	if err != nil {
		log.Printf("[EE] Failed to create filter: %v", err)
		return
	}
	dc := qChan.Data.Join()
	go p.Run()
	time.Sleep(time.Duration(50)*time.Millisecond)
	p.Log("info", fmt.Sprintf("Benchmark sends %d messages to grok", endCnt))
	qm := qtypes.NewMessage(qtypes.NewBase("test"), "test", "testMsg", "none")
	for i := 1; i <= endCnt; i++ {
		msg := fmt.Sprintf("test%d", i)
		qm.Message = msg
		qChan.Data.Send(qm)
	}
	done := false
	for {
		select {
		case val := <- dc.Read:
			switch val.(type) {
			case int:
				vali := val.(int)
				assert.Equal(b, endCnt, vali)
				done = true
			}
		case <-time.After(5 * time.Second):
				b.Fatal("metrics receive timeout")
		}
		if done {
			break
		}
	}
}


