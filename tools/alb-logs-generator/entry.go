package main

import "time"

type IrrelevantFields struct {
	LogType     string
	ELBName     string
	ClientIP    string
	ClientPort  int
	TargetIP    string
	TargetPort  int
	DomainName  string
	UserAgent   string
	TargetGroup string
}

type RelevantFields struct {
	RequestProcessingTime  float64
	TargetProcessingTime   float64
	ResponseProcessingTime float64
	ELBStatusCode          int
	TargetStatusCode       int
	RequestLine            string
}

type ALBLogEntry struct {
	Timestamp        time.Time
	IrrelevantFields IrrelevantFields
	RelevantFields   RelevantFields
}

func ComposeEntry(ts time.Time, irr IrrelevantFields, rel RelevantFields) ALBLogEntry {
	return ALBLogEntry{
		Timestamp:        ts,
		IrrelevantFields: irr,
		RelevantFields:   rel,
	}
}

func (e ALBLogEntry) String() string {
	return ""
}
