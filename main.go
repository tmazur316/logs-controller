package main

import (
	"k8s.io/client-go/kubernetes"
	"logs-controller/controller"
	runtime "sigs.k8s.io/controller-runtime"
)

func main() {
	client := kubernetes.NewForConfigOrDie(runtime.GetConfigOrDie())
	namespace := "default"
	selectors := map[string]string{"app": "manager"}

	ctrl := controller.NewController(client, namespace, selectors)
	ctrl.Run()
}
