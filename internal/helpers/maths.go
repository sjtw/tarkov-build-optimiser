package helpers

func Pow(base int, exp int) int {
	result := 1
	for exp > 0 {
		result *= base
		exp--
	}
	return result
}
