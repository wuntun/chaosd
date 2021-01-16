package heartbeat

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/chaos-mesh/chaosd/pkg/config"
)

type Node struct {
	// kind means the node's kind, the value can be k8s or physic
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	Config string `json:"config"`
}

var RegisterDone chan bool

func Register(conf *config.Config) {
	// RegisterDone = make(chan bool)

	nodeInfo := &Node{
		Kind:   "physic",
		Name:   conf.ExportHost,
		Config: fmt.Sprintf("http://%s:%d", conf.ExportIP, conf.ExportPort),
	}
	nodeInfoJSON, _ := json.Marshal(nodeInfo)
	respData := HTTPPOST(conf.HeartbeatAddr, string(nodeInfoJSON))
	respDataJSON, _ := json.Marshal(respData)
	log.Println(string(respDataJSON))

	// ticker := time.NewTicker(time.Duration(conf.HeartbeatTime) * time.Second)
	// go func() {
	// 	for {
	// 		select {
	// 		case <-RegisterDone:
	// 			ticker.Stop()
	// 			return
	// 		case <-ticker.C:
	// 			nodeInfo := &Node{
	// 				Kind:   "physic",
	// 				Name:   conf.ExportHost,
	// 				Config: fmt.Sprintf("http://%s:%d", conf.ExportIP, conf.ExportPort),
	// 			}
	// 			nodeInfoJSON, _ := json.Marshal(nodeInfo)
	// 			respData := HTTPPOST(conf.HeartbeatAddr, string(nodeInfoJSON))
	// 			respDataJSON, _ := json.Marshal(respData)
	// 			log.Println(string(respDataJSON))
	// 		}
	// 	}
	// }()
}

func HTTPPOST(reqURL, reqData string) map[string]interface{} {
	log.Println(reqURL, reqData)
	req, _ := http.NewRequest("POST", reqURL, strings.NewReader(reqData))
	req.Header.Add("Content-Type", "application/json")
	c := &http.Client{
		Timeout: 10 * time.Second,
	}
	res, perr := c.Do(req)
	if perr != nil {
		log.Println(perr)
		return nil
	}
	resBody, berr := ioutil.ReadAll(res.Body)
	_ = res.Body.Close()
	if berr != nil {
		log.Println(berr)
		return nil
	}
	responeDate := make(map[string]interface{})
	_ = json.Unmarshal(resBody, &responeDate)
	return responeDate
}
