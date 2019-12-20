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

	"github.com/golang/protobuf/proto"
	"github.com/uuid"

	executor "github.com/mesos/mesos-go/api/v1/lib/executor"

	mesos "github.com/mesos/mesos-go/api/v1/lib"
)

type ComputeExecutor struct {
	url                      string
	httpClient               *http.Client
	resp                     *http.Response
	isRunning                bool
	heartbeatIntervalSeconds float64
	ExecutorInfo             mesos.ExecutorInfo
	ExecId                   mesos.ExecutorID
	FWId                     mesos.FrameworkID
	unackedTasks             map[mesos.TaskID]mesos.TaskInfo
	unackedUpdates           map[string]executor.Call_Update
	failedTask               map[mesos.TaskID]mesos.TaskStatus
}

func (exec *ComputeExecutor) readEventLen() (eventLen int64, err error) {
	lenBuf := make([]byte, 64)
	index := 0
	for true {
		_, err := exec.resp.Body.Read(lenBuf[index : index+1])
		if err != nil {
			return -1, err
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

func (exec *ComputeExecutor) readEventBody(eventLen int64) ([]byte, error) {
	buffer := make([]byte, eventLen)
	_, err := exec.resp.Body.Read(buffer[0:eventLen])
	if err != nil {
		return nil, err
	}
	return buffer, nil
}

func (exec *ComputeExecutor) readEvent() (event *executor.Event, err error) {
	timer := time.AfterFunc(time.Duration(exec.heartbeatIntervalSeconds*3)*time.Second, func() {
		exec.close()
	})
	defer timer.Stop()

	eventLen, err := exec.readEventLen()
	if err != nil {
		return
	}
	eventByte, err := exec.readEventBody(eventLen)
	if err != nil {
		return
	}
	event = &executor.Event{}
	proto.Unmarshal(eventByte, event)
	return event, nil
}

func (exec *ComputeExecutor) close() {
	exec.resp.Body.Close()
}

func (exec *ComputeExecutor) unackonwledgedUpdates() (result []executor.Call_Update) {
	if n := len(exec.unackedUpdates); n > 0 {
		result := make([]executor.Call_Update, 0, n)
		for k := range exec.unackedUpdates {
			result = append(result, exec.unackedUpdates[k])
		}
	}
	return
}

func (exec *ComputeExecutor) unacknowledgedTasks() (result []mesos.TaskInfo) {
	if n := len(exec.unackedTasks); n > 0 {
		result = make([]mesos.TaskInfo, 0, n)
		for k := range exec.unackedTasks {
			result = append(result, exec.unackedTasks[k])
		}
	}
	return
}
func (exec *ComputeExecutor) connect() (err error) {
	exec.httpClient = &http.Client{}
	subscribe := &executor.Call_Subscribe{
		UnacknowledgedTasks:   exec.unacknowledgedTasks(),
		UnacknowledgedUpdates: exec.unackonwledgedUpdates(),
	}
	call := &executor.Call{
		Type:        executor.Call_SUBSCRIBE,
		ExecutorID:  exec.ExecId,
		FrameworkID: exec.FWId,
		Subscribe:   subscribe,
	}
	data, err := proto.Marshal(call)
	if err != nil {
		return
	}
	req, err := http.NewRequest("POST", "http://"+exec.url+"/api/v1/executor", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Accept", "application/x-protobuf")
	req.Header.Set("User-Agent", "CKE-EXEC-"+exec.ExecId.Value)
	exec.resp, err = exec.httpClient.Do(req)
	if err != nil {
		return
	}
	if exec.resp.StatusCode != 200 {
		return errors.New("http response status code Error" + strconv.Itoa(exec.resp.StatusCode))
	}

	event, err := exec.readEvent()
	switch event.Type {
	case executor.Event_SUBSCRIBED:
		exec.ExecutorInfo = event.Subscribed.ExecutorInfo
		exec.heartbeatIntervalSeconds = 10.0
		log.Println("Registered executor : ", exec.ExecutorInfo.ExecutorID.Value)
		break
	default:
		err := errors.New("Error get exector error")
		break
	}
	return
}

func (exec *ComputeExecutor) loopEvents() (err error) {
	for true {
		event, err := exec.readEvent()
		if err != nil {
			return err
		}
		exec.recieved(event)
	}
	return
}

func (exec *ComputeExecutor) recieved(event *executor.Event) {
	switch event.Type {
	case executor.Event_LAUNCH:
		exec.onLaunch(event.Launch)
		break
	default:
		log.Println("unknown event type", event.Type)
		break
	}
}

func (exec *ComputeExecutor) onLaunch(launch *executor.Event_Launch) {
	exec.unackedTasks[launch.Task.TaskID] = launch.Task

	log.Println("Received Launch Task: " + launch.Task.TaskID.Value)

	update := &executor.Call_Update{
		Status: mesos.TaskStatus{
			TaskID:     launch.Task.TaskID,
			ExecutorID: &exec.ExecId,
			State:      mesos.TASK_RUNNING.Enum(),
			Source:     mesos.SOURCE_EXECUTOR.Enum(),
			UUID:       []byte(uuid.NewRandom()),
		},
	}

	err := exec.CallAgent(update)
	if err != nil {
		log.Println("Call UPDATE error: ", err)
	} else {
		exec.unackedUpdates[string(update.Status.UUID)] = *update
	}

	update = &executor.Call_Update{
		Status: mesos.TaskStatus{
			TaskID: launch.Task.TaskID,
			State:  mesos.TASK_FINISHED.Enum(),
			Source: mesos.SOURCE_EXECUTOR.Enum(),
			UUID:   []byte(uuid.NewRandom()),
		},
	}
	err = exec.CallAgent(update)
	if err != nil {
		log.Println("Call UPDATE error: ", err)
	} else {
		exec.unackedUpdates[string(update.Status.UUID)] = *update
	}
}

func (exec *ComputeExecutor) CallAgent(call interface{}) (err error) {
	calReq := &executor.Call{
		ExecutorID:  exec.ExecId,
		FrameworkID: exec.FWId,
	}

	switch call.(type) {
	case *executor.Call_Update:
		calReq.Type = executor.Call_UPDATE
		calReq.Update = call.(*executor.Call_Update)
		break
	default:
		return errors.New("unknown call type" + reflect.TypeOf(call).String() + "in call agent ")
		break
	}
	data, err := proto.Marshal(calReq)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", "http://"+exec.url+"/api/v1/executor", bytes.NewReader(data))
	if err != nil {
		return
	}

	req.Header.Set("Host", exec.url)
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Accept", "application/x-protobuf")
	req.Header.Set("User-Agent", exec.ExecId.Value)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		buf, err := ioutil.ReadAll(io.LimitReader(resp.Body, 4*1024))
		if err != nil {
			return errors.New("Error read response")
		}
		return errors.New("Http response status code Error: " + strconv.Itoa(resp.StatusCode) + " body:" + string(buf))
	}
	return
}
