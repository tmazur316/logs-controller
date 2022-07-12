package main

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"log"
	"logs-controller/informer"
	"logs-controller/pod"
	ctrlRuntime "sigs.k8s.io/controller-runtime"
)

func main() {
	var client = kubernetes.NewForConfigOrDie(ctrlRuntime.GetConfigOrDie())
	namespace := "default"
	label := "app"
	selectors := map[string]string{"app": "manager"}

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	defer queue.ShutDown()

	newPod := pod.NewPodFunc(client, namespace, selectors)

	inf := informer.PodsInformerFactory{
		Queue: queue,
	}.New(client, namespace, label)

	stopChan := make(chan struct{})
	go inf.Run(stopChan)

	if !cache.WaitForCacheSync(stopChan) {
		close(stopChan)
		log.Println("cache was not synced")
		return
	}

	for {
		key, shutdown := queue.Get()
		if shutdown {
			continue
		}

		err := process(inf, key.(string), newPod)

		if err == nil {
			queue.Forget(key)
		} else if queue.NumRequeues(key) < 5 {
			queue.AddRateLimited(key)
		} else {
			queue.Forget(key)
			log.Println(err)
		}

		queue.Done(key)
	}
}

func process(inf cache.SharedIndexInformer, key string, newP func(object interface{}) (pod.Pod, error)) error {
	obj, exists, err := inf.GetIndexer().GetByKey(key)
	if err != nil {
		return err
	}

	if !exists {
		log.Println("deleted event")
		return nil
	}

	p, err := newP(obj)
	if p.IsNotBeingDeleted() {
		if p.FinalizerIsNotPresent() {
			err := p.SetFinalizer()
			if err != nil {
				return err
			}
		}

		return nil
	}

	if err := p.RemoveFinalizer(); err != nil {
		return err
	}

	return nil
}
