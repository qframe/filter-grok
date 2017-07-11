package main

import (
	"log"
	"fmt"
	"time"

	"github.com/zpatrick/go-config"
	"github.com/qnib/qframe-types"
	"github.com/qframe/filter-grok"
)

func Run(qChan qtypes.QChan, cfg *config.Config, name string) {
	p, _ := qfilter_grok.New(qChan, cfg, name)
	p.Run()
}

func main() {
	qChan := qtypes.NewQChan()
	qChan.Broadcast()
	cfgMap := map[string]string{
		"filter.test.pattern": "%{INT:number}",
		"filter.test.inputs": "test1,test2",
	}

	cfg := config.NewConfig(
		[]config.Provider{
			config.NewStatic(cfgMap),
		},
	)
	p, err := qfilter_grok.New(qChan, cfg, "test")
	if err != nil {
		log.Printf("[EE] Failed to create filter: %v", err)
		return
	}
	go p.Run()
	time.Sleep(2*time.Second)
	bg := qChan.Data.Join()
	res := []string{}
	qm := qtypes.NewQMsg("test", "test1")
	qm.Msg = "test1"
	log.Println("Send message test1")
	qChan.Data.Send(qm)
	qm2 := qtypes.NewQMsg("test", "test2")
	qm2.Msg = "test2"
	log.Println("Send message test2")
	qChan.Data.Send(qm2)
	for {
		qm = bg.Recv().(qtypes.QMsg)
		if qm.Source == "test" {
			continue
		}
		fmt.Printf("#### Received result from grok (pattern:%s) filter for input: %s\n", p.GetPattern(), qm.Msg)
		res = append(res, qm.Msg)
		for k, v := range qm.KV {
			fmt.Printf("%+15s: %s\n", k, v)
		}
		if len(res) == 2 {
			break
		}
	}
}
