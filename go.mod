module github.com/hspitzlerc/kube-sqs-autoscaler

go 1.14

require (
	github.com/aws/aws-sdk-go v1.34.4
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
)
