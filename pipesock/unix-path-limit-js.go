//go:build js || wasip1 || plan9

package pipesock

func validateUnixSocketPath(string) error {
	return nil
}
