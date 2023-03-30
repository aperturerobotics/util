package gotargets

import "sort"

//go:generate go run -v -tags=generate generate.go

// GetOsArchValues returns the list of GOARCH values for each GOOS.
func GetOsArchValues() map[string][]string {
	m := make(map[string][]string, len(KnownGoDists))
	for _, kd := range KnownGoDists {
		m[kd.GOOS] = append(m[kd.GOOS], kd.GOARCH)
	}
	for k := range m {
		sort.Strings(m[k])
	}
	return m
}
