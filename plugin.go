package main

import (
	"github.com/virtmonitor/driver"

	"github.com/virtmonitor/xen/driver"
)

//Driver driver backend
var Driver driver.Driver = &xen.Xen{}

func main() {}
