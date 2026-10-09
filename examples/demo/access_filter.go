package demo

import "encoding/json"

type AccessFilterFunc func(*RequestInfo)

type AccessFilterRule struct {
	filters  []accessFilter
	excludes []string
}

func NewAccessFilterRule() *AccessFilterRule {
	return &AccessFilterRule{}
}

func (rule *AccessFilterRule) Handle(
	pattern string,
	filter AccessFilterFunc,
) {
	rule.filters = append(rule.filters, accessFilter{
		pattern: pattern,
		apply:   filter,
	})
}

func (rule *AccessFilterRule) Exclude(patterns ...string) {
	rule.excludes = append(rule.excludes, patterns...)
}

func (rule *AccessFilterRule) Apply(path string, info *RequestInfo) {
	for _, filter := range rule.filters {
		if matchRoutePattern(filter.pattern, path) {
			filter.apply(info)
		}
	}
}

func (rule *AccessFilterRule) Excluded(path string) bool {
	return matchesAnyRoute(rule.excludes, path)
}

func RedactRequestFields(fields ...string) AccessFilterFunc {
	sensitiveFields := makeSensitiveFields(fields)
	return func(info *RequestInfo) {
		if info.Req == nilLogValue {
			return
		}
		var value any
		if err := json.Unmarshal([]byte(info.Req), &value); err != nil {
			return
		}
		redactLogValue(value, sensitiveFields)
		info.Req = logJSON(value)
	}
}

type accessFilter struct {
	pattern string
	apply   AccessFilterFunc
}
