package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/spotahome/kooper/v2/controller"
	kooperlog "github.com/spotahome/kooper/v2/log"
	kooperlogrus "github.com/spotahome/kooper/v2/log/logrus"
)

var (
	logger kooperlog.Logger
)

type PodEventsHandler struct {
	pods map[string]*corev1.Pod
}

func (h *PodEventsHandler) fullName(pod *corev1.Pod) string {
	podName, podNamespace := pod.Name, pod.Namespace
	return fmt.Sprintf("%s/%s", podNamespace, podName)
}

func (h *PodEventsHandler) addPod(pod *corev1.Pod) {
	fullName := h.fullName(pod)
	if _, ok := h.pods[fullName];ok {
		return 
	}
	podStatus := pod.Status
	logger.Infof("Adding pod %s phase %s", fullName, podStatus.Phase)
	h.pods[fullName] = pod
	podSpec := pod.Spec
	podContainers := podSpec.Containers
	for _, container := range(podContainers) {
		containerName, containrPorts := container.Name, container.Ports
		if len(containrPorts) == 0 {
			continue
		}
		logger.Infof("\tContainer %s ports %v", containerName, containrPorts)
	}
}

func (h *PodEventsHandler) removePod(pod *corev1.Pod) {
	fullName := h.fullName(pod)
	if _, ok := h.pods[fullName];!ok {
		return 
	}
	podStatus := pod.Status
	logger.Infof("Remove pod %s phase %s", fullName, podStatus.Phase)
	delete(h.pods, fullName)
}

// Track the pods life cycle
func (h *PodEventsHandler) handler(_ context.Context, obj runtime.Object) error {
	pod := obj.(*corev1.Pod)
	podStatus := pod.Status
	switch podStatus.Phase {
	case corev1.PodPending, corev1.PodRunning, corev1.PodSucceeded:
		h.addPod(pod)
	default:
		h.removePod(pod)
	}
	return nil
}

func run() error {
	ctx := context.Background()
	// Initialize logger.
	logger = kooperlogrus.New(logrus.NewEntry(logrus.New())).
		WithKV(kooperlog.KV{"example": "ingress-controller"})

	// Get k8s client.
	k8scfg, err := rest.InClusterConfig()
	if err != nil {
		// No in cluster? letr's try locally
		kubehome := filepath.Join(homedir.HomeDir(), ".kube", "config")
		k8scfg, err = clientcmd.BuildConfigFromFlags("", kubehome)
		if err != nil {
			return fmt.Errorf("error loading kubernetes configuration: %w", err)
		}
	}
	k8scli, err := kubernetes.NewForConfig(k8scfg)
	if err != nil {
		return fmt.Errorf("error creating kubernetes client: %w", err)
	}

	// Create our retriever so the controller knows how to get/listen for pod events.
	retr := controller.MustRetrieverFromListerWatcher(&cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return k8scli.CoreV1().Pods("").List(ctx, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return k8scli.CoreV1().Pods("").Watch(ctx, options)
		},
	})

	podEventsHandler := &PodEventsHandler {
		pods: map[string]*corev1.Pod{},
	}
	hand := controller.HandlerFunc(podEventsHandler.handler)

	// Create the controller with custom configuration.
	cfg := &controller.Config{
		Name:      "ingress-controller",
		Handler:   hand,
		Retriever: retr,
		Logger:    logger,

		ProcessingJobRetries: 5,
		ResyncInterval:       5 * time.Second,
		ConcurrentWorkers:    1,
	}
	ctrl, err := controller.New(cfg)
	if err != nil {
		return fmt.Errorf("could not create controller: %w", err)
	}

	// Start our controller.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = ctrl.Run(ctx)
	if err != nil {
		return fmt.Errorf("error running controller: %w", err)
	}

	return nil
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running app: %s", err)
		os.Exit(1)
	}
	
	os.Exit(0)
}