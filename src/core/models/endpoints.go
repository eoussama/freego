package models

import "fmt"

type Endpoint struct {
	Fragments []interface{}
}

func (e Endpoint) Append(params ...interface{}) Endpoint {
	if len(params) > 0 {
		e.Fragments = append(e.Fragments, params...)
	}

	return e
}

func (e Endpoint) Build(params ...interface{}) ([]interface{}, error) {

	if len(params) == 0 {
		return e.Fragments, nil
	}

	var result []interface{}
	paramIndex := 0

	for _, fragment := range e.Fragments {
		if fragment == "?" {
			if paramIndex >= len(params) {
				return nil, fmt.Errorf("not enough parameters provided")
			}

			result = append(result, params[paramIndex])
			paramIndex++
		} else {
			result = append(result, fragment)
		}
	}

	if paramIndex < len(params) {
		return nil, fmt.Errorf("too many parameters provided")
	}

	return result, nil
}
