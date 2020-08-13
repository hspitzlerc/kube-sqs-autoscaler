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

func (p *PodAutoScaler) ScaleUp(numPodsToAdd int32) error {
	deployment, err := p.Client.AppsV1().Deployments(p.Namespace).Get(context.TODO(), p.Deployment, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to get deployment from kube server, no scale up occured")
	}

	currentReplicas := deployment.Spec.Replicas

	if numPodsToAdd < 0 {
		return errors.New("Scaling up by a negative number is not allowed")
	}

	if (*currentReplicas + numPodsToAdd) >= int32(p.Max) {
		return errors.New("Max pods reached")
	}

	deployment.Spec.Replicas = int32Ptr(*currentReplicas + numPodsToAdd)

	_, err = p.Client.AppsV1().Deployments(p.Namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to scale up")
	}

	log.Infof("Scale up successful. Replicas: %d", deployment.Spec.Replicas)
	return nil
}

func (p *PodAutoScaler) ScaleDown(numPodsToRemove int32) error {
	deployment, err := p.Client.AppsV1().Deployments(p.Namespace).Get(context.TODO(), p.Deployment, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to get deployment from kube server, no scale down occured")
	}

	currentReplicas := deployment.Spec.Replicas

	if numPodsToRemove < 0 {
		return errors.New("Scaling down by a negative number is not allowed")
	}

	if (*currentReplicas - numPodsToRemove) <= int32(p.Min) {
		return errors.New("Min pods reached")
	}

	deployment.Spec.Replicas = int32Ptr(*currentReplicas - numPodsToRemove)

	deployment, err = p.Client.AppsV1().Deployments(p.Namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "Failed to scale down")
	}

	log.Infof("Scale down successful. Replicas: %d", deployment.Spec.Replicas)
	return nil
}
