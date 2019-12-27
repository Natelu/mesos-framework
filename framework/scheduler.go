package framework

import (
	"log"
	"math/rand"
	"strconv"
	"strings"

	"github.com/mesos/mesos-go/api/v1/lib/scheduler"

	mesos "github.com/mesos/mesos-go/api/v1/lib"
)

type Scheduler struct {
	Framework *Framework
	isRunning bool
	Count     int
	Cluster   *Cluster
}

func (sch *Scheduler) createContainer(image string) (containerInfo *mesos.ContainerInfo) {
	containerType := mesos.ContainerInfo_DOCKER
	network := mesos.ContainerInfo_DockerInfo_HOST
	return &mesos.ContainerInfo{
		Type: &containerType,
		Docker: &mesos.ContainerInfo_DockerInfo{
			Image:   image,
			Network: &network,
		},
	}
}

func (sch *Scheduler) createExecutor() (executor *mesos.ExecutorInfo) {
	hasShell := false
	cmd := "/opt/cke/bin/luzx-executor:0.0.1"
	image := "reg.mg.hcbss/open/luzx-executor:0.0.1"
	execId := "Luzx-Exec" + strconv.Itoa(rand.Intn(1000))

	return &mesos.ExecutorInfo{
		Type: mesos.ExecutorInfo_CUSTOM,
		ExecutorID: mesos.ExecutorID{
			Value: execId,
		},
		FrameworkID: sch.Framework.FrameworkID,
		Command: &mesos.CommandInfo{
			Shell:     &hasShell,
			Value:     &cmd,
			Arguments: []string{},
		},
		Container: sch.createContainer(image),
	}
}

func PrintResource(resource *mesos.Resource) {
	resourceStr := resource.Name + ""
	switch *resource.Type {
	case mesos.RANGES:
		resourceStr = resourceStr + (*resource.Ranges).String()
		break
	case mesos.SET:
		resourceStr = resourceStr + (*resource.Set).String()
		break
	case mesos.SCALAR:
		resourceStr = resourceStr + (*resource.Scalar).String()
		break
	}
	log.Println("Mesos Resource : " + resourceStr)
}

func PrintOffer(offer *mesos.Offer) {
	log.Println("Recieved Offer: ", offer.ID, ", HostName: ", offer.Hostname,
		", agent : ", offer.AgentID)
	resources := offer.Resources
	for i := 0; i < len(resources); i++ {
		PrintResource(&resources[i])
	}
}

// func (sch *KubernetsScheduler) processOffers(offers []mesos.Offer) (
// 	acceptIds []mesos.OfferID, opers []mesos.Offer_Operation, declineIds []mesos.OfferID) {

// 	acceptIds = make([]mesos.OfferID, 0, len(offers))
// 	declineIds = make([]mesos.OfferID, 0, len(offers))
// 	for _, offer := range offers {
// 		oper := sch.processOffer(offer)
// 		if oper != nil {
// 			acceptIds = append(acceptIds, offer.ID)
// 			opers = append(opers, *oper)
// 		} else {
// 			declineIds = append(declineIds, offer.ID)
// 		}
// 	}
// 	return
// }

func (sch *Scheduler) processOffer(offer mesos.Offer) (oper *mesos.Offer_Operation) {

	cpuNum := 1.0
	memNum := 256.0
	var task *TaskInfo
	tasks := sch.Cluster.GetInconsistentTasks()
	for _, t := range tasks {
		offerCpu := 0.0
		offerMem := 0.0
		for _, resource := range offer.Resources {
			if strings.Compare(resource.Name, "mem") == 0 {
				offerMem += resource.Scalar.Value
			}
			if strings.Compare(resource.Name, "cpus") == 0 {
				offerCpu += resource.Scalar.Value
			}
		}
		if offerCpu < cpuNum || offerMem < memNum {
			return
		}
		task = t
		break
	}
	if task == nil {
		return
	}

	cpuRes := mesos.Resource{
		Name: "cpus",
		Type: mesos.SCALAR.Enum(),
		Scalar: &mesos.Value_Scalar{
			Value: cpuNum,
		},
	}
	memRes := mesos.Resource{
		Name: "mem",
		Type: mesos.SCALAR.Enum(),
		Scalar: &mesos.Value_Scalar{
			Value: memNum,
		},
	}
	resource := []mesos.Resource{cpuRes, memRes}
	mesosTasks := make([]mesos.TaskInfo, 0, 10)
	// 创建 executor & task
	taskID := task.Name + "." + strconv.Itoa(sch.Count)
	sch.Count++
	mesosTask := mesos.TaskInfo{
		Name: task.Name,
		TaskID: mesos.TaskID{
			Value: taskID,
		},
		AgentID:   offer.AgentID,
		Resources: resource,
		Executor:  sch.createExecutor(),
	}
	mesosTasks = append(mesosTasks, mesosTask)

	oper = &mesos.Offer_Operation{
		Type: mesos.Offer_Operation_LAUNCH,
		Launch: &mesos.Offer_Operation_Launch{
			TaskInfos: mesosTasks,
		},
	}
	log.Println("start task (" + taskID + ") at " + offer.Hostname + ", executor: + " + mesosTask.Executor.ExecutorID.Value)
	return
}

func (sch *Scheduler) processOffers(offers []mesos.Offer) (
	acceptIds []mesos.OfferID, opers []mesos.Offer_Operation, declineIds []mesos.OfferID) {
	acceptIds = make([]mesos.OfferID, 0, len(offers))
	declineIds = make([]mesos.OfferID, 0, len(offers))
	for _, offer := range offers {
		oper := sch.processOffer(offer)
		if oper != nil {
			acceptIds = append(acceptIds, offer.ID)
			opers = append(opers, *oper)
		} else {
			declineIds = append(declineIds, offer.ID)
		}
	}
	return
}

func (sch *Scheduler) onDecline(offers []mesos.Offer) (err error) {
	refuseGap := 5.0
	offerIds := make([]mesos.OfferID, 0, len(offers))
	for _, offer := range offers {
		offerIds = append(offerIds, offer.ID)
	}
	callDecline := &scheduler.Call_Decline{
		OfferIDs: offerIds,
		Filters: &mesos.Filters{
			RefuseSeconds: &refuseGap,
		},
	}
	err = sch.Framework.CallMaster(callDecline)
	return
}

func (sch *Scheduler) onOffers(eventOffers *scheduler.Event_Offers) {

	acceptIds, opers, declineIds := sch.processOffers(eventOffers.Offers)

	if len(acceptIds) > 0 && len(opers) > 0 {
		refuseSeconds := 5.0
		accept := &scheduler.Call_Accept{
			OfferIDs:   acceptIds,
			Operations: opers,
			Filters: &mesos.Filters{
				RefuseSeconds: &refuseSeconds,
			},
		}
		err := sch.Framework.CallMaster(accept)
		if err != nil {
			log.Println("Call accept :", err)
		}
	}

	declineSeconds := 5.0
	decline := &scheduler.Call_Decline{
		OfferIDs: declineIds,
		Filters: &mesos.Filters{
			RefuseSeconds: &declineSeconds,
		},
	}
	err := sch.Framework.CallMaster(decline)
	if err != nil {
		log.Println("Call Decline err:", err)
	}
}

func (sch *Scheduler) onHeartBeat() {
	log.Println("recieved heartbeat event")
}

// Received process event
func (sch *Scheduler) Received(event *scheduler.Event) {
	switch event.Type {
	case scheduler.Event_OFFERS:
		sch.onOffers(event.Offers)
	case scheduler.Event_HEARTBEAT:
		sch.onHeartBeat()
	default:
		log.Println("Received unknown event:", event.Type)
		break
	}
}
