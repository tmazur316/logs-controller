package pods

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"logs-controller/collection"
)

const finalizer = "operator.logs/finalizer"

var (
	log = &logrus.Logger{
		Out:          os.Stderr,
		Formatter:    new(logrus.TextFormatter),
		Level:        logrus.InfoLevel,
		ReportCaller: false,
	}

	source = rand.NewSource(time.Now().UnixNano())
)

func NewPodFunc(cli *kubernetes.Clientset, namespace string, selectors map[string]string) func(object interface{}) (Pod, error) {
	return func(object interface{}) (Pod, error) {
		pod, isPod := object.(*v1.Pod)
		if !isPod {
			return Pod{}, fmt.Errorf("object is not a kubernetes pod")
		}

		return Pod{
			pod:       pod,
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

func (p *Pod) Name() string {
	return p.pod.Name
}

func (p *Pod) IsNotBeingDeleted() bool {
	return p.pod.DeletionTimestamp.IsZero()
}

func (p *Pod) FinalizerIsSet() bool {
	return collection.ContainsAllValues(p.pod.Labels, p.selectors) && lo.Contains(p.pod.Finalizers, finalizer)
}

func (p *Pod) Finalizers() []string {
	return p.pod.Finalizers
}

func (p *Pod) SetFinalizer() error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pod, err := p.client.CoreV1().Pods(p.namespace).Get(context.Background(), p.Name(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		newPod := pod.DeepCopy()
		newPod.Finalizers = append(newPod.Finalizers, finalizer)

		_, err = p.client.CoreV1().Pods(p.namespace).Update(context.Background(), newPod, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.WithField("pod", p.pod.Name).Info("finalizer added")

	return nil
}

func (p *Pod) RemoveFinalizer() error {
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		pod, err := p.client.CoreV1().Pods(p.namespace).Get(context.Background(), p.Name(), metav1.GetOptions{})
		if err != nil {
			return err
		}

		newPod := pod.DeepCopy()
		newPod.SetFinalizers(collection.Remove(newPod.GetFinalizers(), finalizer))

		_, err = p.client.CoreV1().Pods(p.namespace).Update(context.Background(), newPod, metav1.UpdateOptions{})
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	log.WithField("pod", p.pod.Name).Info("finalizer removed")

	return nil
}

func (p *Pod) CopyLogs() error {
	logsRequest := p.client.CoreV1().Pods(p.namespace).GetLogs(p.pod.GetName(), &v1.PodLogOptions{})
	response := logsRequest.Do(context.Background())

	logs, err := response.Raw()
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}

		log.WithError(err).Error()
		return err
	}

	log.WithField("pod", p.Name()).WithField("logs", string(logs)).Debug("logs read")

	now := time.Now().Format("2006_01_02_15_04_05")
	destPath := fmt.Sprintf("/var/log/copy/%s/copy_%s_%d", p.pod.Name, now, source.Int63())
	if err := saveLogs(destPath, logs); err != nil {
		log.WithError(err).Error("failed to save logs")
		return err
	}

	return nil
}

func saveLogs(destination string, logs []byte) error {
	err := os.MkdirAll(filepath.Dir(destination), os.ModePerm)
	if err != nil && !os.IsExist(err) {
		log.WithField("destination", destination).WithError(err).Error("failed to create directory for logs copy")
		return err
	}

	file, err := os.OpenFile(destination, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		log.WithField("destination", destination).WithError(err).Error("failed to open file for logs copy")
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			log.WithError(err).Error("failed to close log file")
		}
	}()

	if _, err = file.Write(logs); err != nil {
		log.WithField("destination", destination).WithError(err).Error("failed to copy logs to file")
		return err
	}

	log.WithField("destination", destination).Info("logs copied to destination")

	return nil
}
