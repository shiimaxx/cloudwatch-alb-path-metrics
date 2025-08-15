package models

import (
	"strconv"
	"time"
)

type ALBLogEntry struct {
	Type                     string
	Timestamp                time.Time
	ELB                      string
	ClientIP                 string
	ClientPort               int
	TargetIP                 string
	TargetPort               int
	RequestProcessingTime    float64
	TargetProcessingTime     float64
	ResponseProcessingTime   float64
	ELBStatusCode            int
	TargetStatusCode         int
	ReceivedBytes            int64
	SentBytes                int64
	RequestVerb              string
	RequestURL               string
	RequestProto             string
	UserAgent                string
	SSLCipher                string
	SSLProtocol              string
	TargetGroupArn           string
	TraceID                  string
	DomainName               string
	ChosenCertArn            string
	MatchedRulePriority      string
	RequestCreationTime      time.Time
	ActionsExecuted          string
	RedirectURL              string
	ErrorReason              string
	TargetPortList           string
	TargetStatusCodeList     string
	Classification           string
	ClassificationReason     string
	ConnTraceID              string
}

func (e *ALBLogEntry) TotalLatency() float64 {
	return e.RequestProcessingTime + e.TargetProcessingTime + e.ResponseProcessingTime
}

func (e *ALBLogEntry) IsSuccessful() bool {
	return e.ELBStatusCode >= 200 && e.ELBStatusCode <= 499
}

func parseFloat(s string) float64 {
	if s == "-1" || s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func parseInt(s string) int {
	if s == "-1" || s == "" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func parseInt64(s string) int64 {
	if s == "-1" || s == "" {
		return 0
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

func parseTime(s string) time.Time {
	if s == "-" || s == "" {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02T15:04:05.000000Z", s)
	if err != nil {
		return time.Time{}
	}
	return t
}