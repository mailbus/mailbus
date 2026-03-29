package core

import (
	"regexp"
	"strings"
	"time"
)

// Filter defines criteria for filtering messages
type Filter struct {
	SubjectPattern string         // Regular expression for subject matching
	FromPattern    string         // Pattern for sender matching (supports wildcards)
	ToPattern      string         // Pattern for recipient matching
	MinDate        *time.Time     // Minimum message date
	MaxDate        *time.Time     // Maximum message date
	Flags          []MessageFlag  // Required flags
	Tags           []string       // Required tags (from subject)
	UnreadOnly     bool           // Only match unread messages
	Custom         func(*Message) bool // Custom filter function
}

// FilterEngine provides message filtering capabilities
type FilterEngine struct {
	filters []*Filter
}

// NewFilterEngine creates a new filter engine
func NewFilterEngine() *FilterEngine {
	return &FilterEngine{
		filters: make([]*Filter, 0),
	}
}

// AddFilter adds a filter to the engine
func (e *FilterEngine) AddFilter(f *Filter) {
	e.filters = append(e.filters, f)
}

// Match checks if a message matches all filters in the engine
func (e *FilterEngine) Match(msg *Message) bool {
	for _, f := range e.filters {
		if !f.Match(msg) {
			return false
		}
	}
	return true
}

// Match checks if a message matches this filter
func (f *Filter) Match(msg *Message) bool {
	// Check unread filter
	if f.UnreadOnly && msg.IsSeen() {
		return false
	}

	// Check subject pattern
	if f.SubjectPattern != "" {
		matched, err := regexp.MatchString(f.SubjectPattern, msg.Subject)
		if err != nil || !matched {
			return false
		}
	}

	// Check from pattern
	if f.FromPattern != "" {
		if !matchPattern(f.FromPattern, msg.From) {
			return false
		}
	}

	// Check to pattern
	if f.ToPattern != "" {
		found := false
		for _, to := range msg.To {
			if matchPattern(f.ToPattern, to) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check date range
	if f.MinDate != nil && msg.Timestamp.Before(*f.MinDate) {
		return false
	}
	if f.MaxDate != nil && msg.Timestamp.After(*f.MaxDate) {
		return false
	}

	// Check flags
	if len(f.Flags) > 0 {
		for _, flag := range f.Flags {
			if !msg.hasFlag(flag) {
				return false
			}
		}
	}

	// Check tags
	if len(f.Tags) > 0 {
		msgTags := msg.GetTags()
		for _, requiredTag := range f.Tags {
			found := false
			for _, msgTag := range msgTags {
				if strings.EqualFold(msgTag, requiredTag) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	// Check custom filter
	if f.Custom != nil {
		if !f.Custom(msg) {
			return false
		}
	}

	return true
}

// matchPattern matches a string against a pattern with wildcard support
func matchPattern(pattern, str string) bool {
	// Convert wildcard pattern to regex
	regexPattern := strings.ReplaceAll(regexp.QuoteMeta(pattern), `\*`, `.*`)
	regexPattern = "^" + regexPattern + "$"

	matched, err := regexp.MatchString(regexPattern, str)
	if err != nil {
		return false
	}
	return matched
}

// NewFilter creates a new filter with the given criteria
func NewFilter() *Filter {
	return &Filter{
		Flags: make([]MessageFlag, 0),
		Tags:  make([]string, 0),
	}
}

// WithSubject sets the subject pattern
func (f *Filter) WithSubject(pattern string) *Filter {
	f.SubjectPattern = pattern
	return f
}

// WithFrom sets the from pattern
func (f *Filter) WithFrom(pattern string) *Filter {
	f.FromPattern = pattern
	return f
}

// WithTo sets the to pattern
func (f *Filter) WithTo(pattern string) *Filter {
	f.ToPattern = pattern
	return f
}

// WithUnreadOnly sets only unread messages
func (f *Filter) WithUnreadOnly() *Filter {
	f.UnreadOnly = true
	return f
}

// WithTags sets required tags
func (f *Filter) WithTags(tags ...string) *Filter {
	f.Tags = tags
	return f
}

// WithFlags sets required flags
func (f *Filter) WithFlags(flags ...MessageFlag) *Filter {
	f.Flags = flags
	return f
}

// WithDateRange sets the date range
func (f *Filter) WithDateRange(min, max time.Time) *Filter {
	f.MinDate = &min
	f.MaxDate = &max
	return f
}

// WithCustom sets a custom filter function
func (f *Filter) WithCustom(fn func(*Message) bool) *Filter {
	f.Custom = fn
	return f
}
