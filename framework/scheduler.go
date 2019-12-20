package framework

import (
	"log"
	"strings"

	"github.com/mesos/mesos-go/api/v1/lib/scheduler"

	mesos "github.com/mesos/mesos-go/api/v1/lib"
)

type Scheduler struct {
	Framework *Framework
	isRunning bool
	Count     int
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

func (sch *Scheduler) processOffer(offer mesos.Offer) (oper mesos.Offer_Operation, err error) {

	cpuNum := 1.0
	memNum := 256.0
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
		return nil, nil
	}
	cpuRes := &mesos.Resource{
		Name: "cpus",
		Type: mesos.SCALAR.Enum(),
		Scalar: &mesos.Value_Scalar{
			Value: cpuNum,
		},
	}
	memRes := &mesos.Resource{
		Name: "mem",
		Type: mesos.SCALAR.Enum(),
		Scalar: &mesos.Value_Scalar{
			Value: memNum,
		},
	}
	resource := []mesos.Resource{cpuNum, memRes}
	// 创建 executor & task
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

// func (sch *Scheduler) processOffers(offers []mesos.Offer) (acceptIds []mesos.OfferID,
// 	opers []mesos.Offer_Operation, declineIds []mesos.OfferID) {
// 	acceptIds = make([]mesos.OfferID, 0, len(offers))
// 	opers = make([]mesos.Offer_Operation, 0, len(offers))
// 	declineIds = make([]mesos.OfferID, 0, len(offers))
// 	for _, offer := range offers {
// 		// oper := sch.p/
// 	}
// }

func (sch *Scheduler) onOffers(offers *scheduler.Event_Offers) {
	for _, offer := range offers.Offers {
		PrintOffer(&offer)
	}
	sch.onDecline(offers.Offers)
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
