package main

// normalizeLogLine returns the parsed entry and normalized route when the log line matches a rule.
func (p *MetricsProcessor) normalizeLogLine(line string) (*albLogEntry, string, bool) {
	if p.rules == nil || !p.rules.enabled {
		return nil, "", false
	}

	entry, err := parseALBLogLine(line)
	if err != nil {
		return nil, "", false
	}

	route, matched := p.rules.normalize(*entry)
	if !matched {
		return nil, "", false
	}

	return entry, route, true
}
