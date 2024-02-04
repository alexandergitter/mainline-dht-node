package main

import "testing"

func TestDecodeInteger(t *testing.T) {
	var input = "i123e"
	var result, err = decodeInteger(&scanner{input: input})
	if result != 123 || err != nil {
		t.Error("expected -123, got", result, "err:", err)
	}

	input = "i-123e"
	result, err = decodeInteger(&scanner{input: input})
	if result != -123 || err != nil {
		t.Error("expected -123, got", result, "err:", err)
	}

	input = "123"
	_, err = decodeInteger(&scanner{input: input})
	if err == nil {
		t.Error("Expected ", input, " to return an error")
	}

	input = "i123"
	_, err = decodeInteger(&scanner{input: input})
	if err == nil {
		t.Error("Expected ", input, " to return an error")
	}

	input = "ie"
	_, err = decodeInteger(&scanner{input: input})
	if err == nil {
		t.Error("Expected", input, "to return an error")
	}
}

func TestDecodeString(t *testing.T) {
	var input = "4:spam"
	var result, err = decodeString(&scanner{input: input})
	if result != "spam" || err != nil {
		t.Error("expected \"spam\", got", result, "err:", err)
	}

	input = "0:"
	result, err = decodeString(&scanner{input: input})
	if result != "" || err != nil {
		t.Error("expected \"\", got", result, "err:", err)
	}

	input = "spam"
	_, err = decodeString(&scanner{input: input})
	if err == nil {
		t.Error("Expected", input, "to return an error")
	}

	input = "4spam"
	_, err = decodeString(&scanner{input: input})
	if err == nil {
		t.Error("Expected", input, "to return an error")
	}
}

func TestDecodeList(t *testing.T) {
	var input = "l4:spami123ee"
	var result, err = decodeList(&scanner{input: input})
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
	_, err = decodeList(&scanner{input: input})
	if err == nil {
		t.Error("Expected", input, "to return an error")
	}

	input = "l4:spami123e"
	_, err = decodeList(&scanner{input: input})
	if err == nil {
		t.Error("Expected", input, "to return an error")
	}
}

func TestDecodeDict(t *testing.T) {
	var input = "d3:cow3:moo4:spam4:eggse"
	var result, err = decodeDict(&scanner{input: input})
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
	_, err = decodeDict(&scanner{input: input})
	if err == nil {
		t.Error("Expected", input, "to return an error")
	}
}

func TestDecodeValue(t *testing.T) {
	var input = "i123e"
	var result, err = decodeScannerBencodeValue(&scanner{input: input})
	if result.(bencodeInt) != 123 || err != nil {
		t.Error("expected 123, got", result, "err:", err)
	}

	input = "4:spam"
	result, err = decodeScannerBencodeValue(&scanner{input: input})
	if result.(bencodeString) != "spam" || err != nil {
		t.Error("expected \"spam\", got", result, "err:", err)
	}

	input = "l4:spami123ee"
	result, err = decodeScannerBencodeValue(&scanner{input: input})
	if err != nil {
		t.Error("Expected", input, "to return a list")
	}

	input = "d3:cow3:moo4:spam4:eggse"
	result, err = decodeScannerBencodeValue(&scanner{input: input})
	if err != nil {
		t.Error("Expected", input, "to return a dict")
	}
}

func TestEncodeValue(t *testing.T) {
	var input bencodeValue = bencodeInt(123)
	var result = input.encode()
	if result != "i123e" {
		t.Error("expected \"i123e\", got", result)
	}

	input = bencodeString("spam")
	result = input.encode()
	if result != "4:spam" {
		t.Error("expected \"4:spam\", got", result)
	}

	input = bencodeList{bencodeString("spam"), bencodeInt(123)}
	result = input.encode()
	if result != "l4:spami123ee" {
		t.Error("expected \"l4:spami123ee\", got", result)
	}

	input = bencodeDict{"dog": bencodeString("woof"), "cow": bencodeString("moo")}
	result = input.encode()
	if result != "d3:cow3:moo3:dog4:woofe" {
		t.Error("expected \"d3:cow3:moo3:dog4:woofe\", got", result)
	}
}
