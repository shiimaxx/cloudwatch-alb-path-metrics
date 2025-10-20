GOOS := linux
GOARCH := arm64

dist/cloudwatch-alb-path-metrics.zip: dist/bootstrap
	cd dist && zip cloudwatch-alb-path-metrics.zip bootstrap

dist/bootstrap:
	GOOS=$(GOOS) GOARCH=$(GOARCH)	go build -tags lambda.norpc -o $@ ./cmd/cloudwatch-alb-path-metrics/main.go

.PHONY: clean
clean:
	rm -f dist/*

