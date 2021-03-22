package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
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


type (
	endPoint struct {
		name      string
		port      corev1.ContainerPort
		pod       *corev1.Pod
		container corev1.Container
	}	

	podEventsHandler struct {
		// I need a fast lookup for not relevant events in the cluster
		// and a map of endpoints
		processedPods map[string]*corev1.Pod
		endPoints     map[string]endPoint
	}
)

func getKeys(m map[string]endPoint) []string {
	keys := make([]string, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys	
}

func NewPodEventsHandler() *podEventsHandler {
	podEventsHandler := &podEventsHandler {
		processedPods: map[string]*corev1.Pod{},
		endPoints:     map[string]endPoint{},
	}

	m := http.NewServeMux()
   	m.HandleFunc("/", podEventsHandler.muxHandleFunc)
   	s := &http.Server{
	   Addr:    ":80",
	   Handler: m,
	}

   	go s.ListenAndServe()
   	return podEventsHandler
}

func (h *podEventsHandler) showList(w http.ResponseWriter, r *http.Request, err string) {
	// Cutting corners: not thread safe
	msg := fmt.Sprintf("I do not have '%s'\nI have %v\n", err, getKeys(h.endPoints))
	w.Write([]byte(msg))
}

func (h *podEventsHandler) muxHandleFunc(w http.ResponseWriter, r *http.Request)  {
	endPoints := h.endPoints
	urlPath := r.URL.Path
	if len(urlPath) < 1 {
		h.showList(w, r, "")
		return
	}
	urlPath = urlPath[1:]
	// Cutting corners: not thread safe
	endPoint, ok := endPoints[urlPath];
	if !ok {
		h.showList(w, r, urlPath)
		return
	}

	// Cutting corners: I need a proxy here, I/O streaming, load balancer, etc
	// Meanwhile only HTTP GET, no URL params
	ipAddr := fmt.Sprintf("http://%s:%d", endPoint.pod.Status.PodIP, endPoint.port.ContainerPort)
	resp, _ := http.Get(ipAddr)
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	w.Write(body)
}

func (h *podEventsHandler) fullName(pod *corev1.Pod) string {
	podName, podNamespace := pod.Name, pod.Namespace
	return fmt.Sprintf("%s/%s", podNamespace, podName)
}

func (h *podEventsHandler) fullNameContainer(pod *corev1.Pod, container corev1.Container) string {
	podName, podNamespace, containerName := pod.Name, pod.Namespace, container.Name
	return fmt.Sprintf("%s/%s/%s", podNamespace, podName, containerName)
}

// See if there are declared ports in a newly added pod
func (h *podEventsHandler) addPod(pod *corev1.Pod) {
	fullName := h.fullName(pod)
	// Cutting corners: not thread safe
	if _, ok := h.processedPods[fullName];ok {
		return 
	}
	podStatus := pod.Status
	logger.Infof("Processing pod %s phase %s", fullName, podStatus.Phase)
	h.processedPods[fullName] = pod
	podSpec := pod.Spec
	podContainers := podSpec.Containers
	for _, container := range(podContainers) {
		containerName, containerPorts := h.fullNameContainer(pod, container), container.Ports
		if len(containerPorts) == 0 {
			continue
		}

		logger.Infof("Container %s, IP %v, ports %v", containerName, podStatus.PodIP, containerPorts)
		h.endPoints[fullName] = endPoint{
			name:      containerName,
			port:      containerPorts[0],
			pod:       pod,
			container: container,
		}	
	}
}

func (h *podEventsHandler) removePod(pod *corev1.Pod) {
	fullName, podStatus := h.fullName(pod), pod.Status
	// Cutting corners: not thread safe
	if _, ok := h.processedPods[fullName];!ok {
		return 
	}
	logger.Infof("Removing pod %s phase %s", fullName, podStatus.Phase)
	delete(h.processedPods, fullName)
	delete(h.endPoints, fullName)
}

// Track the pods life cycle
func (h *podEventsHandler) handler(_ context.Context, obj runtime.Object) error {
	pod := obj.(*corev1.Pod)
	podStatus := pod.Status
	switch podStatus.Phase {
	case corev1.PodRunning:
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

	podEventsHandler := NewPodEventsHandler()
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

