package scale

import (
	"context"

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	restclient "k8s.io/client-go/rest"
	kclient "k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PodAutoScaler struct {
	Client     *kclient.Clientset
	Max        int
	Min        int
	Deployment string
	Namespace  string
}

func NewPodAutoScaler(kubernetesDeploymentName string, kubernetesNamespace string, max int, min int) *PodAutoScaler {
	config, err := restclient.InClusterConfig()
	if err != nil {
		panic("Failed to configure incluster config")
	}

	k8sClient, err := kclient.NewForConfig(config)
	if err != nil {
		panic("Failed to configure client")
	}

	return &PodAutoScaler{
		Client:     k8sClient,
		Min:        min,
		Max:        max,
		Deployment: kubernetesDeploymentName,
		Namespace:  kubernetesNamespace,
	}
}

func int32Ptr(x int32) *int32 {
	return &x
}

func (p *PodAutoScaler) GetPods() (int32, error) {
	deployment, err := p.Client.AppsV1beta2().Deployments(p.Namespace).Get(context.TODO(), p.Deployment, metav1.GetOptions{})
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get deployment from kube server")
	}

	return deployment.Status.AvailableReplicas, nil
}

func (p *PodAutoScaler) Scale(numPods int32) error {
	if numPods < 0 {
		numPods = int32(p.Min)
	}

	if numPods >= int32(p.Max) {
		numPods = int32(p.Max)
	}
	if numPods <= int32(p.Min) {
		numPods = int32(p.Min)
	}

	deployment, err := p.Client.AppsV1().Deployments(p.Namespace).Get(context.TODO(), p.Deployment, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to get deployment from kube server, no scale up occured")
	}

	deployment.Spec.Replicas = int32Ptr(numPods)

	_, err = p.Client.AppsV1().Deployments(p.Namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to scale")
	}

	log.Infof("Scale successful. Replicas: %d", *deployment.Spec.Replicas)
	return nil
}
