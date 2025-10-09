package main

import (
	"fmt"
	"strconv"
	"time"
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
	HasTarget              bool
}

func (e albLogEntry) String() string {
	targetProcessing := fmt.Sprintf("%.6f", e.TargetProcessingTime)
	targetStatus := fmt.Sprintf("%d", e.TargetStatusCode)
	targetAddr := e.TargetAddr
	targetPortList := e.TargetPortList
	targetStatusCodeList := e.TargetStatusCodeList
	targetGroupArn := e.TargetGroupArn

	if !e.HasTarget {
		targetProcessing = "-"
		targetStatus = "-"
		targetAddr = "-"
		targetPortList = "-"
		targetStatusCodeList = "-"
		targetGroupArn = "-"
	}

	return fmt.Sprintf(
		"%s %s %s %s %s %.6f %s %.6f %d %s %d %d %q %q %s %s %s %q %q %q %s %s %q %q %q %q %q %q %q",
		e.Type,
		e.Timestamp.Format(logTimeFormat),
		e.ELB,
		e.ClientAddr,
		targetAddr,
		e.RequestProcessingTime,
		targetProcessing,
		e.ResponseProcessingTime,
		e.ELBStatusCode,
		targetStatus,
		e.ReceivedBytes,
		e.SentBytes,
		e.RequestLine,
		e.UserAgent,
		e.SSLCipher,
		e.SSLProtocol,
		targetGroupArn,
		e.TraceID,
		e.DomainName,
		e.ChosenCertArn,
		e.MatchedRulePriority,
		e.RequestCreationTime.Format(logTimeFormat),
		e.ActionsExecuted,
		e.RedirectURL,
		e.ErrorReason,
		targetPortList,
		targetStatusCodeList,
		e.Classification,
		e.ClassificationReason,
	)
}

func buildLogEntry(template entryTemplate, timestamp time.Time) albLogEntry {
	reqProc := template.RequestProcessingSeconds
	targetProc := template.TargetProcessingSeconds
	respProc := template.ResponseProcessingSeconds
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
	targetAddr := fmt.Sprintf("%s:%d", template.TargetIP, template.TargetPort)
	targetPortList := formatQuotedList([]string{targetAddr})
	targetStatusCodeList := formatQuotedList([]string{strconv.Itoa(template.TargetStatusCode)})
	targetCode := template.TargetStatusCode
	classification, classificationReason := template.Classification, template.ClassificationReason
	if classification == "-" {
		classificationReason = "-"
	}
	matchedRule := template.MatchedRulePriority
	if matchedRule == "" {
		matchedRule = "-"
	}
	hasTarget := actionContactsTarget(template.ActionsExecuted)
	targetGroupArn := template.TargetGroupArn
	if !hasTarget {
		targetPortList = "-"
		targetStatusCodeList = "-"
		targetAddr = "-"
		targetGroupArn = "-"
		targetProc = 0
		targetCode = 0
	}

	return albLogEntry{
		Type:                   template.Type,
		Timestamp:              timestamp.UTC(),
		ELB:                    template.ELB,
		ClientAddr:             fmt.Sprintf("%s:%d", template.ClientIP, template.ClientPort),
		TargetAddr:             targetAddr,
		RequestProcessingTime:  reqProc,
		TargetProcessingTime:   targetProc,
		ResponseProcessingTime: respProc,
		ELBStatusCode:          template.ELBStatusCode,
		TargetStatusCode:       targetCode,
		ReceivedBytes:          receivedBytes,
		SentBytes:              sentBytes,
		RequestLine:            buildRequestLine(template),
		UserAgent:              template.UserAgent,
		SSLCipher:              sslCipher,
		SSLProtocol:            sslProtocol,
		TargetGroupArn:         targetGroupArn,
		TraceID:                template.TraceID,
		DomainName:             template.DomainName,
		ChosenCertArn:          certArn,
		MatchedRulePriority:    matchedRule,
		RequestCreationTime:    timestamp.Add(-time.Duration((reqProc + targetProc + respProc) * float64(time.Second))).UTC(),
		ActionsExecuted:        template.ActionsExecuted,
		RedirectURL:            deriveRedirectURL(template),
		ErrorReason:            template.ErrorReason,
		TargetPortList:         targetPortList,
		TargetStatusCodeList:   targetStatusCodeList,
		Classification:         classification,
		ClassificationReason:   classificationReason,
		HasTarget:              hasTarget,
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
