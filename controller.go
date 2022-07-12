package main

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"logs-controller/pod"
)

type Controller struct {
	Queue    workqueue.RateLimitingInterface
	Informer cache.SharedIndexInformer
	NewPod   func(object runtime.Object) (pod.Pod, error)
}
