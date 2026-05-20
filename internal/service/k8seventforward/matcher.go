package k8seventforward

import (
	"encoding/json"
	"strings"

	"yunshu/internal/model"
)

type RuleFilter struct {
	Namespaces []string
	Names      []string
	Reasons    []string
	Reverse    bool
}

func ParseRuleFilter(rule model.K8sEventForwardRule) RuleFilter {
	var rf RuleFilter
	_ = json.Unmarshal([]byte(rule.RuleNamespaces), &rf.Namespaces)
	_ = json.Unmarshal([]byte(rule.RuleNames), &rf.Names)
	_ = json.Unmarshal([]byte(rule.RuleReasons), &rf.Reasons)
	rf.Reverse = rule.RuleReverse
	return rf
}

func ParseClusterIDSet(clusterIDs string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, p := range strings.Split(clusterIDs, ",") {
		c := strings.TrimSpace(p)
		if c != "" {
			out[c] = struct{}{}
		}
	}
	return out
}

func (f RuleFilter) Match(ev *model.K8sForwardedEvent) bool {
	matched := f.matchPositive(ev)
	if f.Reverse {
		return !matched
	}
	return matched
}

func (f RuleFilter) matchPositive(ev *model.K8sForwardedEvent) bool {
	if len(f.Namespaces) > 0 {
		ok := false
		for _, ns := range f.Namespaces {
			if strings.TrimSpace(ns) == ev.Namespace {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(f.Names) > 0 {
		ok := false
		for _, n := range f.Names {
			if n != "" && strings.Contains(ev.Name, n) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(f.Reasons) > 0 {
		ok := false
		for _, r := range f.Reasons {
			if r == "" {
				continue
			}
			if strings.Contains(ev.Reason, r) || strings.Contains(ev.Message, r) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}
