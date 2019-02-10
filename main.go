package main

import (
	"github.com/kubeless/kubeless/pkg/functions"
	"github.com/nadilas/godaddy-oc-updater/kubeless"
)

func main() {
	kubeless.Handler(functions.Event{}, functions.Context{})
}
