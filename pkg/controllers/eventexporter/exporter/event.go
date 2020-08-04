package exporter

import (
	"regexp"

	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/eventexporter/kube"
)

// Rule is for matching an event
type Rule struct {
	Labels      map[string]string
	Annotations map[string]string
	Message     string
	APIVersion  string
	Kind        string
	Namespace   string
	Reason      string
	Type        string
	MinCount    int32
	Component   string
	Host        string
	Receiver    string
}

// Route allows using rules to drop events or match events to specific receivers.
// It also allows using routes recursively for complex route building to fit
// most of the needs
type Route struct {
	Drop   []Rule
	Match  []Rule
	Routes []Route
}

// matchString is a method to clean the code. Error handling is omitted here because these
// rules are validated before use. According to regexp.MatchString, the only way it fails its
// that the pattern does not compile.
func matchString(pattern, s string) bool {
	matched, _ := regexp.MatchString(pattern, s)
	return matched
}

func (r *Route) ProcessEvent(ev *kube.EnhancedEvent, registry ReceiverRegistry) {
	// First determine whether we will drop the event: If any of the drop is matched, we break the loop
	for _, v := range r.Drop {
		if v.MatchesEvent(ev) {
			return
		}
	}

	// It has match rules, it should go to the matchers
	matchesAll := true
	for _, rule := range r.Match {
		if rule.MatchesEvent(ev) {
			if rule.Receiver != "" {
				registry.SendEvent(rule.Receiver, ev)
				// Send the event down the hole
			}
		} else {
			matchesAll = false
		}
	}

	// If all matches are satisfied, we can send them down to the rabbit hole
	if matchesAll {
		for _, subRoute := range r.Routes {
			subRoute.ProcessEvent(ev, registry)
		}
	}
}

// MatchesEvent compares the rule to an event and returns a boolean value to indicate
// whether the event is compatible with the rule. All fields are compared as regular expressions
// so the user must keep that in mind while writing rules.
func (r *Rule) MatchesEvent(ev *kube.EnhancedEvent) bool {
	// These rules are just basic comparison rules, if one of them fails, it means the event does not match the rule
	rules := [][2]string{
		{r.Message, ev.Message},
		{r.APIVersion, ev.InvolvedObject.APIVersion},
		{r.Kind, ev.InvolvedObject.Kind},
		{r.Namespace, ev.Namespace},
		{r.Reason, ev.Reason},
		{r.Type, ev.Type},
		{r.Component, ev.Source.Component},
		{r.Host, ev.Source.Host},
	}

	for _, v := range rules {
		rule := v[0]
		value := v[1]
		if rule != "" {
			matches := matchString(rule, value)
			if !matches {
				return false
			}
		}
	}

	// Labels are also mutually exclusive, they all need to be present
	if r.Labels != nil && len(r.Labels) > 0 {
		for k, v := range r.Labels {
			if val, ok := ev.InvolvedObject.Labels[k]; !ok {
				return false
			} else {
				matches := matchString(val, v)
				if !matches {
					return false
				}
			}
		}
	}

	// Annotations are also mutually exclusive, they all need to be present
	if r.Annotations != nil && len(r.Annotations) > 0 {
		for k, v := range r.Annotations {
			if val, ok := ev.InvolvedObject.Annotations[k]; !ok {
				return false
			} else {
				matches := matchString(v, val)
				if !matches {
					return false
				}
			}
		}
	}

	// If minCount is not given via a config, it's already 0 and the count is already 1 and this passes.
	if ev.Count >= r.MinCount {
		return true
	} else {
		return false
	}
}
