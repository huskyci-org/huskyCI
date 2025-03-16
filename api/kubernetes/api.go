package kubernetes

import (
	"errors"
	"fmt"
	"io/ioutil"

	apiContext "github.com/huskyci-org/huskyCI/api/context"
	"github.com/huskyci-org/huskyCI/api/log"
	goContext "golang.org/x/net/context"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (k Kubernetes) WaitPod(name string, podSchedulingTimeoutInSeconds, testTimeOutInSeconds int) (string, error) {

	ctx := goContext.Background()

	timeout := func(i int64) *int64 { return &i }(int64(podSchedulingTimeoutInSeconds))
	schedulingTimeout := true

	watchScheduling, err := k.client.CoreV1().Pods(k.Namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector:  fmt.Sprintf("name=%s", name),
		Watch:          true,
		TimeoutSeconds: timeout,
	})
	if err != nil {
		return "", err
	}

schedulingLoop:
	for event := range watchScheduling.ResultChan() {
		p, ok := event.Object.(*core.Pod)
		if !ok {
			return "", errors.New("Unexpected Event while waiting for Pod")
		}

		switch p.Status.Phase {
		case "Running":
			schedulingTimeout = false
			watchScheduling.Stop()
			break schedulingLoop
		case "Succeeded", "Completed":
			return string(p.Status.Phase), nil
		case "Failed":
			return "", errors.New("Pod execution failed")
		case "Unknown":
			return "", errors.New("Pod terminated with Unknown status")
		}
	}

	if schedulingTimeout {
		err = k.RemovePod(name)
		if err != nil {
			return "", err
		}

		return "", errors.New(fmt.Sprintf("Timed-out waiting for pod scheduling: %s", name))
	}

	timeoutResult := func(i int64) *int64 { return &i }(int64(testTimeOutInSeconds))
	watchRunning, err := k.client.CoreV1().Pods(k.Namespace).Watch(ctx, metav1.ListOptions{
		LabelSelector:  fmt.Sprintf("name=%s", name),
		Watch:          true,
		TimeoutSeconds: timeoutResult,
	})
	if err != nil {
		return "", err
	}

	for event := range watchRunning.ResultChan() {
		p, ok := event.Object.(*core.Pod)
		if !ok {
			return "", errors.New("Unexpected Event while waiting for Pod")
		}

		switch p.Status.Phase {
		case "Succeeded", "Completed":
			watchRunning.Stop()
			return string(p.Status.Phase), nil
		case "Failed":
			return "", errors.New("Pod execution failed")
		case "Unknown":
			return "", errors.New("Pod terminated with Unknown status")
		}
	}

	err = k.RemovePod(name)
	if err != nil {
		return "", err
	}

	return "", errors.New(fmt.Sprintf("Timed-out waiting for pod to finish: %s", name))
}

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

	result, err := ioutil.ReadAll(podLogs)
	if err != nil {
		errRemovePod := k.RemovePod(name)
		if errRemovePod != nil {
			return "", errRemovePod
		}
		return "", err
	}

	return string(result), nil
}

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
