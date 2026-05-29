package utils

// Ternary 通用三元运算符函数
func If[T any](condition bool, trueVal, falseVal T) T {
	if condition {
		return trueVal
	}
	return falseVal
}
