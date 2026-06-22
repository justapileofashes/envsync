// Package schema implements optional .env.schema validation and typo-squashing.
//
// A team lead commits a .env.schema file describing the expected variables; on
// push the client validates the local file against it and blocks pushes that are
// missing required keys, violate a prefix/regex constraint, or appear to contain
// a typo of a required key (e.g. DATABSE_URL for DATABASE_URL).
//
// Schema grammar (one rule per line, # for comments):
//
//	KEY [required|optional] [prefix=<str>] [regex=<re>]
//
// Examples:
//
//	DATABASE_URL required
//	STRIPE_KEY   required prefix=sk_
//	PORT         optional regex=^[0-9]+$
package schema

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// DefaultFile is the conventional schema filename.
const DefaultFile = ".env.schema"

// Rule describes the constraints on a single variable.
type Rule struct {
	Key      string
	Required bool
	Prefix   string
	Regex    *regexp.Regexp
	rawRegex string
}

// Schema is an ordered set of rules.
type Schema struct {
	Rules []Rule
}

// Violation is a single validation failure with an optional suggestion.
type Violation struct {
	Key        string
	Message    string
	Suggestion string // e.g. a likely-misspelled key the dev meant
}

func (v Violation) String() string {
	if v.Suggestion != "" {
		return fmt.Sprintf("%s: %s (did you mean %q?)", v.Key, v.Message, v.Suggestion)
	}
	return fmt.Sprintf("%s: %s", v.Key, v.Message)
}

// Exists reports whether a schema file is present at path.
func Exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// Load parses a schema file.
func Load(path string) (*Schema, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("schema: cannot open %s: %w", path, err)
	}
	defer f.Close()

	var s Schema
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rule, err := parseRule(line)
		if err != nil {
			return nil, fmt.Errorf("schema: %s line %d: %w", path, lineNo, err)
		}
		s.Rules = append(s.Rules, rule)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("schema: error reading %s: %w", path, err)
	}
	return &s, nil
}

func parseRule(line string) (Rule, error) {
	fields := strings.Fields(line)
	r := Rule{Key: fields[0], Required: true} // default required
	for _, tok := range fields[1:] {
		switch {
		case tok == "required":
			r.Required = true
		case tok == "optional":
			r.Required = false
		case strings.HasPrefix(tok, "prefix="):
			r.Prefix = strings.TrimPrefix(tok, "prefix=")
		case strings.HasPrefix(tok, "regex="):
			r.rawRegex = strings.TrimPrefix(tok, "regex=")
			re, err := regexp.Compile(r.rawRegex)
			if err != nil {
				return r, fmt.Errorf("invalid regex %q: %w", r.rawRegex, err)
			}
			r.Regex = re
		default:
			return r, fmt.Errorf("unknown token %q", tok)
		}
	}
	return r, nil
}

// Validate checks values against the schema and returns all violations. It also
// performs typo-squashing: for a missing required key, if a present (unexpected)
// key is within edit distance 2, it is suggested as a likely misspelling.
func (s *Schema) Validate(values map[string]string) []Violation {
	var violations []Violation

	// Set of keys named by the schema, to find "unexpected" present keys.
	schemaKeys := make(map[string]struct{}, len(s.Rules))
	for _, r := range s.Rules {
		schemaKeys[r.Key] = struct{}{}
	}

	for _, r := range s.Rules {
		val, present := values[r.Key]
		if !present {
			if r.Required {
				v := Violation{Key: r.Key, Message: "required variable is missing"}
				if sugg := nearestTypo(r.Key, values, schemaKeys); sugg != "" {
					v.Suggestion = sugg
				}
				violations = append(violations, v)
			}
			continue
		}
		if r.Prefix != "" && !strings.HasPrefix(val, r.Prefix) {
			violations = append(violations, Violation{
				Key:     r.Key,
				Message: fmt.Sprintf("value must start with %q", r.Prefix),
			})
		}
		if r.Regex != nil && !r.Regex.MatchString(val) {
			violations = append(violations, Violation{
				Key:     r.Key,
				Message: fmt.Sprintf("value does not match /%s/", r.rawRegex),
			})
		}
	}
	return violations
}

// nearestTypo returns a present key that is a likely misspelling of want: it is
// not itself named by the schema and is within edit distance 2.
func nearestTypo(want string, values map[string]string, schemaKeys map[string]struct{}) string {
	best := ""
	bestDist := 3 // strictly less than 3 (i.e. <=2) to qualify
	for k := range values {
		if _, named := schemaKeys[k]; named {
			continue
		}
		d := levenshtein(strings.ToUpper(k), strings.ToUpper(want))
		if d < bestDist {
			bestDist = d
			best = k
		}
	}
	return best
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
