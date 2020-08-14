.PHONY: test clean compile build push

IMAGE=hspitzlerc/kube-sqs-autoscaler
VERSION=latest

test:
	go test ./...

clean:
	rm -f kube-sqs-autoscaler

compile: clean
	GOOS=linux go build .

build: compile
	docker build -t $(IMAGE):$(VERSION) .

push: build
	docker push $(IMAGE):$(VERSION)
