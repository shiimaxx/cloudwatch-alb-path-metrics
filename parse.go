package main

import (
	"encoding/csv"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type albLogEntry struct {
	timestamp time.Time
	method    string
	host      string
	path      string
	status    int
	duration  float64
}

// AWS ALB log field constants (0-based indices)
// Fields:
// type time elb client:port target:port request_processing_time target_processing_time response_processing_time
// elb_status_code target_status_code received_bytes sent_bytes "request" "user_agent" ssl_cipher ssl_protocol
// target_group_arn "trace_id" "domain_name" "chosen_cert_arn" matched_rule_priority request_creation_time
// "actions_executed" "redirect_url" "error_reason" "target:port_list" "target_status_code_list"
// "classification" "classification_reason" conn_trace_id
const (
	timestampFieldIndex              = 1
	requestProcessingTimeFieldIndex  = 5
	targetProcessingTimeFieldIndex   = 6
	responseProcessingTimeFieldIndex = 7
	statusFieldIndex                 = 8
	requestFieldIndex                = 12
)

func parseALBLogFields(fields []string) (*albLogEntry, error) {
	if len(fields) <= requestFieldIndex {
		return nil, errors.New("invalid ALB log entry: missing required fields")
	}

	// Parse timestamp (ISO 8601 format: 2018-07-02T22:23:00.186641Z)
	timestamp, err := time.Parse(time.RFC3339Nano, fields[timestampFieldIndex])
	if err != nil {
		return nil, errors.New("failed to parse timestamp: " + err.Error())
	}

	status, err := strconv.Atoi(fields[statusFieldIndex])
	if err != nil {
		return nil, errors.New("failed to parse status: " + err.Error())
	}

	requestProcessingTime, err := strconv.ParseFloat(fields[requestProcessingTimeFieldIndex], 64)
	if err != nil {
		return nil, errors.New("failed to parse request processing time: " + err.Error())
	}

	targetProcessingTime, err := strconv.ParseFloat(fields[targetProcessingTimeFieldIndex], 64)
	if err != nil {
		return nil, errors.New("failed to parse target processing time: " + err.Error())
	}

	responseProcessingTime, err := strconv.ParseFloat(fields[responseProcessingTimeFieldIndex], 64)
	if err != nil {
		return nil, errors.New("failed to parse response processing time: " + err.Error())
	}

	duration := requestProcessingTime + targetProcessingTime + responseProcessingTime

	requestParts := strings.Fields(fields[requestFieldIndex])
	if len(requestParts) < 3 {
		return nil, errors.New("invalid request field format")
	}

	method := requestParts[0]
	urlStr := requestParts[1]

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, errors.New("failed to parse request URL: " + err.Error())
	}

	return &albLogEntry{
		timestamp: timestamp,
		method:    method,
		host:      u.Hostname(),
		path:      u.Path,
		status:    status,
		duration:  duration,
	}, nil
}

func parseALBLogLine(line string) (*albLogEntry, error) {
	reader := csv.NewReader(strings.NewReader(line))
	reader.Comma = ' '
	reader.ReuseRecord = true

	fields, err := reader.Read()
	if err != nil {
		return nil, errors.New("failed to parse ALB log line: " + err.Error())
	}

	return parseALBLogFields(fields)
}
