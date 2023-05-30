package result

// Result contains the result tuple of an operation.
type Result[T comparable] struct {
	// val is the value
	val T
	// err is the error
	err error
}

// NewResult constructs a new result container.
func NewResult[T comparable](val T, err error) *Result[T] {
	return &Result[T]{val: val, err: err}
}

// Compare compares two Result objects for equality.
func (r *Result[T]) Compare(ot *Result[T]) bool {
	return r.val == ot.val && r.err == ot.err
}

// GetValue returns the result and error value.
func (r *Result[T]) GetValue() (val T, err error) {
	return r.val, r.err
}
