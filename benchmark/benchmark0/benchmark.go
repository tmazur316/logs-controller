package main

import (
	"context"
	"fmt"
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
	sizes := []int{5_000, 500_000, 1_000_000, 5_000_000, 15_000_000}

	if err := os.MkdirAll(copyPath, os.ModePerm); err != nil {
		log.Println(err)
	}

	for _, size := range sizes {
		testPod, err := createPod(client, size)
		if err != nil {
			panic(err)
		}

		time.Sleep(30 * time.Second)

		score := benchmark{
			successfulTries: 0,
			totalTime:       0,
		}

		testPodName := testPod.GetName()

		for i := 1; i <= 300; i++ {
			t, err := measuredFunc(client, newPod, testPodName)
			if err != nil {
				log.Println(err)
				continue
			}

			score.successfulTries++
			score.totalTime += t

			log.Printf("current benchmark state: size: %d, successful tries: %d, last time: %f, total time: %f, mean time: %f\n",
				size, score.successfulTries, t, score.totalTime, score.totalTime/float64(score.successfulTries))

			if i%5 == 0 {
				if err := os.RemoveAll(copyPath + "/" + testPodName); err != nil {
					log.Println(err)
				}
			}
		}

		if err := deletePod(client, testPodName); err != nil {
			log.Println(err)

			os.Exit(1)
		}

		time.Sleep(60 * time.Second)
	}
}

func createPod(client *kubernetes.Clientset, size int) (*v1.Pod, error) {
	sizeStr := fmt.Sprintf("%d", size)
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "core/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name + sizeStr,
			Labels: map[string]string{"app": "busy-logger"},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "busy-logger",
					Image:           "tmazur316/busy-logger:1.0",
					ImagePullPolicy: v1.PullAlways,
					Command:         []string{"go", "run", "./main.go"},
					Args:            []string{"-period", "24h", "-size", sizeStr, "-pods", "1"},
				},
			},
		},
	}

	return client.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
}

func deletePod(client *kubernetes.Clientset, podName string) error {
	return client.CoreV1().Pods(namespace).Delete(context.Background(), podName, metav1.DeleteOptions{})
}

func measuredFunc(client *kubernetes.Clientset, newPod func(object interface{}) (pods.Pod, error), podName string) (float64, error) {
	start := time.Now()

	object, err := client.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}

	pod, err := newPod(object)
	if err != nil {
		return 0, err
	}

	if err := pod.CopyLogs(); err != nil {
		return 0, err
	}

	duration := time.Since(start).Seconds()

	return duration, nil
}

type benchmark struct {
	successfulTries int
	totalTime       float64
}
