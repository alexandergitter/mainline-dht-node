package main

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type scanner struct {
	input    string
	position int
}

func (c *scanner) next() (result byte, eof bool) {
	if c.position >= len(c.input) {
		return 0, true
	}

	result = c.input[c.position]
	c.position++

	return result, false
}

func (c *scanner) peek() (result byte, eof bool) {
	if c.position >= len(c.input) {
		return 0, true
	}

	return c.input[c.position], false
}

func (c *scanner) acceptSingle(accepted string) (result byte, found bool) {
	if b, eof := c.peek(); eof || !bytes.Contains([]byte(accepted), []byte{b}) {
		return 0, false
	} else {
		c.next()
		return b, true
	}
}

func (c *scanner) acceptMultiple(accepted string) string {
	var length = 0

	for _, found := c.acceptSingle(accepted); found; _, found = c.acceptSingle(accepted) {
		length++
	}

	return c.input[c.position-length : c.position]
}

func (c *scanner) acceptRun(length int) (string, error) {
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
	kind() bencodeType
	encode() string
	String() string
}

func (i bencodeInt) encode() string {
	return fmt.Sprintf("i%de", i)
}

func (i bencodeInt) String() string {
	return fmt.Sprintf("%d", i)
}

func (i bencodeInt) kind() bencodeType {
	return BencodeInteger
}

func (s bencodeString) encode() string {
	return fmt.Sprintf("%d:%s", len(s), s)
}

func (s bencodeString) String() string {
	return string(s)
}

func (s bencodeString) kind() bencodeType {
	return BencodeString
}

func (l bencodeList) encode() string {
	var buffer strings.Builder
	buffer.WriteString("l")

	for _, v := range l {
		buffer.WriteString(v.encode())
	}

	buffer.WriteString("e")
	return buffer.String()
}

func (l bencodeList) String() string {
	var buffer strings.Builder
	buffer.WriteString("[")
	for i, v := range l {
		if i > 0 {
			buffer.WriteString(", ")
		}
		buffer.WriteString(v.String())
	}
	buffer.WriteString("]")
	return buffer.String()
}

func (l bencodeList) kind() bencodeType {
	return BencodeList
}

func (d bencodeDict) encode() string {
	var buffer strings.Builder
	buffer.WriteString("d")

	var keys = make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		buffer.WriteString(bencodeString(k).encode())
		buffer.WriteString(d[k].encode())
	}

	buffer.WriteString("e")
	return buffer.String()
}

func (d bencodeDict) String() string {
	var buffer strings.Builder
	buffer.WriteString("{ ")
	var keys = make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			buffer.WriteString(", ")
		}
		buffer.WriteString(bencodeString(k).String())
		buffer.WriteString(": ")
		buffer.WriteString(d[k].String())
	}
	buffer.WriteString(" }")
	return buffer.String()
}

func (d bencodeDict) kind() bencodeType {
	return BencodeDict
}

func parseInteger(scanner *scanner) (int, error) {
	var sign = 1
	if c, found := scanner.acceptSingle("-+"); found && c == '-' {
		sign = -1
	}

	var chars = scanner.acceptMultiple("0123456789")

	if len(chars) == 0 {
		return 0, fmt.Errorf("unexpected integer with zero digits")
	}

	res, err := strconv.Atoi(chars)
	return res * sign, err
}

func decodeInteger(scanner *scanner) (bencodeInt, error) {
	if c, found := scanner.acceptSingle("i"); !found {
		return 0, fmt.Errorf("expected integer to start with \"i\", got %c instead", c)
	}

	var result, err = parseInteger(scanner)

	if c, found := scanner.acceptSingle("e"); !found {
		return 0, fmt.Errorf("unexpected end of integer: %c", c)
	}

	return bencodeInt(result), err
}

func decodeString(scanner *scanner) (bencodeString, error) {
	var length, err = parseInteger(scanner)
	if err != nil {
		return "", err
	} else if length < 0 {
		return "", fmt.Errorf("unexpected negative string length")
	}

	if c, found := scanner.acceptSingle(":"); !found {
		return "", fmt.Errorf("expected string to start with \":\", got %c instead", c)
	}

	result, err := scanner.acceptRun(length)
	return bencodeString(result), err
}

func decodeList(scanner *scanner) (bencodeList, error) {
	if c, found := scanner.acceptSingle("l"); !found {
		return nil, fmt.Errorf("expected list to start with \"l\", got %c instead", c)
	}

	var result = make(bencodeList, 0)

	for {
		if _, found := scanner.acceptSingle("e"); found {
			return result, nil
		} else {
			var value, err = decodeScannerBencodeValue(scanner)
			if err != nil {
				return nil, err
			}

			result = append(result, value)
		}
	}
}

func decodeDict(scanner *scanner) (bencodeDict, error) {
	if c, found := scanner.acceptSingle("d"); !found {
		return nil, fmt.Errorf("expected dict to start with \"d\", got %c instead", c)
	}

	var result = make(bencodeDict)

	for {
		if _, found := scanner.acceptSingle("e"); found {
			return result, nil
		} else {
			var key, err = decodeString(scanner)
			if err != nil {
				return nil, err
			}

			value, err := decodeScannerBencodeValue(scanner)
			if err != nil {
				return nil, err
			}

			result[string(key)] = value
		}
	}
}

func decodeBencodeDict(input string) (bencodeDict, error) {
	var scanner = &scanner{input: input}
	return decodeDict(scanner)
}

func decodeScannerBencodeValue(scanner *scanner) (bencodeValue, error) {
	var c, eof = scanner.peek()
	if eof {
		return nil, fmt.Errorf("unexpected end of input")
	}

	switch c {
	case 'i':
		return decodeInteger(scanner)
	case 'l':
		return decodeList(scanner)
	case 'd':
		return decodeDict(scanner)
	default:
		return decodeString(scanner)
	}
}

func decodeBencodeValue(input string) (bencodeValue, error) {
	var scanner = &scanner{input: input}
	return decodeScannerBencodeValue(scanner)
}
