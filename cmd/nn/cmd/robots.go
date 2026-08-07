package cmd

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// robotsAllowed fetches robots.txt for the host of rawURL and reports whether
// the path is allowed for the given User-Agent. Fails open: if robots.txt
// cannot be fetched or parsed, the URL is considered allowed.
func robotsAllowed(rawURL, userAgent string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	robotsURL := parsed.Scheme + "://" + parsed.Host + "/robots.txt"

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest(http.MethodGet, robotsURL, nil)
	if err != nil {
		return true
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return true
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return true
	}

	return parseRobotsTxt(string(body), parsed.Path, userAgent)
}

// parseRobotsTxt checks whether path is allowed for ua per the robots.txt content.
// Implements the subset of the spec needed: User-agent and Disallow directives.
// Matching is case-insensitive on User-agent; the most specific matching group wins.
// Fails open: unrecognised syntax is ignored.
func parseRobotsTxt(content, path, ua string) bool {
	uaLower := strings.ToLower(ua)

	type group struct {
		agents    []string
		disallows []string
	}

	var groups []group
	var current *group

	for _, line := range strings.Split(content, "\n") {
		// Strip comments and whitespace.
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			if current != nil {
				groups = append(groups, *current)
				current = nil
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(parts[0]))
		val := strings.TrimSpace(parts[1])

		switch key {
		case "user-agent":
			if current == nil {
				current = &group{}
			}
			current.agents = append(current.agents, strings.ToLower(val))
		case "disallow":
			if current != nil {
				current.disallows = append(current.disallows, val)
			}
		}
	}
	if current != nil {
		groups = append(groups, *current)
	}

	// Find matching groups: prefer specific UA match over wildcard.
	var disallows []string
	wildcardDisallows := []string{}
	hasSpecific := false

	for _, g := range groups {
		isWildcard := false
		isMatch := false
		for _, agent := range g.agents {
			if agent == "*" {
				isWildcard = true
			} else if strings.Contains(uaLower, agent) {
				isMatch = true
			}
		}
		if isMatch {
			hasSpecific = true
			disallows = append(disallows, g.disallows...)
		} else if isWildcard {
			wildcardDisallows = append(wildcardDisallows, g.disallows...)
		}
	}
	if !hasSpecific {
		disallows = wildcardDisallows
	}

	for _, d := range disallows {
		if d == "" {
			continue
		}
		if strings.HasPrefix(path, d) {
			return false
		}
	}
	return true
}
