package main

import (
	"log"
	"mine-mesos-framework/framework"
	"mine-mesos-framework/http"
)

var (
	name        = "root_2037"
	httpAddr    = "0.0.0.0:8512"
	mesosMaster = "10.124.142.191:5050"
	role        = "*"
)

func main() {
	fw := &framework.Framework{
		Name: name,
		Url:  mesosMaster,
	}
	err := fw.Start(name, role, mesosMaster)
	if err != nil {
		log.Println("Register mesos framework err: ", err)
	}

	server := http.NewHttpServer(framework.NewCluster())
	server.Run(&httpAddr)
}
