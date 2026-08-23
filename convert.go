package tool

func ConvertArrayToMap[T any, K comparable](arr []T, keyFunc func(T) K) map[K]T {
	result := make(map[K]T)
	if arr == nil {
		return result
	}
	for i := range arr {
		key := keyFunc(arr[i])
		result[key] = arr[i]
	}
	return result
}
func ConvertArrayToMapArray[T any, K comparable](arr []T, keyFunc func(T) K) map[K][]T {
	result := make(map[K][]T)
	if arr == nil {
		return result
	}
	for i := range arr {
		key := keyFunc(arr[i])
		result[key] = append(result[key], arr[i])
	}
	return result
}
func ConvertMapToArray[T any, V comparable](arr []T, keyFunc func(T) V) []V {
	result := make([]V, 0)
	if arr == nil {
		return result
	}
	is := make(map[V]struct{})
	for i := range arr {
		key := keyFunc(arr[i])
		if _, ok := is[key]; ok {
			continue
		}
		is[key] = struct{}{}
		result = append(result, key)
	}
	return result
}
