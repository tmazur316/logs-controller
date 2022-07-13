package controller

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"log"
	"logs-controller/informer"
	"logs-controller/pods"
)

type Controller struct {
	queue    workqueue.RateLimitingInterface
	informer cache.SharedIndexInformer
	newPod   func(object interface{}) (pods.Pod, error)
}

func NewController(client *kubernetes.Clientset, PodNamespace string, LabelSelectors map[string]string) Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	newPod := pods.NewPodFunc(client, PodNamespace, LabelSelectors)

	inf := informer.PodsInformerFactory{
		Queue: queue,
	}.New(client, PodNamespace, LabelSelectors)

	return Controller{
		queue:    queue,
		informer: inf,
		newPod:   newPod,
	}
}

func (c *Controller) Run() {
	stopChan := make(chan struct{})
	go c.informer.Run(stopChan)

	cacheSynced := cache.WaitForCacheSync(stopChan)
	if !cacheSynced {
		close(stopChan)
		log.Println("cache was not synced")
		return
	}

	defer c.queue.ShutDown()

	c.reconcile()
}

func (c *Controller) reconcile() {
	for {
		key, shutdown := c.queue.Get()
		if shutdown {
			continue
		}

		err := c.handleKey(key)

		if err == nil {
			c.queue.Forget(key)
		} else if c.queue.NumRequeues(key) < 5 {
			c.queue.AddRateLimited(key)
		} else {
			c.queue.Forget(key)
		}

		c.queue.Done(key)
	}
}

func (c *Controller) handleKey(key interface{}) error {
	object, exists, err := c.informer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		return err
	}

	if !exists {
		log.Println("deleted event")
		return nil
	}

	pod, err := c.newPod(object)
	if pod.IsNotBeingDeleted() {
		if pod.FinalizerIsNotPresent() {
			err := pod.SetFinalizer()
			if err != nil {
				return err
			}
		}

		return nil
	}

	if err := pod.RemoveFinalizer(); err != nil {
		return err
	}

	return nil
}
