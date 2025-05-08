package slices

// Filter 通用过滤函数，接受任何类型的切片和谓词函数
func Filter[T any](slice []T, predicate func(T) bool) []T {
	var result []T
	for _, v := range slice {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}
