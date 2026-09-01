package proof

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func LoadPolicy(path string) (Policy, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Policy{}, nil, err
	}
	policy, err := ParsePolicy(string(raw))
	if err != nil {
		return Policy{}, nil, fmt.Errorf("%s: %w", path, err)
	}
	return policy, raw, nil
}

func ParsePolicy(text string) (Policy, error) {
	policy := Policy{Rules: map[string]RuleSpec{}}
	scanner := bufio.NewScanner(strings.NewReader(text))
	seenHeader := false
	inside := false
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}
		tokens, err := tokenize(line)
		if err != nil {
			return Policy{}, fmt.Errorf("line %d: %w", lineNo, err)
		}
		if !seenHeader {
			if len(tokens) != 3 || tokens[0] != "contract" || tokens[2] != "{" {
				return Policy{}, fmt.Errorf("line %d: expected contract header", lineNo)
			}
			policy.Schema = tokens[1]
			seenHeader = true
			inside = true
			continue
		}
		if len(tokens) == 1 && tokens[0] == "}" {
			if !inside {
				return Policy{}, fmt.Errorf("line %d: unexpected closing brace", lineNo)
			}
			inside = false
			continue
		}
		if !inside {
			return Policy{}, fmt.Errorf("line %d: content outside contract", lineNo)
		}
		if err := parseRecord(&policy, tokens, lineNo); err != nil {
			return Policy{}, err
		}
	}
	if err := scanner.Err(); err != nil {
		return Policy{}, err
	}
	if !seenHeader || inside {
		return Policy{}, fmt.Errorf("incomplete .gooo contract")
	}
	if err := ValidatePolicy(policy); err != nil {
		return Policy{}, err
	}
	return policy, nil
}

func parseRecord(policy *Policy, tokens []string, lineNo int) error {
	if len(tokens) == 0 {
		return nil
	}
	bad := func(message string) error { return fmt.Errorf("line %d: %s", lineNo, message) }
	switch tokens[0] {
	case "authority":
		if len(tokens) != 2 {
			return bad("authority requires one value")
		}
		policy.Authority = tokens[1]
	case "precedence":
		if len(tokens) != 4 {
			return bad("precedence requires three values")
		}
		for _, token := range tokens[1:] {
			policy.Precedence = append(policy.Precedence, Status(token))
		}
	case "unknown_fields":
		policy.UnknownFields = append([]string(nil), tokens[1:]...)
	case "denominator":
		if len(tokens) < 3 {
			return bad("denominator is incomplete")
		}
		pairs, err := pairsAfter(tokens, 2)
		if err != nil {
			return bad(err.Error())
		}
		policy.Denominator.ID = tokens[1]
		policy.Denominator.Count, err = strconv.Atoi(pairs["count"])
		if err != nil {
			return bad("denominator count is not an integer")
		}
	case "inventory":
		pairs, err := pairsAfter(tokens, 1)
		if err != nil {
			return bad(err.Error())
		}
		policy.Inventory.RootReadme = pairs["root_readme"]
		policy.Inventory.FileClasses = splitCSV(pairs["file_classes"])
		policy.Inventory.PhysicalLines = pairs["physical_lines"]
	case "metrics":
		pairs, err := pairsAfter(tokens, 1)
		if err != nil {
			return bad(err.Error())
		}
		policy.Metrics.Stages = splitCSV(pairs["stages"])
		policy.Metrics.Fields = splitCSV(pairs["fields"])
		policy.Metrics.TestCounts = splitCSV(pairs["test_counts"])
		policy.Metrics.Improvement = pairs["improvement"]
		policy.Metrics.Missing = pairs["missing"]
	case "authority_rule":
		pairs, err := pairsAfter(tokens, 1)
		if err != nil {
			return bad(err.Error())
		}
		policy.AuthorityRule.OutputScope = pairs["output_scope"]
		policy.AuthorityRule.LocalVerification = pairs["local_verification_authority"]
		for key, target := range map[string]*int{
			"repository_writes": &policy.AuthorityRule.RepositoryWrites,
			"automatic_commit":  &policy.AuthorityRule.AutomaticCommit,
			"automatic_push":    &policy.AuthorityRule.AutomaticPush,
			"automatic_merge":   &policy.AuthorityRule.AutomaticMerge,
			"automatic_release": &policy.AuthorityRule.AutomaticRelease,
		} {
			value, parseErr := strconv.Atoi(pairs[key])
			if parseErr != nil {
				return bad("authority rule integer is malformed")
			}
			*target = value
		}
	case "generation":
		if len(tokens) < 3 {
			return bad("generation is incomplete")
		}
		pairs, err := pairsAfter(tokens, 3)
		if err != nil {
			return bad(err.Error())
		}
		policy.Generation.Language = tokens[2]
		policy.Generation.Role = pairs["role"]
		policy.Generation.OutputScope = pairs["output_scope"]
	case "activity":
		if len(tokens) < 3 {
			return bad("activity is incomplete")
		}
		pairs, err := pairsAfter(tokens, 1)
		if err != nil {
			return bad(err.Error())
		}
		policy.Activities = append(policy.Activities, ActivitySpec{ID: pairs["id"], Name: pairs["name"], Proof: pairs["proof"], Artifact: pairs["artifact"], Authority: pairs["authority"]})
	case "rule":
		pairs, err := pairsAfter(tokens, 1)
		if err != nil {
			return bad(err.Error())
		}
		rule := RuleSpec{ID: pairs["id"], Status: Status(pairs["status"]), Requires: pairs["requires"], UnknownClass: pairs["unknown_class"], NextOperation: pairs["next_operation"], BlockedBy: pairs["blocked_by"]}
		if rule.ID == "" {
			return bad("rule id is missing")
		}
		policy.Rules[rule.ID] = rule
	case "scenario":
		if len(tokens) < 3 {
			return bad("scenario is incomplete")
		}
		pairs, err := pairsAfter(tokens, 3)
		if err != nil {
			return bad(err.Error())
		}
		replay, err := strconv.ParseBool(pairs["replay"])
		if err != nil {
			return bad("scenario replay is not boolean")
		}
		policy.Scenarios = append(policy.Scenarios, ScenarioSpec{ID: tokens[2], Expected: Status(pairs["expected"]), Rule: pairs["rule"], Replay: replay})
	default:
		return bad("unknown .gooo record " + tokens[0])
	}
	return nil
}

func ValidatePolicy(policy Policy) error {
	if policy.Schema != "gooo/incremental-release-proof/v1" || policy.Authority != "metacode" {
		return fmt.Errorf(".gooo schema or authority is invalid")
	}
	if !sameStatuses(policy.Precedence, []Status{Refuted, Unknown, Closed}) {
		return fmt.Errorf("status precedence must be REFUTED > UNKNOWN > CLOSED")
	}
	if strings.Join(policy.UnknownFields, ",") != "stage,step,reason,unknown_class,next_operation,blocked_by" {
		return fmt.Errorf("UNKNOWN six-field tuple is incomplete")
	}
	if policy.Denominator.ID == "" || policy.Denominator.Count != 9 || len(policy.Scenarios) != 9 {
		return fmt.Errorf("denominator must declare exactly nine scenarios")
	}
	if policy.Inventory.RootReadme != "excluded" || strings.Join(policy.Inventory.FileClasses, ",") != "go,gooo,subdirectory,regular_file" || policy.Inventory.PhysicalLines != "required" {
		return fmt.Errorf("inventory boundary is incomplete")
	}
	if strings.Join(policy.Metrics.Stages, ",") != "compile,build,test,conformance,integration" || strings.Join(policy.Metrics.Fields, ",") != "wall_ms,peak_rss_kib" || strings.Join(policy.Metrics.TestCounts, ",") != "total,selected,executed,reused,failed,unknown" || policy.Metrics.Improvement == "" || policy.Metrics.Missing != "null+UNKNOWN" {
		return fmt.Errorf("measurement contract is incomplete")
	}
	if policy.AuthorityRule.RepositoryWrites != 0 || policy.AuthorityRule.AutomaticCommit != 0 || policy.AuthorityRule.AutomaticPush != 0 || policy.AuthorityRule.AutomaticMerge != 0 || policy.AuthorityRule.AutomaticRelease != 0 || policy.AuthorityRule.OutputScope != "CALLER_OWNED_TEMP_OUTPUT_ONLY" || policy.AuthorityRule.LocalVerification != "GITHUB_ACTIONS_ONLY" {
		return fmt.Errorf("authority boundary is invalid")
	}
	if policy.Generation.Language != "go" || policy.Generation.Role != "evaluator,generator,runtime" || policy.Generation.OutputScope != "CALLER_OWNED_TEMP_OUTPUT_ONLY" {
		return fmt.Errorf("generation boundary is invalid")
	}
	requiredActivities := []string{"load-parent-checkpoint", "verify-parent-immutability", "verify-parent-digest-links", "verify-changed-remote-evidence", "verify-chain-continuity", "preserve-historical-unknown", "refuse-cache-auto-close", "verify-exact-pair", "emit-six-field-unknown", "emit-refutation", "deterministic-replay", "emit-ci-evidence"}
	if len(policy.Activities) != len(requiredActivities) {
		return fmt.Errorf("activity denominator is incomplete")
	}
	seenActivities := map[string]bool{}
	for _, activity := range policy.Activities {
		if activity.ID == "" || activity.Name == "" || activity.Proof == "" || activity.Artifact == "" || activity.Authority != "READ_ONLY" || seenActivities[activity.ID] {
			return fmt.Errorf("activity mapping is incomplete")
		}
		seenActivities[activity.ID] = true
	}
	for _, id := range requiredActivities {
		if !seenActivities[id] {
			return fmt.Errorf("missing activity %q", id)
		}
	}
	requiredScenarios := map[string]Status{
		"parent-checkpoint-proven":       Closed,
		"changed-release-evidence-verified": Closed,
		"deterministic-replay":           Closed,
		"parent-checkpoint-missing":      Unknown,
		"parent-checkpoint-stale":        Unknown,
		"improvement-pair-missing":       Unknown,
		"parent-digest-contradiction":   Refuted,
		"changed-asset-digest-mismatch":  Refuted,
		"chain-discontinuity":            Refuted,
	}
	seenScenarios := map[string]bool{}
	for _, scenario := range policy.Scenarios {
		wanted, ok := requiredScenarios[scenario.ID]
		if !ok || seenScenarios[scenario.ID] || scenario.Expected != wanted || policy.Rules[scenario.Rule].ID == "" {
			return fmt.Errorf("scenario %q is invalid", scenario.ID)
		}
		seenScenarios[scenario.ID] = true
	}
	for id := range requiredScenarios {
		if !seenScenarios[id] {
			return fmt.Errorf("missing scenario %q", id)
		}
	}
	return nil
}

func sameStatuses(actual, expected []Status) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i := range actual {
		if actual[i] != expected[i] {
			return false
		}
	}
	return true
}

func tokenize(line string) ([]string, error) {
	var tokens []string
	for index := 0; index < len(line); {
		for index < len(line) && (line[index] == ' ' || line[index] == '\t') {
			index++
		}
		if index == len(line) {
			break
		}
		if line[index] != '"' {
			start := index
			for index < len(line) && line[index] != ' ' && line[index] != '\t' {
				index++
			}
			tokens = append(tokens, line[start:index])
			continue
		}
		start := index
		index++
		escaped := false
		closed := false
		for index < len(line) {
			if escaped {
				escaped = false
				index++
				continue
			}
			if line[index] == '\\' {
				escaped = true
				index++
				continue
			}
			if line[index] == '"' {
				index++
				closed = true
				break
			}
			index++
		}
		if !closed {
			return nil, fmt.Errorf("unterminated quoted value")
		}
		value, err := strconv.Unquote(line[start:index])
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, value)
	}
	return tokens, nil
}

func pairsAfter(tokens []string, start int) (map[string]string, error) {
	if start > len(tokens) || (len(tokens)-start)%2 != 0 {
		return nil, fmt.Errorf("key/value pairs are malformed")
	}
	pairs := map[string]string{}
	for index := start; index < len(tokens); index += 2 {
		pairs[tokens[index]] = tokens[index+1]
	}
	return pairs, nil
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func stripComment(line string) string {
	if index := strings.Index(line, "//"); index >= 0 {
		return line[:index]
	}
	return line
}
