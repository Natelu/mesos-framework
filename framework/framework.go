package framework

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/gogo/protobuf/proto"

	scheduler "github.com/mesos/mesos-go/api/v1/lib/scheduler"

	mesos "github.com/mesos/mesos-go/api/v1/lib"
)

// framework constructor for framework.
type Framework struct {
	Role                     *string
	Name                     string
	Url                      string
	httpClient               *http.Client
	resp                     *http.Response
	isRunning                bool
	FrameworkID              *mesos.FrameworkID
	heartbeatIntervalSeconds float64
	mesosStreamID            string
	scheduler                *Scheduler
}

func (fw *Framework) readEventLen() (int64, error) {
	lenBuf := make([]byte, 64)
	index := 0
	// fmt.Println(lenBuf)
	for true {
		_, err := fw.resp.Body.Read(lenBuf[index : index+1])
		if err != nil {
			return -1, nil
		}
		if lenBuf[index] == '\n' {
			lenBuf[index] = 0
			break
		}
		index++
		if index >= 64 {
			return -1, errors.New("out of max buffer")
		}
	}
	event_len, err := strconv.ParseInt(string(lenBuf[0:index]), 10, 64)
	if err != nil {
		return -1, err
	}
	return event_len, nil
}

func (fw *Framework) readEventBody(event_len int64) ([]byte, error) {
	buffer := make([]byte, event_len)
	_, err := fw.resp.Body.Read(buffer[0:event_len])
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func (fw *Framework) readEvent() (event *scheduler.Event, err error) {
	timer := time.AfterFunc(time.Duration(fw.heartbeatIntervalSeconds*3)*time.Second, func() {
		fw.close()
	})
	defer timer.Stop()
	eventLen, err := fw.readEventLen()
	if err != nil {
		return
	}
	eventByte, err := fw.readEventBody(eventLen)
	if err != nil {
		return
	}
	event = &scheduler.Event{}
	proto.Unmarshal(eventByte, event)

	return event, nil
}

func (fw *Framework) close() {
	fw.resp.Body.Close()
}

// connect
func (fw *Framework) connect() error {
	fw.httpClient = &http.Client{}

	user := "root"
	failoverTimeout := 10.0

	subscribe := &scheduler.Call_Subscribe{
		FrameworkInfo: &mesos.FrameworkInfo{
			Name:            fw.Name,
			User:            user,
			Role:            fw.Role,
			FailoverTimeout: &failoverTimeout,
		},
	}
	call := &scheduler.Call{
		Type:      scheduler.Call_SUBSCRIBE,
		Subscribe: subscribe,
	}
	data, err := proto.Marshal(call)
	if err != nil {
		return err
	}
	log.Println("fw.connect -> call %v", call)
	req, err := http.NewRequest("POST", "http://"+fw.Url+"/api/v1/scheduler", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Accept", "application/x-protobuf")
	fw.resp, err = fw.httpClient.Do(req)
	if err != nil {
		log.Println("fw.connect fw.resp -> err %v\n", err)
		return err
	}
	if fw.resp.StatusCode != 200 {
		return errors.New("http response status code Eooro " + strconv.Itoa(fw.resp.StatusCode))
	}
	log.Println("fw.connect -> mesos master connected ! \n")
	// return nil
	event, err := fw.readEvent()
	if err != nil {
		return err
	}
	switch event.Type {
	case scheduler.Event_SUBSCRIBED:
		fw.mesosStreamID = fw.resp.Header.Get("Mesos-Stream-Id")
		fw.FrameworkID = event.Subscribed.FrameworkID
		fw.heartbeatIntervalSeconds = *event.Subscribed.HeartbeatIntervalSeconds
		log.Println("fw.connect -> registered framework : %v", event.Subscribed.FrameworkID.Value)
		break
	default:
		err = errors.New("Error get framework id")
		break
	}
	log.Println("fw.connect -> frameworkID %v", event.Subscribed.FrameworkID)
	return err
}

func (fw *Framework) Start(name string, role string, url string) (err error) {
	fw.Name = name
	fw.Role = &role
	fw.Url = url
	fw.scheduler = &Scheduler{}
	fw.scheduler.Framework = fw
	fw.heartbeatIntervalSeconds = 10
	err = fw.connect()
	if err != nil {
		return
	}
	fw.run()
	return nil
}

func (fw *Framework) run() {
	// reConnected := false
	count := 0
	for true {
		if count > 0 {
			err := fw.loopEvents()
			if err == nil {
				log.Println("fw.run -> Mesos Framwork has stoped ")
				break
			}
			// event, err := fw.readEvent()
			// if err == nil {
			// 	log.Println("mesos framework has stoped.")
			// 	break
			// }
			// fw.scheduler.Received(event)
		}
		count++
		// reConnected = false
		// for !reConnected {
		// 	err := fw.connect()
		// 	if err == nil {
		// 		// log.Println("Mesos connection successed")
		// 		reConnected = true
		// 	} else {
		// 		time.Sleep(time.Duration(10) * time.Second)
		// 	}
		// }
	}
}

func (fw *Framework) loopEvents() (err error) {
	for true {
		event, err := fw.readEvent()
		if err != nil {
			return err
		}
		fw.scheduler.Received(event)
		//lastTime = time.Now()
	}
	return
}

func (fw *Framework) CallMaster(call interface{}) (err error) {
	callReq := &scheduler.Call{
		FrameworkID: fw.FrameworkID,
	}
	switch call.(type) {
	case scheduler.Call_Decline:
		callReq.Type = scheduler.Call_DECLINE
		callReq.Decline = call.(*scheduler.Call_Decline)
		break
	default:
		return errors.New("Unknown type " + reflect.TypeOf(call).String() + " in call master")
		break
	}
	data, err := proto.Marshal(callReq)
	if err != nil {
		return
	}
	req, err := http.NewRequest("POST", "http://"+fw.Url+"/api/v1/scheduler", bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Mesos-Stream-Id", fw.mesosStreamID)
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Accept", "application/x-protobuf")
	httpClient := http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(io.LimitReader(resp.Body, 4*1024))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if err != nil {
			return errors.New("Error read response")
		}
		return errors.New("Http response status code Error :" + strconv.Itoa(resp.StatusCode) + string(buf))
	}
	log.Println("Call response: " + string(buf))
	return
}
