package main

import (
	"encoding/csv"
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"
)

type albLogEntry struct {
	method   string
	host     string
	path     string
	status   int
	duration float64
}

// AWS ALB log field constants (0-based indices)
// Fields: type time elb client:port target:port request_processing_time target_processing_time response_processing_time
//
//	elb_status_code target_status_code received_bytes sent_bytes "request" "user_agent" ssl_cipher ssl_protocol
//	target_group_arn "trace_id" "domain_name" "chosen_cert_arn" matched_rule_priority request_creation_time
//	"actions_executed" "redirect_url" "error_reason" "target:port_list" "target_status_code_list"
//	"classification" "classification_reason" conn_trace_id
const (
	durationFieldIndex = 6
	statusFieldIndex   = 8
	requestFieldIndex  = 12
)

func parseALBLogFields(fields []string) (*albLogEntry, error) {
	status, err := strconv.Atoi(fields[statusFieldIndex])
	if err != nil {
		return nil, errors.New("failed to parse status: " + err.Error())
	}

	duration, err := strconv.ParseFloat(fields[durationFieldIndex], 64)
	if err != nil {
		return nil, errors.New("failed to parse duration: " + err.Error())
	}

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
		method:   method,
		host:     u.Host,
		path:     u.Path,
		status:   status,
		duration: duration,
	}, nil
}

func parseALBLogFile(logContent string) ([]*albLogEntry, error) {
	reader := csv.NewReader(strings.NewReader(logContent))
	reader.Comma = ' '
	reader.ReuseRecord = true

	var entries []*albLogEntry

	for {
		fields, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.New("failed to parse ALB log file: " + err.Error())
		}

		entry, err := parseALBLogFields(fields)
		if err != nil {
			// Skip invalid lines instead of failing the entire parse
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}
