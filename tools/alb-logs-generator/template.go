package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/go-faker/faker/v4"
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

type entryTemplate struct {
	ClientIP                  string `faker:"ipv4"`
	TargetIP                  string `faker:"ipv4"`
	ClientPort                int    `faker:"boundary_start=1024, boundary_end=65535"`
	TargetPort                int    `faker:"oneof:80,443,8080,9000"`
	ListenerPort              int
	Method                    string `faker:"oneof:GET,POST,PUT,PATCH,DELETE"`
	Path                      string `faker:"oneof:/,/health,/login,/logout,/api/orders,/api/users,/static/css/main.css"`
	UserAgent                 string `faker:"user_agent"`
	DomainName                string `faker:"oneof:www.example.com,api.example.com,static.example.com,auth.example.com"`
	Type                      string `faker:"oneof:http,https"`
	ELB                       string `faker:"oneof:app/prod-web/1a2b3c4d,app/staging-api/2b3c4d5e,app/dev-orders/3c4d5e6f,app/prod-auth/4d5e6f7a"`
	ELBStatusCode             int    `faker:"oneof:200,200,200,200,301,302,400,403,404,500,502,503,504"`
	TargetStatusCode          int    `faker:"oneof:200,200,200,200,301,302,400,403,404,500,502,503,504"`
	ReceivedBytes             int    `faker:"boundary_start=0, boundary_end=8192"`
	SentBytes                 int    `faker:"boundary_start=512, boundary_end=65536"`
	RequestProcessingSeconds  float64
	TargetProcessingSeconds   float64
	ResponseProcessingSeconds float64
	MatchedRulePriority       string `faker:"oneof:-,1,5,10,25,100"`
	ActionsExecuted           string `faker:"oneof:forward,redirect,authenticate,fixed-response,waf"`
	RedirectPath              string `faker:"oneof:-,/,/login,/home,/dashboard,/orders"`
	ErrorReason               string `faker:"oneof:-,Target.Response,Target.Timeout,Target.ConnectionError,LambdaInvalidResponse"`
	Classification            string `faker:"oneof:-,waf"`
	ClassificationReason      string `faker:"oneof:-,waf-blocked,rule-match"`
	SSLCipher                 string `faker:"oneof:-,ECDHE-RSA-AES128-GCM-SHA256,ECDHE-RSA-AES256-GCM-SHA384,ECDHE-ECDSA-AES128-GCM-SHA256"`
	SSLProtocol               string `faker:"oneof:-,TLSv1.2,TLSv1.3"`
	ChosenCertArn             string `faker:"-"`
	TargetGroupArn            string `faker:"-"`
	TraceID                   string `faker:"oneof:Root=1-5f84c3aa-1aa2bb3cc4dd5ee6ff778899"`
}

func newEntryTemplate(rng *rand.Rand) entryTemplate {
	var template entryTemplate
	if err := faker.FakeData(&template); err != nil {
		log.Fatalf("faker failed to populate entry template: %v", err)
	}
	template.TargetGroupArn = randomChoice(rng, targetGroupARNs)
	template.ChosenCertArn = randomChoice(rng, chosenCertARNs)
	template.ListenerPort = listenerPortForType(template.Type, rng)
	template.RequestProcessingSeconds = randomSeconds(rng, 0.00001, 0.015)
	template.TargetProcessingSeconds = randomSeconds(rng, 0.001, 0.6)
	template.ResponseProcessingSeconds = randomSeconds(rng, 0.00005, 0.05)
	return template
}

func randomChoice[T any](rng *rand.Rand, options []T) T {
	if len(options) == 0 {
		log.Fatal("randomChoice called with empty options")
	}
	return options[rng.Intn(len(options))]
}

func randomSeconds(rng *rand.Rand, min, max float64) float64 {
	if max <= min {
		return min
	}
	return min + rng.Float64()*(max-min)
}

func listenerPortForType(scheme string, rng *rand.Rand) int {
	if scheme == "https" {
		return randomChoice(rng, httpsListenerPorts)
	}
	return randomChoice(rng, httpListenerPorts)
}

func actionContactsTarget(action string) bool {
	switch action {
	case "redirect", "fixed-response", "authenticate", "waf":
		return false
	default:
		return true
	}
}

func formatQuotedList(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	return fmt.Sprintf("\"%s\"", strings.Join(items, "\",\""))
}
