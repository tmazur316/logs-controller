package main

import (
	"context"
	"log"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"logs-controller/pods"
	runtime "sigs.k8s.io/controller-runtime"
)

const (
	namespace = v1.NamespaceDefault
	name      = "busy-logger"
	copyPath  = "/var/log/copy/"
)

func main() {
	client := kubernetes.NewForConfigOrDie(runtime.GetConfigOrDie())
	selectors := map[string]string{"app": "busy-logger"}
	newPod := pods.NewPodFunc(client, namespace, selectors)

	if err := os.MkdirAll(copyPath, os.ModePerm); err != nil {
		log.Println(err)
	}

	testPod, err := createPod(client)
	if err != nil {
		panic(err)
	}

	time.Sleep(30 * time.Second)

	score := benchmark{
		successfulTriesAdd:    0,
		successfulTriesRemove: 0,
		totalAddTime:          0,
		totalRemoveTime:       0,
		errorsAdd:             0,
		errorsRemove:          0,
	}

	testPodName := testPod.GetName()
	skipAdd := false

	for i := 1; i <= 300; i++ {
		if !skipAdd {
			t, err := measureAdd(client, newPod, testPodName)
			if err != nil {
				score.errorsAdd++
				log.Printf("current benchmark add state: successful tries: %d, last time: %f, total time: %f, mean time: %f, errors: %d\n",
					score.successfulTriesAdd, t, score.totalAddTime, score.totalAddTime/float64(score.successfulTriesAdd), score.errorsAdd)
				continue
			}

			score.successfulTriesAdd++
			score.totalAddTime += t

			log.Printf("current benchmark add state: successful tries: %d, last time: %f, total time: %f, mean time: %f, errors: %d\n",
				score.successfulTriesAdd, t, score.totalAddTime, score.totalAddTime/float64(score.successfulTriesAdd), score.errorsAdd)
		}

		t2, err := measureDelete(client, newPod, testPodName)
		if err != nil {
			score.errorsRemove++
			skipAdd = true
			log.Printf("current benchmark remove state: successful tries: %d, last time: %f, total time: %f, mean time: %f, errors: %d\n",
				score.successfulTriesRemove, t2, score.totalRemoveTime, score.totalRemoveTime/float64(score.successfulTriesRemove),
				score.errorsRemove)
			continue
		}

		score.successfulTriesRemove++
		score.totalRemoveTime += t2
		skipAdd = false

		log.Printf("current benchmark remove state: successful tries: %d, last time: %f, total time: %f, mean time: %f, errors: %d\n",
			score.successfulTriesRemove, t2, score.totalRemoveTime, score.totalRemoveTime/float64(score.successfulTriesRemove),
			score.errorsRemove)

	}

	if err := deletePod(client, testPodName); err != nil {
		log.Println(err)

		os.Exit(1)
	}
}

func createPod(client *kubernetes.Clientset) (*v1.Pod, error) {
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "core/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"app": "busy-logger"},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "busy-logger",
					Image:           "tmazur316/busy-logger:1.0",
					ImagePullPolicy: v1.PullAlways,
					Command:         []string{"go", "run", "./main.go"},
					Args:            []string{"-period", "24h", "-size", "1000000", "-pods", "1"},
				},
			},
		},
	}

	return client.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
}

func deletePod(client *kubernetes.Clientset, podName string) error {
	return client.CoreV1().Pods(namespace).Delete(context.Background(), podName, metav1.DeleteOptions{})
}

func measureAdd(client *kubernetes.Clientset, newPod func(object interface{}) (pods.Pod, error), podName string) (float64, error) {
	start := time.Now()

	object, err := client.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}

	pod, err := newPod(object)
	if err != nil {
		return 0, err
	}

	if err := pod.SetFinalizer(); err != nil {
		return 0, err
	}

	duration := time.Since(start).Seconds()

	return duration, nil
}

func measureDelete(client *kubernetes.Clientset, newPod func(object interface{}) (pods.Pod, error), podName string) (float64, error) {
	start := time.Now()

	object, err := client.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}

	pod, err := newPod(object)
	if err != nil {
		return 0, err
	}

	if err := pod.RemoveFinalizer(); err != nil {
		return 0, err
	}

	duration := time.Since(start).Seconds()

	return duration, nil
}

type benchmark struct {
	successfulTriesAdd    int
	successfulTriesRemove int
	errorsAdd             int
	totalAddTime          float64
	totalRemoveTime       float64
	errorsRemove          int
}
