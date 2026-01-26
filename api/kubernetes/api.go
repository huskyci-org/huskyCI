package kubernetes

import (
	"errors"
	"fmt"
	"io"

	apiContext "github.com/huskyci-org/huskyCI/api/context"
	"github.com/huskyci-org/huskyCI/api/log"
	goContext "golang.org/x/net/context"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	kube "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Kubernetes is the Kubernetes struct
type Kubernetes struct {
	PID              string `json:"Id"`
	client           *kube.Clientset
	Namespace        string
	ProxyAddress     string
	NoProxyAddresses string
}

const logActionNew = "NewKubernetes"
const logInfoAPI = "KUBERNETES"

// NewKubernetes creates a new Kubernetes client instance.
func NewKubernetes() (*Kubernetes, error) {
	configAPI, err := apiContext.DefaultConf.GetAPIConfig()
	if err != nil {
		log.Error(logActionNew, logInfoAPI, 3026, err)
		return nil, err
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", configAPI.KubernetesConfig.ConfigFilePath)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kube.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	kubernetes := &Kubernetes{
		client:           clientset,
		Namespace:        configAPI.KubernetesConfig.Namespace,
		ProxyAddress:     configAPI.KubernetesConfig.ProxyAddress,
		NoProxyAddresses: configAPI.KubernetesConfig.NoProxyAddresses,
	}

	return kubernetes, nil

}

// CreatePod creates a new Kubernetes pod with the specified image, command, and configuration.
func (k Kubernetes) CreatePod(image, cmd, podName, securityTestName string) (string, error) {

	ctx := goContext.Background()

	podToCreate := &core.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"name":    podName,
				"huskyCI": securityTestName,
			},
		},
		Spec: core.PodSpec{
			Containers: []core.Container{
				{
					Name:            podName,
					Image:           image,
					ImagePullPolicy: core.PullIfNotPresent,
					Command: []string{
						"/bin/sh",
						"-c",
						cmd,
					},
					Env: []core.EnvVar{
						{
							Name:  "http_proxy",
							Value: k.ProxyAddress,
						},
						{
							Name:  "https_proxy",
							Value: k.ProxyAddress,
						},
						{
							Name:  "no_proxy",
							Value: k.NoProxyAddresses,
						},
					},
				},
			},
			TopologySpreadConstraints: []core.TopologySpreadConstraint{
				{
					MaxSkew:           1,
					TopologyKey:       "kubernetes.io/hostname",
					WhenUnsatisfiable: "ScheduleAnyway",
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"huskyCI": securityTestName,
						},
					},
				},
			},
			RestartPolicy: "Never",
		},
	}

	pod, err := k.client.CoreV1().Pods(k.Namespace).Create(ctx, podToCreate, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return string(pod.UID), nil
}

// WaitPod waits for a pod to be scheduled and complete execution, with configurable timeouts.
func (k Kubernetes) WaitPod(name string, podSchedulingTimeoutInSeconds, testTimeOutInSeconds int) (string, error) {
	ctx := goContext.Background()

	// Wait for pod scheduling
	scheduled, phase, err := k.waitForPodScheduling(ctx, name, podSchedulingTimeoutInSeconds)
	if err != nil {
		return "", err
	}
	if !scheduled {
		return k.handleSchedulingTimeout(name)
	}
	if phase != "" {
		return phase, nil
	}

	// Wait for pod completion
	return k.waitForPodCompletion(ctx, name, testTimeOutInSeconds)
}

func (k Kubernetes) waitForPodScheduling(ctx goContext.Context, name string, timeoutSeconds int) (bool, string, error) {
	timeout := int64Ptr(int64(timeoutSeconds))
	watchScheduling, err := k.client.CoreV1().Pods(k.Namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector:  fmt.Sprintf("name=%s", name),
		Watch:          true,
		TimeoutSeconds: timeout,
	})
	if err != nil {
		return false, "", err
	}
	defer watchScheduling.Stop()

	for event := range watchScheduling.ResultChan() {
		pod, err := extractPodFromEvent(event)
		if err != nil {
			return false, "", err
		}

		phase, err := handleSchedulingPhase(pod.Status.Phase)
		if err != nil {
			return false, "", err
		}
		if phase != "" {
			return true, phase, nil
		}
		if pod.Status.Phase == "Running" {
			return true, "", nil
		}
	}

	return false, "", nil
}

func (k Kubernetes) waitForPodCompletion(ctx goContext.Context, name string, timeoutSeconds int) (string, error) {
	timeout := int64Ptr(int64(timeoutSeconds))
	watchRunning, err := k.client.CoreV1().Pods(k.Namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector:  fmt.Sprintf("name=%s", name),
		Watch:          true,
		TimeoutSeconds: timeout,
	})
	if err != nil {
		return "", err
	}
	defer watchRunning.Stop()

	for event := range watchRunning.ResultChan() {
		pod, err := extractPodFromEvent(event)
		if err != nil {
			return "", err
		}

		phase, err := handleCompletionPhase(pod.Status.Phase)
		if err != nil {
			return "", err
		}
		if phase != "" {
			watchRunning.Stop()
			return phase, nil
		}
	}

	return k.handleCompletionTimeout(name)
}

func extractPodFromEvent(event watch.Event) (*core.Pod, error) {
	pod, ok := event.Object.(*core.Pod)
	if !ok {
		return nil, errors.New("Unexpected Event while waiting for Pod")
	}
	return pod, nil
}

func handleSchedulingPhase(phase core.PodPhase) (string, error) {
	switch string(phase) {
	case "Succeeded", "Completed":
		return string(phase), nil
	case "Failed":
		return "", errors.New("Pod execution failed")
	case "Unknown":
		return "", errors.New("Pod terminated with Unknown status")
	default:
		return "", nil
	}
}

func handleCompletionPhase(phase core.PodPhase) (string, error) {
	switch string(phase) {
	case "Succeeded", "Completed":
		return string(phase), nil
	case "Failed":
		return "", errors.New("Pod execution failed")
	case "Unknown":
		return "", errors.New("Pod terminated with Unknown status")
	default:
		return "", nil
	}
}

func (k Kubernetes) handleSchedulingTimeout(name string) (string, error) {
	if err := k.RemovePod(name); err != nil {
		return "", err
	}
	return "", fmt.Errorf("timed-out waiting for pod scheduling: %s", name)
}

func (k Kubernetes) handleCompletionTimeout(name string) (string, error) {
	if err := k.RemovePod(name); err != nil {
		return "", err
	}
	return "", fmt.Errorf("timed-out waiting for pod to finish: %s", name)
}

func int64Ptr(i int64) *int64 {
	return &i
}

// ReadOutput reads the logs from a Kubernetes pod.
func (k Kubernetes) ReadOutput(name string) (string, error) {
	ctx := goContext.Background()

	req := k.client.CoreV1().Pods(k.Namespace).GetLogs(name, &core.PodLogOptions{})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		errRemovePod := k.RemovePod(name)
		if errRemovePod != nil {
			return "", errRemovePod
		}
		return "", err
	}
	defer podLogs.Close()

	result, err := io.ReadAll(podLogs)
	if err != nil {
		errRemovePod := k.RemovePod(name)
		if errRemovePod != nil {
			return "", errRemovePod
		}
		return "", err
	}

	return string(result), nil
}

// RemovePod deletes a Kubernetes pod by name.
func (k Kubernetes) RemovePod(name string) error {
	ctx := goContext.Background()

	return k.client.CoreV1().Pods(k.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// HealthCheckKubernetesAPI returns true if a 200 status code is received from kubernetes or false otherwise.
func HealthCheckKubernetesAPI() error {
	k, err := NewKubernetes()
	if err != nil {
		log.Error("HealthCheckKubernetesAPI", logInfoAPI, 3011, err)
		return err
	}

	ctx := goContext.Background()
	_, err = k.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error("HealthCheckKubernetesAPI", logInfoAPI, 3011, err)
		return err
	}
	return nil
}
