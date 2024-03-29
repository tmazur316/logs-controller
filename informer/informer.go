package informer

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type PodsInformerFactory struct {
	Queue workqueue.RateLimitingInterface
}

func (i PodsInformerFactory) New(cli *kubernetes.Clientset, namespace string, labelSelectors map[string]string) cache.SharedIndexInformer {
	watcher := PodsWatcherFactory{
		Client:    cli,
		Namespace: namespace,
		Selectors: labelSelectors,
	}.New()

	informer := cache.NewSharedIndexInformer(&watcher, &v1.Pod{}, 0, nil)
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    i.addFunc,
		UpdateFunc: i.updateFunc,
		DeleteFunc: i.deleteFunc,
	})

	return informer
}

func (i PodsInformerFactory) addFunc(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}

	i.Queue.Add(key)
}

func (i PodsInformerFactory) updateFunc(oldObj, newObj interface{}) {
	oldPod := oldObj.(*v1.Pod)
	newPod := newObj.(*v1.Pod)
	if oldPod.ResourceVersion == newPod.ResourceVersion {
		return
	}

	newKey, err := cache.MetaNamespaceKeyFunc(newObj)
	if err != nil {
		return
	}

	i.Queue.Add(newKey)
}

func (i PodsInformerFactory) deleteFunc(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}

	i.Queue.Add(key)
}
