package heartbeat

import (
	"log"
	"time"

	"github.com/chaos-mesh/chaosd/pkg/config"
)

var RegisterDone chan bool

func Register(conf *config.Config) {
	RegisterDone = make(chan bool)
	ticker := time.NewTicker(time.Duration(conf.HeartbeatTime) * time.Second)
	go func() {
		for {
			select {
			case <-RegisterDone:
				ticker.Stop()
				return
			case <-ticker.C:
				log.Println("HeartBeat", conf.ExportHost, conf.ExportIP, conf.ExportHost)
			}
		}
	}()
}
