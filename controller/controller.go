package controller

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"logs-controller/informer"
	"logs-controller/pods"
)

var log = &logrus.Logger{
	Out:          os.Stderr,
	Formatter:    new(logrus.TextFormatter),
	Level:        logrus.InfoLevel,
	ReportCaller: true,
}

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
		log.Error("cache was not synced")
		return
	}

	defer c.queue.ShutDown()

	wait.Until(c.reconcile, time.Second, stopChan)
}

func (c *Controller) reconcile() {
	for c.processKey() {
	}
}

func (c *Controller) processKey() bool {
	key, shutdown := c.queue.Get()
	if shutdown {
		return false
	}

	err := c.handleKey(key)

	if err == nil {
		c.queue.Forget(key)
	} else if c.queue.NumRequeues(key) < 25 {
		c.queue.AddRateLimited(key)
		return true
	} else {
		c.queue.Forget(key)
	}

	c.queue.Done(key)

	return true
}

func (c *Controller) handleKey(key interface{}) error {
	object, exists, err := c.informer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		return err
	}

	if !exists {
		log.Debug("deleted event")
		return nil
	}

	pod, err := c.newPod(object)
	if err != nil {
		log.WithField("pod", pod.Name()).WithError(err).Error("failed to retrieve pod")
		return err
	}

	if pod.IsNotBeingDeleted() {
		if !pod.FinalizerIsSet() {
			if err := pod.SetFinalizer(); err != nil {
				log.WithField("pod", pod.Name()).WithError(err).Error("failed to set finalizer")
				return err
			}
		}

		return nil
	}

	if pod.FinalizerIsSet() {
		if err := pod.CopyLogs(); err != nil {
			log.WithField("pod", pod.Name()).WithError(err).Errorf("failed to copy logs")
			return err
		}

		if err := pod.RemoveFinalizer(); err != nil {
			log.WithField("pod", pod.Name()).WithError(err).Errorf("failed to remove finalizer")
			return err
		}
	}

	return nil
}
