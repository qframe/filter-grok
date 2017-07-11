# filter-grok
Filter to apply GROK statements against a message.

## Hello World

As a standalone program like this:

```go
func main() {
	qChan := qtypes.NewQChan()
	qChan.Broadcast()
	cfgMap := map[string]string{
		"filter.test.pattern": "%{INT:number}",
		"filter.test.inputs": "test",
	}

	cfg := config.NewConfig(
		[]config.Provider{
			config.NewStatic(cfgMap),
		},
	)
	p, err := qframe_filter_grok.New(qChan, *cfg, "test")
	if err != nil {
		log.Printf("[EE] Failed to create filter: %v", err)
		return
	}
	go p.Run()
	time.Sleep(2*time.Second)
	bg := qChan.Data.Join()
	qm := qtypes.NewQMsg("test", "test")
	qm.Msg = "1"
	log.Println("Send message")
	qChan.Data.Send(qm)
	for {
		qm = bg.Recv().(qtypes.QMsg)
		if qm.Source == "test" {
			continue
		}
		fmt.Printf("#### Received result from grok (pattern:%s) filter for input: %s\n", p.GetPattern(), qm.Msg)
		for k, v := range qm.KV {
			fmt.Printf("%+15s: %s\n", k, v)
		}
		break

	}
}
```

The plugin produces the following outcome:

```bash
$ go run main.go
2017/04/14 07:28:16 [II] Dispatch broadcast for Data and Tick
2017/04/14 07:28:16 [II] Start grok filter 'test' v0.0.0
2017/04/14 07:28:18 Send message
#### Received result from grok (pattern:%{INT:number}) filter for input: 1
         number: 1
```

## Benchmark

```bash
$ go test -bench=Grok  -benchtime=5s
2017/05/20 19:38:16 [II] Dispatch broadcast for Back, Data and Tick
2017/05/20 19:38:16.627589 [  INFO]            grok Name:grok       >> Add patterns from directory '/usr/local/src/github.com/qnib/qframe-filter-grok/resources/patterns'
2017/05/20 19:38:16.631901 [NOTICE]            grok Name:grok       >> Start grok filter v0.1.10
2017/05/20 19:38:16.684517 [  INFO]            grok Name:grok       >> Benchmark sends 1 messages to grok
BenchmarkGrok-2   	2017/05/20 19:38:16.688889 [II] Dispatch broadcast for Back, Data and Tick
2017/05/20 19:38:16.696598 [  INFO]            grok Name:grok       >> Add patterns from directory '/usr/local/src/github.com/qnib/qframe-filter-grok/resources/patterns'
2017/05/20 19:38:16.700789 [NOTICE]            grok Name:grok       >> Start grok filter v0.1.10
2017/05/20 19:38:16.751433 [  INFO]            grok Name:grok       >> Benchmark sends 100 messages to grok
2017/05/20 19:38:16.761260 [II] Dispatch broadcast for Back, Data and Tick
2017/05/20 19:38:16.763537 [  INFO]            grok Name:grok       >> Add patterns from directory '/usr/local/src/github.com/qnib/qframe-filter-grok/resources/patterns'
2017/05/20 19:38:16.767631 [NOTICE]            grok Name:grok       >> Start grok filter v0.1.10
2017/05/20 19:38:16.818787 [  INFO]            grok Name:grok       >> Benchmark sends 10000 messages to grok
2017/05/20 19:38:17.287998 [II] Dispatch broadcast for Back, Data and Tick
2017/05/20 19:38:17.290940 [  INFO]            grok Name:grok       >> Add patterns from directory '/usr/local/src/github.com/qnib/qframe-filter-grok/resources/patterns'
2017/05/20 19:38:17.295367 [NOTICE]            grok Name:grok       >> Start grok filter v0.1.10
2017/05/20 19:38:17.346071 [  INFO]            grok Name:grok       >> Benchmark sends 200000 messages to grok
  200000	     40593 ns/op
PASS
ok  	github.com/qnib/qframe-filter-grok	8.979s
```
