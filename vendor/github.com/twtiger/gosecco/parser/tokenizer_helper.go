package parser

type tokenData struct {
	t  token
	td []byte
}

func tokenize(data string, tokenError func(int, int, []byte) error) ([]tokenData, error) {
	result := []tokenData{}

	err := tokenizeRaw([]byte(data), func(t token, td []byte) {
		result = append(result, tokenData{t, td})
	}, tokenError)

	if err != nil {
		return nil, err
	}

	return result, nil
}
