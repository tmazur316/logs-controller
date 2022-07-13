package pods

import (
	"context"
	"fmt"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"logs-controller/collection"
)

const finalizer = "operator.logs/finalizer"

func NewPodFunc(cli *kubernetes.Clientset, namespace string, selectors map[string]string) func(object interface{}) (Pod, error) {
	return func(object interface{}) (Pod, error) {
		podObject, isPod := object.(*v1.Pod)
		if !isPod {
			return Pod{}, fmt.Errorf("object is not a kubernetes pods")
		}

		return Pod{
			pod:       podObject,
			namespace: namespace,
			selectors: selectors,
			client:    cli,
		}, nil
	}
}

type Pod struct {
	pod       *v1.Pod
	namespace string
	selectors map[string]string
	client    *kubernetes.Clientset
}

func (p *Pod) IsNotBeingDeleted() bool {
	return p.pod.DeletionTimestamp.IsZero()
}

func (p *Pod) FinalizerIsNotPresent() bool {
	return collection.ContainsAllValues(p.pod.Labels, p.selectors) && !lo.Contains(p.pod.Finalizers, finalizer)
}

func (p *Pod) SetFinalizer() error {
	newPod := p.pod.DeepCopy()
	newPod.Finalizers = append(newPod.Finalizers, finalizer)

	log.Println("finalizer added")

	_, err := p.client.CoreV1().Pods(p.namespace).Update(context.Background(), newPod, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (p *Pod) RemoveFinalizer() error {
	newPod := p.pod.DeepCopy()
	newPod.SetFinalizers(collection.Remove(newPod.GetFinalizers(), finalizer))

	_, err := p.client.CoreV1().Pods(p.namespace).Update(context.Background(), newPod, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	log.Println("finalizer removed")

	return nil
}
