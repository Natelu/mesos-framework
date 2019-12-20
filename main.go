package main

import (
	"mine-mesos-framework/framework"
	// framework "framework"
)

var (
	name        = "root_2037"
	mesosMaster = "10.124.142.191:5050"
	role        = "*"
)

func main() {
	fw := &framework.Framework{
		Name: name,
		Url:  mesosMaster,
	}
	fw.Start(name, role, mesosMaster)
	// fmt.Println("frameworkID is s%", *fw.FrameworkID)
}
