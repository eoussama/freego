package helpers

func IntToInterfaceSlice(slice []int) []interface{} {
	interfaceSlice := make([]interface{}, len(slice))

	for i, v := range slice {
		interfaceSlice[i] = v
	}

	return interfaceSlice
}

func StringToInterfaceSlice(slice []string) []interface{} {
	interfaceSlice := make([]interface{}, len(slice))

	for i, v := range slice {
		interfaceSlice[i] = v
	}

	return interfaceSlice
}
