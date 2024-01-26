package main

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type cursor struct {
	input    string
	position int
}

func (c *cursor) next() (result byte, eof bool) {
	if c.position >= len(c.input) {
		return 0, true
	}

	result = c.input[c.position]
	c.position++

	return result, false
}

func (c *cursor) peek() (result byte, eof bool) {
	if c.position >= len(c.input) {
		return 0, true
	}

	return c.input[c.position], false
}

func (c *cursor) acceptSingle(accepted string) (result byte, found bool) {
	if b, eof := c.peek(); eof || !bytes.Contains([]byte(accepted), []byte{b}) {
		return 0, false
	} else {
		c.next()
		return b, true
	}
}

func (c *cursor) acceptMultiple(accepted string) string {
	var length = 0

	for _, found := c.acceptSingle(accepted); found; _, found = c.acceptSingle(accepted) {
		length++
	}

	return c.input[c.position-length : c.position]
}

func (c *cursor) acceptRun(length int) (string, error) {
	if c.position+length > len(c.input) {
		return "", fmt.Errorf("unexpected end of input")
	}

	var result = c.input[c.position : c.position+length]
	c.position += length

	return result, nil
}

type bencodeType int
type bencodeInt int
type bencodeString string
type bencodeList []bencodeValue
type bencodeDict map[string]bencodeValue

const (
	BencodeInteger bencodeType = iota
	BencodeString
	BencodeList
	BencodeDict
)

type bencodeValue interface {
	Type() bencodeType
}

func (i bencodeInt) Type() bencodeType {
	return BencodeInteger
}

func (s bencodeString) Type() bencodeType {
	return BencodeString
}

func (l bencodeList) Type() bencodeType {
	return BencodeList
}

func (d bencodeDict) Type() bencodeType {
	return BencodeDict
}

func parseInteger(cursor *cursor) (int, error) {
	var sign = 1
	if c, found := cursor.acceptSingle("-+"); found && c == '-' {
		sign = -1
	}

	var chars = cursor.acceptMultiple("0123456789")

	if len(chars) == 0 {
		return 0, fmt.Errorf("unexpected integer with zero digits")
	}

	res, err := strconv.Atoi(chars)
	return res * sign, err
}

func decodeInteger(cursor *cursor) (bencodeInt, error) {
	if c, found := cursor.acceptSingle("i"); !found {
		return 0, fmt.Errorf("expected integer to start with \"i\", got %c instead", c)
	}

	var result, err = parseInteger(cursor)

	if c, found := cursor.acceptSingle("e"); !found {
		return 0, fmt.Errorf("unexpected end of integer: %c", c)
	}

	return bencodeInt(result), err
}

func decodeString(cursor *cursor) (bencodeString, error) {
	var length, err = parseInteger(cursor)
	if err != nil {
		return "", err
	} else if length < 0 {
		return "", fmt.Errorf("unexpected negative string length")
	}

	if c, found := cursor.acceptSingle(":"); !found {
		return "", fmt.Errorf("expected string to start with \":\", got %c instead", c)
	}

	result, err := cursor.acceptRun(length)
	return bencodeString(result), err
}

func decodeList(cursor *cursor) (bencodeList, error) {
	if c, found := cursor.acceptSingle("l"); !found {
		return nil, fmt.Errorf("expected list to start with \"l\", got %c instead", c)
	}

	var result = make(bencodeList, 0)

	for {
		if _, found := cursor.acceptSingle("e"); found {
			return result, nil
		} else {
			var value, err = decodeValue(cursor)
			if err != nil {
				return nil, err
			}

			result = append(result, value)
		}
	}
}

func decodeDict(cursor *cursor) (bencodeDict, error) {
	if c, found := cursor.acceptSingle("d"); !found {
		return nil, fmt.Errorf("expected dict to start with \"d\", got %c instead", c)
	}

	var result = make(bencodeDict)

	for {
		if _, found := cursor.acceptSingle("e"); found {
			return result, nil
		} else {
			var key, err = decodeString(cursor)
			if err != nil {
				return nil, err
			}

			value, err := decodeValue(cursor)
			if err != nil {
				return nil, err
			}

			result[string(key)] = value
		}
	}
}

func decodeValue(cursor *cursor) (bencodeValue, error) {
	var c, eof = cursor.peek()
	if eof {
		return nil, fmt.Errorf("unexpected end of input")
	}

	switch c {
	case 'i':
		return decodeInteger(cursor)
	case 'l':
		return decodeList(cursor)
	case 'd':
		return decodeDict(cursor)
	default:
		return decodeString(cursor)
	}
}

func decodeBencode(input string) (bencodeValue, error) {
	var cursor = &cursor{input: input}
	return decodeValue(cursor)
}

func encodeValue(value bencodeValue) (string, error) {
	switch value.Type() {
	case BencodeInteger:
		return fmt.Sprintf("i%de", value.(bencodeInt)), nil
	case BencodeString:
		return fmt.Sprintf("%d:%s", len(value.(bencodeString)), value.(bencodeString)), nil
	case BencodeList:
		var buffer strings.Builder
		buffer.WriteString("l")

		for _, v := range value.(bencodeList) {
			s, err := encodeValue(v)
			if err != nil {
				return "", err
			}

			buffer.WriteString(s)
		}

		buffer.WriteString("e")
		return buffer.String(), nil
	case BencodeDict:
		var buffer strings.Builder
		buffer.WriteString("d")

		var keys = make([]string, 0, len(value.(bencodeDict)))
		for k := range value.(bencodeDict) {
			keys = append(keys, k)
		}

		sort.Strings(keys)

		for _, k := range keys {
			s, err := encodeValue(bencodeString(k))
			if err != nil {
				return "", err
			}

			buffer.WriteString(s)

			s, err = encodeValue(value.(bencodeDict)[k])
			if err != nil {
				return "", err
			}

			buffer.WriteString(s)
		}

		buffer.WriteString("e")
		return buffer.String(), nil
	default:
		return "", fmt.Errorf("unknown bencode type")
	}
}

func printBencodeValue(value bencodeValue) {
	switch value.Type() {
	case BencodeInteger:
		fmt.Printf("%d", value.(bencodeInt))
	case BencodeString:
		fmt.Printf("%s", value.(bencodeString))
	case BencodeList:
		fmt.Printf("[")
		for i, v := range value.(bencodeList) {
			if i > 0 {
				fmt.Printf(", ")
			}
			printBencodeValue(v)
		}
		fmt.Printf("]")
	case BencodeDict:
		fmt.Printf("{ ")
		var keys = make([]string, 0, len(value.(bencodeDict)))
		for k := range value.(bencodeDict) {
			keys = append(keys, k)
		}

		for i, k := range keys {
			if i > 0 {
				fmt.Printf(", ")
			}
			printBencodeValue(bencodeString(k))
			fmt.Printf(": ")
			printBencodeValue(value.(bencodeDict)[k])
		}
		fmt.Printf(" }")
	default:
		fmt.Printf("unknown bencode type")
	}
}
