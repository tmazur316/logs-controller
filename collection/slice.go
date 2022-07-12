package collection

func Remove[T comparable](slice []T, elem T) []T {
	for i, value := range slice {
		if value == elem {
			return append(slice[:i], slice[i+1:]...)
		}
	}

	return slice
}

func ContainsAllValues[K, V comparable](collection map[K]V, values map[K]V) bool {
	for key, value := range values {
		if val, exists := collection[key]; !exists || val != value {
			return false
		}
	}

	return true
}
