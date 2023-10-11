package common

// KeyMatch checks if the value matches a key pattern.
// vIndex and pIndex are start positions of value and pattern strings to match.
// Mimics redis pattern matcher - i.e, glob like pattern matcher which
// matches '/' against wildcard.
// Supports '*' and '?' wildcards with '\' as the escape character.
// '*' matches any char sequence or none; '?' matches exactly one char.
// Character classes are not supported (redis supports it).
func KeyMatch(value, pattern string) bool {
	return keyMatch(value, 0, pattern, 0)
}

func keyMatch(value string, vIndex int, pattern string, pIndex int) bool {
	for pIndex < len(pattern) {
		switch pattern[pIndex] {
		case '*':
			// Skip successive *'s in the pattern
			pIndex++
			for pIndex < len(pattern) && pattern[pIndex] == '*' {
				pIndex++
			}
			// Pattern ends with *. Its a match always
			if pIndex == len(pattern) {
				return true
			}
			// Try to match remaining pattern with every value substring
			for ; vIndex < len(value); vIndex++ {
				if keyMatch(value, vIndex, pattern, pIndex) {
					return true
				}
			}
			// No match for remaining pattern
			return false

		case '?':
			// Accept any char.. there should be at least one
			if vIndex >= len(value) {
				return false
			}
			vIndex++
			pIndex++

		case '\\':
			// Do not treat \ as escape char if it is the last pattern char.
			// Redis commands behave this way.
			if pIndex+1 < len(pattern) {
				pIndex++
			}
			fallthrough

		default:
			if vIndex >= len(value) || pattern[pIndex] != value[vIndex] {
				return false
			}
			vIndex++
			pIndex++
		}
	}

	// All pattern chars have been compared.
	// It is a match if all value chars have been exhausted too.
	return (vIndex == len(value))
}
