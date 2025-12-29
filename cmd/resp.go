package cmd

import (
	"errors"
	"fmt"
)

func readLength(data []byte) (int, int) {
	pos, len := 0, 0

	for pos = range data {
		b := data[pos]

		if !(b >= '0' && b <= '9') {
			return len, pos + 2
		}

		len = len*10 + int(b-'0')
	}

	return 0, 0
}

func readSimpleStr(data []byte) (string, int, error) {
	pos := 1

	for ; data[pos] != '\r'; pos++ {
	}

	return string(data[1:pos]), pos + 2, nil
}

func readBulkStr(data []byte) (string, int, error) {
	pos := 1

	len, extra := readLength(data)

	pos += extra

	return string(data[pos:(pos + len)]), pos + len + 2, nil
}

func readInt64(data []byte) (int64, int, error) {
	var value int64
	pos := 1

	for ; data[pos] != '\r'; pos++ {
		value = value*10 + int64(data[pos]-'0')
	}

	return value, pos + 2, nil
}

func readError(data []byte) (string, int, error) {
	return readSimpleStr(data)
}

func readArray(data []byte) (interface{}, int, error) {
	pos := 1
	count, delta := readLength(data)
	pos += delta

	var elems []interface{} = make([]interface{}, count)
	for i := range elems {
		elem, delta, err := DecodeOne(data[pos:])
		if err != nil {
			return nil, 0, err
		}
		elems[i] = elem
		pos += delta
	}

	return elems, pos, nil
}

func DecodeOne(data []byte) (interface{}, int, error) {
	if len(data) == 0 {
		return nil, 0, errors.New("no data")
	}
	switch data[0] {
	case '+':
		return readSimpleStr(data)
	case '-':
		return readError(data)
	case ':':
		return readInt64(data)
	case '$':
		return readBulkStr(data)
	case '*':
		return readArray(data)
	}
	return nil, 0, nil
}

func Decode(data []byte) (interface{}, error) {
	if len(data) == 0 {
		return nil, errors.New("No data found!")
	}

	value, _, err := DecodeOne(data)

	return value, err
}

func DecodeArrayString(data []byte) ([]string, error) {
	val, err := Decode(data)

	if err != nil {
		return nil, errors.New("Error while decoding")
	}

	ts, ok := val.([]interface{})

	if !ok {
		return nil, errors.New("interface not found")
	}
	tokens := make([]string, len(ts))

	for i := range tokens {
		tokens[i] = ts[i].(string)
	}

	return tokens, nil
}

func Encode(value interface{}, isSimpleString bool) []byte {
	switch v := value.(type) {
	case string:
		if isSimpleString {
			return []byte(fmt.Sprintf("+%s\r\n", v))
		}
		return []byte(fmt.Sprintf("$%d\r\n%s\r\n", len(v), v))
	case int64:
		return []byte(fmt.Sprintf(":%d\r\n", v))
	default:
		return RESP_NIL
	}
}
