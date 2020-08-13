package cloudwatch

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"

	"github.com/pkg/errors"
)

type CloudWatch interface {
	GetMetricData(*cloudwatch.GetMetricDataInput) (*cloudwatch.GetMetricDataOutput, error)
}

type CloudWatchClient struct {
	Client	CloudWatch
	Queue	string
}

func NewCloudWatchClient(queue string, region string) *CloudWatchClient {
	svc := cloudwatch.New(session.New(), &aws.Config{Region: aws.String(region)})
	return &CloudWatchClient {
		svc,
		queue
	}
}

func (s *CloudWatchClient) GetQueueMetric(metric string) (float64, error) {
	params := &cloudwatch.GetMetricStatisticsInput {
		Dimensions: []*cloudwatch.Dimension{
			Name: aws.String("QueueName"),
			Value: aws.String(s.Queue),
		},
		MetricName: aws.String(metric),
		Namespace: aws.String("AWS/SQS"),
		StartTime: time.Now().Add(-time.Minute),
		EndTime: time.Now(),
		Period: 60,
		Statistics: aws.String("Sum"),
	}

	out, err := s.Client.GetMetricStatistics(params)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get queue metrics from Cloudwatch")
	}

	if len(out.Datapoints) == 0 {
		return 0, errors.New("Failed to get queue metric datapoints")
	}

	return out.Datapoints[len(out.Datapoints) - 1].Sum, nil
}
