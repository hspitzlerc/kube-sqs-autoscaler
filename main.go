package main

import (
	"flag"
	"time"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/hspitzlerc/kube-sqs-autoscaler/scale"
	"github.com/hspitzlerc/kube-sqs-autoscaler/sqs"
	"github.com/hspitzlerc/kube-sqs-autoscaler/cloudwatch"
)

var (
	pollInterval        time.Duration
	scaleDownCoolPeriod time.Duration
	scaleUpCoolPeriod   time.Duration
	scaleUpMessages     int
	scaleDownMessages   int
	acceptableAge       float64
	maxPods             int
	minPods             int
	awsRegion           string

	sqsQueueUrl              string
	sqsQueueName             string
	kubernetesDeploymentName string
	kubernetesNamespace      string

	lastPodRate		float64
)

func Run(p *scale.PodAutoScaler, sqs *sqs.SqsClient, cloudwatch *cloudwatch.CloudWatchClient) {
	lastScaleUpTime := time.Now()
	lastScaleDownTime := time.Now()

	for {
		select {
		case <-time.After(pollInterval):
			{
				oldestMessage, err := cloudwatch.Age()
				if err != nil {
					log.Errorf("Failed to get oldest message age: %v", err)
					continue
				}

				messagesProcessed, err := cloudwatch.NumDeleted()
				if err != nil {
					log.Errorf("Failed to get number of messages processed: %v", err)
					continue
				}

				messagesIncoming, err := cloudwatch.NumSent()
				if err != nil {
					log.Errorf("Failed to get number of messages sent: %v", err)
					continue
				}

				numMessages, err := sqs.NumMessages()
				if err != nil {
					log.Errorf("Failed to get SQS messages: %v", err)
					continue
				}


				pods, err := p.GetPods()
				if err != nil {
					log.Errorf("Failed to get number of pods: %v", err)
					continue
				}

				if oldestMessage > acceptableAge {
					messagesIncoming += float64(numMessages) / (acceptableAge / 60.0)
				}

				ratePerPod := messagesProcessed / float64(pods)

				if lastPodRate < ratePerPod {
					lastPodRate = ratePerPod
				}

				if numMessages <= scaleDownMessages {
					podDecrement := int32((messagesIncoming - (lastPodRate * pods)) / lastPodRate)
					if podDecrement < 1 {
						podDecrement = 1
					}

					if lastScaleDownTime.Add(scaleDownCoolPeriod).After(time.Now()) {
						log.Info("Waiting for cool down, skipping scale down")
						continue
					}

					if err := p.Scale(pods - podDecrement); err != nil {
						log.Errorf("Failed scaling down: %v", err)
						continue
					}

					lastScaleDownTime = time.Now()
				} else if numMessages >= scaleUpMessages {
					podIncrement := int32(1)
					if messagesIncoming > messagesProcessed {
						podIncrement = int32((messagesIncoming - messagesProcessed) / lastPodRate)
						if podIncrement < 1 {
							podIncrement = 1
						}
					}
					if lastScaleUpTime.Add(scaleUpCoolPeriod).After(time.Now()) {
						log.Info("Waiting for cool down, skipping scale up ")
						continue
					}
					if err := p.Scale(pods + podIncrement); err != nil {
						log.Errorf("Failed scaling up: %v", err)
						continue
					}

					lastScaleUpTime = time.Now()
				} else {
					lastPodRate = ratePerPod
				}
			}
		}
	}

}

func main() {
	flag.DurationVar(&pollInterval, "poll-period", 5*time.Second, "The interval in seconds for checking if scaling is required")
	flag.DurationVar(&scaleDownCoolPeriod, "scale-down-cool-down", 30*time.Second, "The cool down period for scaling down")
	flag.DurationVar(&scaleUpCoolPeriod, "scale-up-cool-down", 10*time.Second, "The cool down period for scaling up")
	flag.Float64Var(&acceptableAge, "acceptable-age", 150, "Maximum age of messages that can sit in the queue without trigging more aggressive scaling logic, in seconds")
	flag.IntVar(&scaleUpMessages, "scale-up-messages", 100, "Number of sqs messages queued up required for scaling up")
	flag.IntVar(&scaleDownMessages, "scale-down-messages", 10, "Number of sqs messages queued up required to scale down")
	flag.IntVar(&maxPods, "max-pods", 5, "Max pods that kube-sqs-autoscaler can scale")
	flag.IntVar(&minPods, "min-pods", 1, "Min pods that kube-sqs-autoscaler can scale")
	flag.StringVar(&awsRegion, "aws-region", "", "Your AWS region")

	flag.StringVar(&sqsQueueUrl, "sqs-queue-url", "", "The sqs queue url")
	flag.StringVar(&kubernetesDeploymentName, "kubernetes-deployment", "", "Kubernetes Deployment to scale. This field is required")
	flag.StringVar(&kubernetesNamespace, "kubernetes-namespace", "default", "The namespace your deployment is running in")

	flag.Parse()

	sqsQueueComponents := strings.Split(sqsQueueUrl, "/")
	sqsQueueName = sqsQueueComponents[len(sqsQueueComponents) - 1]

	lastPodRate = -1

	p := scale.NewPodAutoScaler(kubernetesDeploymentName, kubernetesNamespace, maxPods, minPods)
	sqs := sqs.NewSqsClient(sqsQueueUrl, awsRegion)
	cloudwatch := cloudwatch.NewCloudWatchClient(sqsQueueName, awsRegion)

	log.Info("Starting kube-sqs-autoscaler")
	Run(p, sqs, cloudwatch)
}
