package main

import "testing"

func TestDecodeInteger(t *testing.T) {
	var input = "i123e"
	var result, err = decodeInteger(&cursor{input: input})
	if result != 123 || err != nil {
		t.Error("expected -123, got", result, "err:", err)
	}

	input = "i-123e"
	result, err = decodeInteger(&cursor{input: input})
	if result != -123 || err != nil {
		t.Error("expected -123, got", result, "err:", err)
	}

	input = "123"
	_, err = decodeInteger(&cursor{input: input})
	if err == nil {
		t.Error("Expected ", input, " to return an error")
	}

	input = "i123"
	_, err = decodeInteger(&cursor{input: input})
	if err == nil {
		t.Error("Expected ", input, " to return an error")
	}

	input = "ie"
	_, err = decodeInteger(&cursor{input: input})
	if err == nil {
		t.Error("Expected", input, "to return an error")
	}
}

func TestDecodeString(t *testing.T) {
	var input = "4:spam"
	var result, err = decodeString(&cursor{input: input})
	if result != "spam" || err != nil {
		t.Error("expected \"spam\", got", result, "err:", err)
	}

	input = "0:"
	result, err = decodeString(&cursor{input: input})
	if result != "" || err != nil {
		t.Error("expected \"\", got", result, "err:", err)
	}

	input = "spam"
	_, err = decodeString(&cursor{input: input})
	if err == nil {
		t.Error("Expected", input, "to return an error")
	}

	input = "4spam"
	_, err = decodeString(&cursor{input: input})
	if err == nil {
		t.Error("Expected", input, "to return an error")
	}
}

func TestDecodeList(t *testing.T) {
	var input = "l4:spami123ee"
	var result, err = decodeList(&cursor{input: input})
	if err != nil {
		t.Error("Expected", input, "to return a list")
	}

	if len(result) != 2 {
		t.Error("Expected", input, "to return a list of length 2")
	}

	if (result)[0].(bencodeString) != "spam" {
		t.Error("Expected", input, "to return a list with \"spam\" as the first element")
	}

	if (result)[1].(bencodeInt) != 123 {
		t.Error("Expected", input, "to return a list with 123 as the second element")
	}

	input = "l4:spam"
	_, err = decodeList(&cursor{input: input})
	if err == nil {
		t.Error("Expected", input, "to return an error")
	}

	input = "l4:spami123e"
	_, err = decodeList(&cursor{input: input})
	if err == nil {
		t.Error("Expected", input, "to return an error")
	}
}

func TestDecodeDict(t *testing.T) {
	var input = "d3:cow3:moo4:spam4:eggse"
	var result, err = decodeDict(&cursor{input: input})
	if err != nil {
		t.Error("Expected", input, "to return a dict")
	}

	if len(result) != 2 {
		t.Error("Expected", input, "to return a dict of length 2")
	}

	if (result)["cow"].(bencodeString) != "moo" {
		t.Error("Expected", input, "to return a dict with \"cow\" as the first key")
	}

	if (result)["spam"].(bencodeString) != "eggs" {
		t.Error("Expected", input, "to return a dict with \"spam\" as the second key")
	}

	input = "d3:cow3:moo4:spam4:eggs"
	_, err = decodeDict(&cursor{input: input})
	if err == nil {
		t.Error("Expected", input, "to return an error")
	}
}

func TestDecodeValue(t *testing.T) {
	var input = "i123e"
	var result, err = decodeValue(&cursor{input: input})
	if result.(bencodeInt) != 123 || err != nil {
		t.Error("expected 123, got", result, "err:", err)
	}

	input = "4:spam"
	result, err = decodeValue(&cursor{input: input})
	if result.(bencodeString) != "spam" || err != nil {
		t.Error("expected \"spam\", got", result, "err:", err)
	}

	input = "l4:spami123ee"
	result, err = decodeValue(&cursor{input: input})
	if err != nil {
		t.Error("Expected", input, "to return a list")
	}

	input = "d3:cow3:moo4:spam4:eggse"
	result, err = decodeValue(&cursor{input: input})
	if err != nil {
		t.Error("Expected", input, "to return a dict")
	}
}

func TestEncodeValue(t *testing.T) {
	var input bencodeValue = bencodeInt(123)
	var result, err = encodeValue(input)
	if result != "i123e" || err != nil {
		t.Error("expected \"i123e\", got", result, err)
	}

	input = bencodeString("spam")
	result, err = encodeValue(input)
	if result != "4:spam" || err != nil {
		t.Error("expected \"4:spam\", got", result, err)
	}

	input = bencodeList{bencodeString("spam"), bencodeInt(123)}
	result, err = encodeValue(input)
	if result != "l4:spami123ee" || err != nil {
		t.Error("expected \"l4:spami123ee\", got", result, err)
	}

	input = bencodeDict{"dog": bencodeString("woof"), "cow": bencodeString("moo")}
	result, err = encodeValue(input)
	if result != "d3:cow3:moo3:dog4:woofe" || err != nil {
		t.Error("expected \"d3:cow3:moo3:dog4:woofe\", got", result, err)
	}
}
