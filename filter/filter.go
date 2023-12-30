package filter

import (
	"regexp"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

// Validate validates the string filter.
func (f *StringFilter) Validate() error {
	if reSrc := f.GetRe(); reSrc != "" {
		if _, err := regexp.Compile(reSrc); err != nil {
			return errors.Wrap(err, "re")
		}
	}
	return nil
}

// CheckMatch checks if the given value matches the string filter.
func (f *StringFilter) CheckMatch(value string) bool {
	if f == nil {
		return true
	}
	if f.GetEmpty() && value != "" {
		return false
	}
	if f.GetNotEmpty() && value == "" {
		return false
	}
	if val := f.GetValue(); val != "" && value != val {
		return false
	}
	if matchValues := f.GetValues(); len(matchValues) != 0 && !slices.Contains(matchValues, value) {
		return false
	}
	if reSrc := f.GetRe(); reSrc != "" {
		rgx, err := regexp.Compile(reSrc)
		if err != nil {
			// checked in Validate but treat it as a fail
			return false
		}
		if !rgx.MatchString(value) {
			return false
		}
	}
	if prefixSrc := f.GetHasPrefix(); prefixSrc != "" {
		if !strings.HasPrefix(value, prefixSrc) {
			return false
		}
	}
	if suffixSrc := f.GetHasSuffix(); suffixSrc != "" {
		if !strings.HasSuffix(value, suffixSrc) {
			return false
		}
	}
	if containsSrc := f.GetContains(); containsSrc != "" {
		if !strings.Contains(value, containsSrc) {
			return false
		}
	}

	return true
}
