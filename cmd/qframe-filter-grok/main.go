package main

import (
	"log"
	"time"

	"github.com/zpatrick/go-config"
	"github.com/qframe/filter-grok"
	"github.com/qframe/types/qchannel"
	"github.com/qframe/types/messages"
	"os"
)

type Test struct {
	Str string
	Success bool
	result map[string]string
}


var (
	/*ip4_1 = Test{"192.168.0.1", true, map[string]string{"ip":"192.168.0.1"}}
	ip4_2 = Test{"127.0.0.1", true, map[string]string{"ip":"127.0.0.1"}}
	ip6_1 = Test{":::1", true, map[string]string{"ip":":::1"}}
	ip6_2 = Test{"2001:0db8:85a3:0000:0000:8a2e:0370:7334", true, map[string]string{"ip":"2001:0db8:85a3:0000:0000:8a2e:0370:7334"}}
	ipCase = map[string]Test{
		"ip4_1":ip4_1,
		"ip4_2":ip4_2,
		"ip6_1":ip6_1,
		"ip6_2":ip6_2,
	}*/
	int_1 = Test{"test1", true, map[string]string{"num":"1"}}
	intCase = map[string]Test{
		"int_1":int_1,
	}
)

func Run(qChan qtypes_qchannel.QChan, cfg *config.Config, name string) {
	p, _ := qfilter_grok.New(qChan, cfg, name)
	p.Run()
}

func main() {
	qChan := qtypes_qchannel.NewQChan()
	qChan.Broadcast()
	cfgMap := map[string]string{
		"log.level": "trace",
		"filter.grok.pattern": "test%{INT:num}",
		"filter.grok.pattern-dir": "./patterns/",
		"filter.grok.inputs": "test",
	}
	cfg := config.NewConfig(
		[]config.Provider{
			config.NewStatic(cfgMap),
		},
	)
	p, err := qfilter_grok.New(qChan, cfg, "grok")
	if err != nil {
		log.Printf("[EE] Failed to create filter: %v", err)
		return
	}
	go p.Run()
	time.Sleep(time.Duration(100)*time.Millisecond)
	ticker := time.NewTicker(time.Millisecond*time.Duration(2000)).C
	bg := qChan.Data.Join()
	res := []string{}
	for _, c := range intCase {
		b := qtypes_messages.NewBase("test")
		qm := qtypes_messages.NewMessage(b, c.Str)
		log.Printf("Send message '%s", qm.Message)
		qChan.Data.Send(qm)
	}
	for {
		select {
		case val := <- bg.Read:
			switch val.(type) {
			case qtypes_messages.Message:
				qm := val.(qtypes_messages.Message)
				if ! qm.InputsMatch([]string{"grok"}) {
					continue
				}
				res = append(res, qm.GetLastSource())
				log.Printf("#### Received result from grok (pattern:%s) filter for input: %s\n", p.GetPattern(), qm.Message)
			}
			if len(res) == len(intCase) {
				break
			}
		case <- ticker:
			log.Println("Ticker came along, time's up...")
			os.Exit(1)
		}
		if len(res) == len(intCase) {
			break
		}
	}
	os.Exit(0)
}
