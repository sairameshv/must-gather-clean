package obfuscator

import (
	"sync"

	"k8s.io/klog/v2"
)

type GenerateReplacement func() string

// ReplacementTracker is used to track and generate replacements used by obfuscators
type ReplacementTracker interface {
	// Initialize initializes the tracker with some existing replacements. It should be called only once and before
	// the first use of GetReplacement or AddReplacement
	Initialize(replacements map[string]string)

	// Report returns a mapping of strings which were replaced.
	Report() map[string]string

	// AddReplacement will add a replacement along with its original string to the report.
	// If there is an existing value that does not match the given replacement, it will exit with a non-zero status.
	AddReplacement(original string, replacement string)

	// GenerateIfAbsent returns the previously used replacement along with a true boolean conveying the entry is already present.
	// If the replacement is not present then it uses the GenerateReplacement function to generate a replacement. Returns a false boolean.
	// The "key" parameter must be used for lookup and the "generator" parameter to generate the replacement.
	GenerateIfAbsent(key string, generator GenerateReplacement) (string, bool)
}

type SimpleTracker struct {
	lock    sync.RWMutex
	mapping map[string]string
}

func (s *SimpleTracker) Report() map[string]string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	defensiveCopy := make(map[string]string)
	for k, v := range s.mapping {
		defensiveCopy[k] = v
	}
	return defensiveCopy
}

func (s *SimpleTracker) AddReplacement(original string, replacement string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if val, ok := s.mapping[original]; ok {
		if replacement != val {
			klog.Exitf("'%s' already has a value reported as '%s', tried to report '%s'", original, val, replacement)
		}
		return
	}
	s.mapping[original] = replacement
}

func (s *SimpleTracker) GenerateIfAbsent(key string, generator GenerateReplacement) (string, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	// returning the empty string if the replacement is already present
	// GenerateIfAbsent only should generate the replacement if the key is a new one
	// This helps in avoiding the addition of case-sensitive alternatives to the report
	if val, ok := s.mapping[key]; ok {
		return val, ok
	}
	if generator == nil {
		return "", false
	}
	r := generator()
	// commenting the next line as the Generate function should generate the alternative and return.
	// Addition of the entry should be ideally taken care by the AddReplacement method.
	//	s.mapping[key] = r
	return r, false
}

func (s *SimpleTracker) Initialize(replacements map[string]string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.mapping) > 0 {
		klog.Exitf("tracker was initialized more than once or after some replacements were already added.")
	}
	for k, v := range replacements {
		s.mapping[k] = v
	}
}

func NewSimpleTracker() ReplacementTracker {
	return &SimpleTracker{mapping: map[string]string{}}
}
