package main

// shouldPrintLogLine determines whether the given ALB log line matches any path rule.
func shouldPrintLogLine(line string, rules *pathRules) bool {
	if rules == nil || !rules.enabled {
		return false
	}

	entry, err := parseALBLogLine(line)
	if err != nil {
		return false
	}

	_, matched := rules.normalize(*entry)
	return matched
}
