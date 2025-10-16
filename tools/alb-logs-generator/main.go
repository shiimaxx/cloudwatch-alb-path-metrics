package main

import (
	"flag"
	"fmt"
	"math/rand"
	"reflect"
	"time"

	"github.com/go-faker/faker/v4"
)

var startTime = time.Now().UTC().Add(-5 * time.Minute)
var flagCount int

func init() {
	_ = faker.AddProvider("custom_alb_time", func(v reflect.Value) (any, error) {
		offset := time.Duration(rand.Intn(300)) * time.Second
		return startTime.Add(offset).Format(time.RFC3339Nano), nil
	})

	_ = faker.AddProvider("custom_alb_request", func(v reflect.Value) (any, error) {
		requests := []string{
			"GET https://example.com:443/ HTTP/1.1",
			"GET https://example.com:443/users/123 HTTP/1.1",
			"GET https://admin.example.com:443/ HTTP/1.1",
			"GET https://admin.example.com:443/dashboard HTTP/1.1",
		}
		return requests[rand.Intn(len(requests)-1)], nil
	})

	_ = faker.AddProvider("custom_alb_target_group_arn", func(v reflect.Value) (any, error) {
		return "arn:aws:elasticloadbalancing:region:account-id:targetgroup/my-targets/1234567890abcdef", nil
	})

	_ = faker.AddProvider("custom_alb_chosen_cert_arn", func(v reflect.Value) (any, error) {
		return "arn:aws:acm:region:account-id:certificate/12345678-1234-1234-1234-123456789012", nil
	})

	flag.IntVar(&flagCount, "count", 300, "number of log lines to generate")
	flag.Parse()
}

type FakeALBLogFields struct {
	Type                   string  `faker:"oneof: https"`
	Time                   string  `faker:"custom_alb_time"`
	ELB                    string  `faker:"oneof: app/prod-alb/50dc6c495c0c9188"`
	ClientIP               string  `faker:"ipv4"`
	ClientPort             int     `faker:"boundary_start=1024, boundary_end=65535"`
	TargetIP               string  `faker:"oneof: 192.0.2.10, 192.0.2.11, 192.0.2.12"`
	TargetPort             string  `faker:"oneof: 8080"`
	RequestProcessingTime  float64 `faker:"boundary_start=0.5, boundary_end=1.0"`
	TargetProcessingTime   float64 `faker:"boundary_start=0.5, boundary_end=1.0"`
	ResponseProcessingTime float64 `faker:"boundary_start=0.5, boundary_end=1.0"`
	ELBStatusCode          int     `faker:"oneof: 200"`
	TargetStatusCode       int     `faker:"oneof: 200"`
	ReceivedBytes          int     `faker:"oneof: 0, 100, 500, 1000, 2000, 5000"`
	SentBytes              int     `faker:"oneof: 0, 500, 1000, 5000, 10000, 50000"`
	Request                string  `faker:"custom_alb_request"`
	UserAgent              string  `faker:"user_agent"`
	SSLCipher              string  `faker:"oneof: ECDHE-RSA-AES128-GCM-SHA256"`
	SSLProtocol            string  `faker:"oneof: TLSv1.2, TLSv1.3"`
	TargetGroupARN         string  `faker:"custom_alb_target_group_arn"`
	TraceID                string  `faker:"oneof: Root=1-58337281-1d84f3d73c47ec4e58577259"`
	DomainName             string  `faker:"oneof: www.example.com, admin.example.com"`
	ChosenCertARN          string  `faker:"custom_alb_chosen_cert_arn"`
	MatchedRulePriority    string  `faker:"oneof: 0, 1, 10, 100, 1000"`
	RequestCreationTime    string  `faker:"custom_alb_time"`
	ActionsExecuted        string  `faker:"oneof: -"`
	RedirectURL            string  `faker:"oneof: -"`
	ErrorReason            string  `faker:"oneof: -"`
	PortList               int     `faker:"oneof: 8080"`
	TargetStatusCodeList   int     `faker:"oneof: 200, 201, 204, 400, 500"`
	Classification         string  `faker:"oneof: -"`
	ClassificationReason   string  `faker:"oneof: -"`
	ConnTraceID            string  `faker:"oneof: TID_1234abcd5678ef90"`
}

func (f FakeALBLogFields) String() string {
	return fmt.Sprintf(
		"%s %s %s %s:%d %s:%s %.3f %.3f %.3f %d %d %d %d \"%s\" \"%s\" %s %s %s \"%s\" \"%s\" \"%s\" %s %s \"%s\" \"%s\" \"%s\" \"%s:%d\" \"%d\" \"%s\" \"%s\" %s",
		f.Type,
		f.Time,
		f.ELB,
		f.ClientIP,
		f.ClientPort,
		f.TargetIP,
		f.TargetPort,
		f.RequestProcessingTime,
		f.TargetProcessingTime,
		f.ResponseProcessingTime,
		f.ELBStatusCode,
		f.TargetStatusCode,
		f.ReceivedBytes,
		f.SentBytes,
		f.Request,
		f.UserAgent,
		f.SSLCipher,
		f.SSLProtocol,
		f.TargetGroupARN,
		f.TraceID,
		f.DomainName,
		f.ChosenCertARN,
		f.MatchedRulePriority,
		f.RequestCreationTime,
		f.ActionsExecuted,
		f.RedirectURL,
		f.ErrorReason,
		f.TargetIP,
		f.PortList,
		f.TargetStatusCodeList,
		f.Classification,
		f.ClassificationReason,
		f.ConnTraceID,
	)
}

func main() {
	for i := 0; i < flagCount; i++ {
		var logEntry FakeALBLogFields
		err := faker.FakeData(&logEntry)
		if err != nil {
			fmt.Println("Error generating fake data:", err)
			return
		}

		fmt.Print(logEntry.String())
		if i < flagCount-1 {
			fmt.Print("\n")
		}
	}
}
