package alblog

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shiimaxx/cloudwatch-alb-path-metrics/pkg/models"
)

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(r io.Reader) ([]*models.ALBLogEntry, error) {
	var entries []*models.ALBLogEntry
	
	scanner := bufio.NewScanner(r)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		
		entry, err := p.parseLine(line)
		if err != nil {
			return nil, fmt.Errorf("error parsing line %d: %w", lineNum, err)
		}
		
		entries = append(entries, entry)
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}
	
	return entries, nil
}

func (p *Parser) parseLine(line string) (*models.ALBLogEntry, error) {
	fields := strings.Split(line, " ")
	
	if len(fields) < 29 {
		return nil, fmt.Errorf("insufficient fields in log line: expected at least 29, got %d", len(fields))
	}
	
	entry := &models.ALBLogEntry{}
	
	entry.Type = fields[0]
	
	timestamp, err := parseTime(fields[1])
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}
	entry.Timestamp = timestamp
	
	entry.ELB = fields[2]
	entry.ClientIP = parseIP(fields[3])
	entry.ClientPort = parseInt(parsePort(fields[3]))
	entry.TargetIP = parseIP(fields[4])
	entry.TargetPort = parseInt(parsePort(fields[4]))
	
	entry.RequestProcessingTime = parseFloat(fields[5])
	entry.TargetProcessingTime = parseFloat(fields[6])
	entry.ResponseProcessingTime = parseFloat(fields[7])
	
	entry.ELBStatusCode = parseInt(fields[8])
	entry.TargetStatusCode = parseInt(fields[9])
	
	entry.ReceivedBytes = parseInt64(fields[10])
	entry.SentBytes = parseInt64(fields[11])
	
	request := parseQuotedField(fields[12])
	requestParts := strings.Split(request, " ")
	if len(requestParts) >= 3 {
		entry.RequestVerb = requestParts[0]
		entry.RequestURL = requestParts[1]
		entry.RequestProto = requestParts[2]
	}
	
	entry.UserAgent = parseQuotedField(fields[13])
	entry.SSLCipher = parseQuotedField(fields[14])
	entry.SSLProtocol = parseQuotedField(fields[15])
	
	entry.TargetGroupArn = fields[16]
	entry.TraceID = parseQuotedField(fields[17])
	entry.DomainName = parseQuotedField(fields[18])
	entry.ChosenCertArn = parseQuotedField(fields[19])
	
	if len(fields) > 20 {
		entry.MatchedRulePriority = fields[20]
	}
	if len(fields) > 21 {
		entry.RequestCreationTime, _ = parseTime(fields[21])
	}
	if len(fields) > 22 {
		entry.ActionsExecuted = parseQuotedField(fields[22])
	}
	if len(fields) > 23 {
		entry.RedirectURL = parseQuotedField(fields[23])
	}
	if len(fields) > 24 {
		entry.ErrorReason = parseQuotedField(fields[24])
	}
	if len(fields) > 25 {
		entry.TargetPortList = parseQuotedField(fields[25])
	}
	if len(fields) > 26 {
		entry.TargetStatusCodeList = parseQuotedField(fields[26])
	}
	if len(fields) > 27 {
		entry.Classification = parseQuotedField(fields[27])
	}
	if len(fields) > 28 {
		entry.ClassificationReason = parseQuotedField(fields[28])
	}
	if len(fields) > 29 {
		entry.ConnTraceID = parseQuotedField(fields[29])
	}
	
	return entry, nil
}

func (p *Parser) ExtractPath(requestURL string) (string, error) {
	if requestURL == "" || requestURL == "-" {
		return "", fmt.Errorf("empty or invalid request URL")
	}
	
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}
	
	return parsedURL.Path, nil
}

func parseTime(s string) (time.Time, error) {
	if s == "-" || s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	return time.Parse("2006-01-02T15:04:05.000000Z", s)
}

func parseFloat(s string) float64 {
	if s == "-1" || s == "" || s == "-" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func parseInt(s string) int {
	if s == "-1" || s == "" || s == "-" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func parseInt64(s string) int64 {
	if s == "-1" || s == "" || s == "-" {
		return 0
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

func parseQuotedField(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	if s == "-" {
		return ""
	}
	return s
}

func parseIP(s string) string {
	if colonIndex := strings.LastIndex(s, ":"); colonIndex != -1 {
		return s[:colonIndex]
	}
	return s
}

func parsePort(s string) string {
	if colonIndex := strings.LastIndex(s, ":"); colonIndex != -1 {
		return s[colonIndex+1:]
	}
	return "0"
}