package cloudwatch

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"

	"github.com/pkg/errors"
)

type CloudWatch interface {
	GetMetricStatistics(*cloudwatch.GetMetricStatisticsInput) (*cloudwatch.GetMetricStatisticsOutput, error)
}

type CloudWatchClient struct {
	Client	CloudWatch
	Queue	string
}

func NewCloudWatchClient(queue string, region string) *CloudWatchClient {
	svc := cloudwatch.New(session.New(), &aws.Config{Region: aws.String(region)})
	return &CloudWatchClient {
		svc,
		queue,
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func int64Ptr(x int64) *int64 {
	return &x
}

func (s *CloudWatchClient) GetQueueMetric(metric string, statistic string) (float64, error) {
	params := &cloudwatch.GetMetricStatisticsInput {
		Dimensions: []*cloudwatch.Dimension{
			&cloudwatch.Dimension{
				Name: aws.String("QueueName"),
				Value: aws.String(s.Queue),
			},
		},
		MetricName: aws.String(metric),
		Namespace: aws.String("AWS/SQS"),
		StartTime: timePtr(time.Now().Add(-time.Minute)),
		EndTime: timePtr(time.Now()),
		Period: int64Ptr(60),
		Statistics: []*string{ aws.String(statistic) },
	}

	out, err := s.Client.GetMetricStatistics(params)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to get queue metrics from Cloudwatch")
	}

	if len(out.Datapoints) == 0 {
		return 0, errors.New("Failed to get queue metric datapoints")
	}

	return *out.Datapoints[len(out.Datapoints) - 1].Sum, nil
}

func (s *CloudWatchClient) Age() (float64, error) {
	return s.GetQueueMetric("ApproximateAgeOfOldestMessage", "Maximum")
}

func (s *CloudWatchClient) NumDeleted() (float64, error) {
	return s.GetQueueMetric("NumberOfMessagesDeleted", "Sum")
}

func (s *CloudWatchClient) NumSent() (float64, error) {
	return s.GetQueueMetric("NumberOfMessagesSent", "Sum")
}
