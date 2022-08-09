package main

import (
	"flag"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	runtime "sigs.k8s.io/controller-runtime"

	"logs-controller/controller"
)

var log = &logrus.Logger{
	Out:          os.Stderr,
	Formatter:    new(logrus.TextFormatter),
	Level:        logrus.InfoLevel,
	ReportCaller: true,
}

func main() {
	namespace := flag.String("namespace", "default", "namespace with pods to watch")
	selectors := flag.String("selectors", "app=manager,app2=manager2", "pod selectors with comma separators, example: key1=value1,key2=value2")

	split := strings.Split(*selectors, ",")

	selectorsMap := map[string]string{}
	for _, selector := range split {
		selectorSplit := strings.Split(selector, "=")
		if len(selectorSplit) != 2 {
			log.WithField("selector", selector).Error("Invalid selector argument, stopping program execution")

			os.Exit(1)
		}

		key := selectorSplit[0]
		value := selectorSplit[1]

		selectorsMap[key] = value
	}

	client := kubernetes.NewForConfigOrDie(runtime.GetConfigOrDie())

	ctrl := controller.NewController(client, *namespace, selectorsMap)
	ctrl.Run()
}
