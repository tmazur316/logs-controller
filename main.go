package main

import (
	"k8s.io/client-go/kubernetes"
	controller2 "logs-controller/controller"
	ctrlRuntime "sigs.k8s.io/controller-runtime"
)

func main() {
	client := kubernetes.NewForConfigOrDie(ctrlRuntime.GetConfigOrDie())
	namespace := "default"
	selectors := map[string]string{"app": "manager"}

	controller := controller2.NewController(client, namespace, selectors)
	controller.Run()
}
