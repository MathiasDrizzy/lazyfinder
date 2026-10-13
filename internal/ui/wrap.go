package ui

// wrap keeps a cursor inside a list of n rows going around: one step above the first row is the last one and one
// step below the last row is the first. An empty list has only row 0.
func wrap(i, n int) int {
	if n <= 0 {
		return 0
	}
	return (i%n + n) % n
}
