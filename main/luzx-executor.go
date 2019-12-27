package main

import (
	"log"
	"mine-mesos-framework/framework"
	"os"
)

func main() {
	url := os.Getenv("MESOS_AGENT_ENDPOINT")
	fwId := os.Getenv("MESOS_FRAMEWORK_ID")
	execId := os.Getenv("MESOS_EXECUTOR_ID")

	// exec := executor.NewKubernetsExecutor()
	exec := framework.NewComputeExecutor()
	err := exec.Start(url, fwId, execId)
	if err != nil {
		log.Println("Register mesos framework error:", err)
		return
	}

	log.Println("OK.")
}
