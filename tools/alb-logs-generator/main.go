package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/go-faker/faker/v4"
)

const (
	defaultRPS     = 10.0
	logTimeFormat  = "2006-01-02T15:04:05.000000Z"
	windowDuration = 5 * time.Minute
	windowSeconds  = 300
)

var (
	targetGroupARNs = []string{
		"arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/tg-main/aaaabbbbccccdddd",
		"arn:aws:elasticloadbalancing:us-west-2:210987654321:targetgroup/tg-blue/bbbbccccddddeeee",
	}
	chosenCertARNs = []string{
		"-",
		"arn:aws:acm:us-east-1:123456789012:certificate/cert-1234abcd",
		"arn:aws:acm:us-west-2:210987654321:certificate/cert-5678efgh",
	}
	httpListenerPorts  = []int{80, 8080}
	httpsListenerPorts = []int{443, 8443}
)

type albLogEntry struct {
	Type                   string
	Timestamp              time.Time
	ELB                    string
	ClientAddr             string
	TargetAddr             string
	RequestProcessingTime  float64
	TargetProcessingTime   float64
	ResponseProcessingTime float64
	ELBStatusCode          int
	TargetStatusCode       int
	ReceivedBytes          int
	SentBytes              int
	RequestLine            string
	UserAgent              string
	SSLCipher              string
	SSLProtocol            string
	TargetGroupArn         string
	TraceID                string
	DomainName             string
	ChosenCertArn          string
	MatchedRulePriority    string
	RequestCreationTime    time.Time
	ActionsExecuted        string
	RedirectURL            string
	ErrorReason            string
	TargetPortList         string
	TargetStatusCodeList   string
	Classification         string
	ClassificationReason   string
}

func (e albLogEntry) String() string {
	return fmt.Sprintf(
		"%s %s %s %s %s %.6f %.6f %.6f %d %d %d %d %q %q %s %s %s %q %q %q %s %s %q %q %q %q %q %q %q",
		e.Type,
		e.Timestamp.Format(logTimeFormat),
		e.ELB,
		e.ClientAddr,
		e.TargetAddr,
		e.RequestProcessingTime,
		e.TargetProcessingTime,
		e.ResponseProcessingTime,
		e.ELBStatusCode,
		e.TargetStatusCode,
		e.ReceivedBytes,
		e.SentBytes,
		e.RequestLine,
		e.UserAgent,
		e.SSLCipher,
		e.SSLProtocol,
		e.TargetGroupArn,
		e.TraceID,
		e.DomainName,
		e.ChosenCertArn,
		e.MatchedRulePriority,
		e.RequestCreationTime.Format(logTimeFormat),
		e.ActionsExecuted,
		e.RedirectURL,
		e.ErrorReason,
		e.TargetPortList,
		e.TargetStatusCodeList,
		e.Classification,
		e.ClassificationReason,
	)
}

func main() {
	seedFlag := flag.Int64("seed", time.Now().UnixNano(), "seed for synthetic data generation")
	countFlag := flag.Int("count", 0, "number of log entries to emit (default: derived from --rps)")
	rpsFlag := flag.Float64("rps", defaultRPS, "average requests per second over the five-minute window")
	startFlag := flag.String("start", "", "start time (RFC3339) for the five-minute window; defaults to now minus five minutes")
	flag.Parse()

	if *countFlag == 0 && *rpsFlag <= 0 {
		log.Fatalf("rps must be positive when count is not specified: %.2f", *rpsFlag)
	}
	if *countFlag < 0 {
		log.Fatalf("count must be non-negative: %d", *countFlag)
	}

	startTime := resolveStartTime(*startFlag)
	entryCount := resolveEntryCount(*countFlag, *rpsFlag)

	fakerSource := rand.NewSource(*seedFlag)
	faker.SetRandomSource(faker.NewSafeSource(fakerSource))
	dataRand := rand.New(rand.NewSource(*seedFlag))

	entries := generateEntries(entryCount, startTime, dataRand)
	writer := bufio.NewWriter(os.Stdout)
	for _, entry := range entries {
		if _, err := writer.WriteString(entry.String()); err != nil {
			log.Fatalf("failed to write log entry: %v", err)
		}
		if err := writer.WriteByte('\n'); err != nil {
			log.Fatalf("failed to write newline: %v", err)
		}
	}
	if err := writer.Flush(); err != nil {
		log.Fatalf("failed to flush output: %v", err)
	}
}

func generateEntries(count int, start time.Time, rng *rand.Rand) []albLogEntry {
	entries := make([]albLogEntry, 0, count)
	windowNanos := windowDuration.Nanoseconds()
	var step int64
	if count > 1 {
		step = windowNanos / int64(count-1)
	}

	for i := range count {
		offset := time.Duration(step * int64(i))
		timestamp := start.Add(offset)
		template := newEntryTemplate(rng)
		entries = append(entries, buildLogEntry(template, timestamp))
	}

	return entries
}

func resolveStartTime(startFlag string) time.Time {
	if startFlag == "" {
		return time.Now().UTC().Add(-windowDuration)
	}

	startTime, err := time.Parse(time.RFC3339, startFlag)
	if err != nil {
		log.Fatalf("invalid start timestamp %q: %v", startFlag, err)
	}
	return startTime
}

func resolveEntryCount(count int, rps float64) int {
	if count > 0 {
		return count
	}

	derived := int(math.Round(rps * windowSeconds))
	if derived <= 0 {
		log.Fatalf("derived entry count must be positive (rps=%.2f)", rps)
	}
	return derived
}

func newEntryTemplate(rng *rand.Rand) entryTemplate {
	var template entryTemplate
	if err := faker.FakeData(&template); err != nil {
		log.Fatalf("faker failed to populate entry template: %v", err)
	}
	template.TargetGroupArn = randomChoice(rng, targetGroupARNs)
	template.ChosenCertArn = randomChoice(rng, chosenCertARNs)
	template.ListenerPort = listenerPortForType(template.Type, rng)
	return template
}

type entryTemplate struct {
	ClientIP                    string `faker:"ipv4"`
	TargetIP                    string `faker:"ipv4"`
	ClientPort                  int    `faker:"boundary_start=1024, boundary_end=65535"`
	TargetPort                  int    `faker:"oneof:80,443,8080,9000"`
	ListenerPort                int
	Method                      string `faker:"oneof:GET,POST,PUT,PATCH,DELETE"`
	Path                        string `faker:"oneof:/,/health,/login,/logout,/api/orders,/api/users,/static/css/main.css"`
	UserAgent                   string `faker:"user_agent"`
	DomainName                  string `faker:"oneof:www.example.com,api.example.com,static.example.com,auth.example.com"`
	Type                        string `faker:"oneof:http,https"`
	ELB                         string `faker:"oneof:app/prod-web/1a2b3c4d,app/staging-api/2b3c4d5e,app/dev-orders/3c4d5e6f,app/prod-auth/4d5e6f7a"`
	ELBStatusCode               int    `faker:"oneof:200,200,200,200,301,302,400,403,404,500,502,503,504"`
	TargetStatusCode            int    `faker:"oneof:200,200,200,200,301,302,400,403,404,500,502,503,504"`
	ReceivedBytes               int    `faker:"boundary_start=0, boundary_end=8192"`
	SentBytes                   int    `faker:"boundary_start=512, boundary_end=65536"`
	RequestProcessingMicros     int    `faker:"boundary_start=10, boundary_end=15000"`
	TargetProcessingMillis      int    `faker:"boundary_start=1, boundary_end=600"`
	ResponseProcessingMicros    int    `faker:"boundary_start=50, boundary_end=50000"`
	RequestCreationOffsetMillis int    `faker:"boundary_start=750, boundary_end=2500"`
	MatchedRulePriority         string `faker:"oneof:-,1,5,10,25,100"`
	ActionsExecuted             string `faker:"oneof:forward,redirect,authenticate,fixed-response,waf"`
	RedirectPath                string `faker:"oneof:-,/,/login,/home,/dashboard,/orders"`
	ErrorReason                 string `faker:"oneof:-,Target.Response,Target.Timeout,Target.ConnectionError,LambdaInvalidResponse"`
	Classification              string `faker:"oneof:-,waf"`
	ClassificationReason        string `faker:"oneof:-,waf-blocked,rule-match"`
	SSLCipher                   string `faker:"oneof:-,ECDHE-RSA-AES128-GCM-SHA256,ECDHE-RSA-AES256-GCM-SHA384,ECDHE-ECDSA-AES128-GCM-SHA256"`
	SSLProtocol                 string `faker:"oneof:-,TLSv1.2,TLSv1.3"`
	ChosenCertArn               string `faker:"-"`
	TargetGroupArn              string `faker:"-"`
	TraceID                     string `faker:"oneof:Root=1-5f84c3aa-1aa2bb3cc4dd5ee6ff778899"`
}

func randomChoice[T any](rng *rand.Rand, options []T) T {
	if len(options) == 0 {
		log.Fatal("randomChoice called with empty options")
	}
	return options[rng.Intn(len(options))]
}

func listenerPortForType(scheme string, rng *rand.Rand) int {
	if scheme == "https" {
		return randomChoice(rng, httpsListenerPorts)
	}
	return randomChoice(rng, httpListenerPorts)
}

func buildLogEntry(template entryTemplate, timestamp time.Time) albLogEntry {
	reqProc := float64(template.RequestProcessingMicros) / 1_000_000
	tgtProc := float64(template.TargetProcessingMillis) / 1_000
	respProc := float64(template.ResponseProcessingMicros) / 1_000_000
	receivedBytes, sentBytes := template.ReceivedBytes, template.SentBytes
	if sentBytes < receivedBytes {
		sentBytes = receivedBytes
	}
	sslCipher := template.SSLCipher
	sslProtocol := template.SSLProtocol
	certArn := template.ChosenCertArn
	if template.Type != "https" {
		sslCipher, sslProtocol, certArn = "-", "-", "-"
	}
	requestCreation := timestamp.Add(-time.Duration(template.RequestCreationOffsetMillis) * time.Millisecond)
	redirectURL := deriveRedirectURL(template)
	targetPortList := fmt.Sprintf("%s:%d", template.TargetIP, template.TargetPort)
	targetStatusCodeList := fmt.Sprintf("%d", template.TargetStatusCode)
	requestLine := buildRequestLine(template)
	elbCode := template.ELBStatusCode
	targetCode := template.TargetStatusCode
	classification, classificationReason := template.Classification, template.ClassificationReason
	if classification == "-" {
		classificationReason = "-"
	}
	matchedRule := template.MatchedRulePriority
	if matchedRule == "" {
		matchedRule = "-"
	}

	return albLogEntry{
		Type:                   template.Type,
		Timestamp:              timestamp.UTC(),
		ELB:                    template.ELB,
		ClientAddr:             fmt.Sprintf("%s:%d", template.ClientIP, template.ClientPort),
		TargetAddr:             fmt.Sprintf("%s:%d", template.TargetIP, template.TargetPort),
		RequestProcessingTime:  reqProc,
		TargetProcessingTime:   tgtProc,
		ResponseProcessingTime: respProc,
		ELBStatusCode:          elbCode,
		TargetStatusCode:       targetCode,
		ReceivedBytes:          receivedBytes,
		SentBytes:              sentBytes,
		RequestLine:            requestLine,
		UserAgent:              template.UserAgent,
		SSLCipher:              sslCipher,
		SSLProtocol:            sslProtocol,
		TargetGroupArn:         template.TargetGroupArn,
		TraceID:                template.TraceID,
		DomainName:             template.DomainName,
		ChosenCertArn:          certArn,
		MatchedRulePriority:    matchedRule,
		RequestCreationTime:    requestCreation.UTC(),
		ActionsExecuted:        template.ActionsExecuted,
		RedirectURL:            redirectURL,
		ErrorReason:            template.ErrorReason,
		TargetPortList:         targetPortList,
		TargetStatusCodeList:   targetStatusCodeList,
		Classification:         classification,
		ClassificationReason:   classificationReason,
	}
}

func buildRequestLine(template entryTemplate) string {
	scheme := template.Type
	port := template.ListenerPort
	if port == 0 {
		if scheme == "https" {
			port = 443
		} else {
			port = 80
		}
	}
	hostPort := fmt.Sprintf("%s:%d", template.DomainName, port)
	return fmt.Sprintf("%s %s://%s%s HTTP/1.1", template.Method, scheme, hostPort, template.Path)
}

func deriveRedirectURL(template entryTemplate) string {
	if template.ActionsExecuted != "redirect" || template.RedirectPath == "-" {
		return "-"
	}
	scheme := "https"
	if template.Type == "http" {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s%s", scheme, template.DomainName, template.RedirectPath)
}
