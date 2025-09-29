package main

// normalizeLogLine returns the parsed entry and normalized route when the log line matches a rule.
func normalizeLogLine(line string, rules *pathRules) (*albLogEntry, string, bool) {
	if rules == nil || !rules.enabled {
		return nil, "", false
	}

	entry, err := parseALBLogLine(line)
	if err != nil {
		return nil, "", false
	}

	route, matched := rules.normalize(*entry)
	if !matched {
		return nil, "", false
	}

	return entry, route, true
}
