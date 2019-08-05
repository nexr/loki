package main

import (
	"encoding/json"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/cortexproject/cortex/pkg/util/flagext"
	"github.com/go-kit/kit/log"

	//"google.golang.org/grpc"
	"net/url"

	//"github.com/cortexproject/cortex/pkg/util/flagext"
	//pb "github.com/grafana/loki/pkg/logproto"
	"github.com/grafana/loki/pkg/promtail/client"
	//"github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/r3labs/sse"
	"os"
	"strings"
	"time"

	cb "gopkg.in/cenkalti/backoff.v1"
)

var (
	loggingPrefix = ""
	exist = false
	dcosLogAPI = ""
	logger = log.NewJSONLogger(os.Stderr)
)

func main() {

	// move scrap config
	loggingPrefix, exist = os.LookupEnv("LOGGING_PREFIX")
	if !exist {
		loggingPrefix = "/"
	}

	//// move to scrap config
	dcosLogAPI, exist = os.LookupEnv("DCOS_LOG_API")
	if !exist {
		dcosLogAPI = "http://localhost:61001/system/v1/logs/v1/stream/?skip_prev=10"
	}

	_ = logger.Log("DC/OS Logging API", dcosLogAPI)
	_ = logger.Log("DC/OS Logging Prefix", loggingPrefix)

	urlString := "http://api.testloki.marathon.l4lb.thisdcos.directory:3100/api/prom/push"
	lokiUrl, _ := url.Parse(urlString)
	minBackOff,_ := time.ParseDuration("5s")
	maxBackOff,_ := time.ParseDuration("10s")
	lokiClientCfg := client.Config{
		URL: flagext.URLValue{URL: lokiUrl},
		BatchWait: 100000,
		BatchSize: 10,
		Timeout: 30 * time.Second,
		BackoffConfig: util.BackoffConfig{
			MinBackoff: minBackOff,
			MaxBackoff: maxBackOff,
			MaxRetries: 5,
		},
	}

	lokiClient,err := client.New(lokiClientCfg, logger)
	if err != nil {
		_ = logger.Log("Cannot create loki client: %s", err.Error())
		os.Exit(1)
	}

	events := make(chan *sse.Event)
	dcosLogClient := sse.NewClient(dcosLogAPI)
	dcosLogClient.ReconnectStrategy = cb.NewExponentialBackOff()
	//dcosLogClient.OnDisconnect(func(c *sse.Client){
	//	logger.Log("message", "re-subscribe the stream.")
	//	c.SubscribeChan("messages", events)
	//})
	err = dcosLogClient.SubscribeChan("messages", events)
	if err != nil {
		_ = logger.Log("message", "sse channel subscribe failed.", "level", "error")
	}

	for {
		select {
		case e := <-events:
			dat := &DcosLog{}
			if err := json.Unmarshal(e.Data, &dat); err != nil {
				logger.Log("message", "cannot parse json.", "pkg","sse")
				err = dcosLogClient.SubscribeChan("messages", events)
				if err != nil {
					_ = logger.Log("message", "sse channel subscribe failed.", "level", "error")
				}
				continue
			}
			if dat.Fields.DCOSSPACE != "" && strings.HasPrefix(dat.Fields.DCOSSPACE, loggingPrefix) {
				// generate labels for log entry
				labels := model.LabelSet{}
				labels[model.LabelName("agent")] = model.LabelValue(dat.Fields.AGENTID)
				labels[model.LabelName("container_id")] = model.LabelValue(dat.Fields.CONTAINERID)
				labels[model.LabelName("dcos_space")] = model.LabelValue(dat.Fields.DCOSSPACE)
				labels[model.LabelName("executor_id")] = model.LabelValue(dat.Fields.EXECUTORID)
				labels[model.LabelName("framework_id")] = model.LabelValue(dat.Fields.FRAMEWORKID)
				labels[model.LabelName("stream")] = model.LabelValue(dat.Fields.STREAM)
				labels[model.LabelName("syslog_identifier")] = model.LabelValue(dat.Fields.SYSLOGIDENTIFIER)
				//logger.Log("message", dat.Fields.MESSAGE, "labels", labels.String())
				err = lokiClient.Handle(labels, time.Unix(0, dat.RealtimeTimestamp*1000), dat.Fields.MESSAGE)
				if err != nil {
					_ = logger.Log("Cannot add log: %s", err.Error())
				}
			}
		}
	}

}

type DcosLog struct {
	Fields struct {
		AGENTID          string `json:"AGENT_ID"`
		CONTAINERID      string `json:"CONTAINER_ID"`
		DCOSSPACE        string `json:"DCOS_SPACE"`
		EXECUTORID       string `json:"EXECUTOR_ID"`
		FRAMEWORKID      string `json:"FRAMEWORK_ID"`
		MESSAGE          string `json:"MESSAGE"`
		STREAM           string `json:"STREAM"`
		SYSLOGIDENTIFIER string `json:"SYSLOG_IDENTIFIER"`
		SYSLOGTIMESTAMP  string `json:"@timestamp"`
	} `json:"fields"`
	Cursor             string `json:"cursor"`
	MonotonicTimestamp int64  `json:"monotonic_timestamp"`
	RealtimeTimestamp  int64  `json:"realtime_timestamp"`
}
