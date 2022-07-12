package informer

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type PodsWatcherFactory struct {
	Client    *kubernetes.Clientset
	Namespace string
	Label     string
}

// New creates ListWatch for pods with given namespace and label selector
func (p PodsWatcherFactory) New() cache.ListWatch {
	return cache.ListWatch{
		ListFunc:  p.listPods,
		WatchFunc: p.watchPods,
	}
}

func (p PodsWatcherFactory) listPods(options metav1.ListOptions) (runtime.Object, error) {
	ctx := context.Background()
	options.LabelSelector = p.Label

	return p.Client.CoreV1().Pods(p.Namespace).List(ctx, options)
}

func (p PodsWatcherFactory) watchPods(options metav1.ListOptions) (watch.Interface, error) {
	ctx := context.Background()
	options.LabelSelector = p.Label

	return p.Client.CoreV1().Pods(p.Namespace).Watch(ctx, options)
}
