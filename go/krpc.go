package main

import "fmt"

type krpcMessageType = string

const (
	KrpcTypeQuery krpcMessageType = "q"
	KrpcTypeReply krpcMessageType = "r"
	KrpcTypeError krpcMessageType = "e"
)

type krpcErrorType = int

const (
	KrpcErrorGeneric       krpcErrorType = 201
	KrpcErrorServer        krpcErrorType = 202
	KrpcErrorProtocol      krpcErrorType = 203
	KrpcErrorUnknownMethod krpcErrorType = 204
)

type krpcQuery struct {
	transactionId string
	methodName    string
	arguments     bencodeDict
}
type krpcResponse struct {
	transactionId string
	response      bencodeDict
}

type krpcError struct {
	transactionId string
	kind          krpcErrorType
	message       string
}

type krpcMessage interface {
	encode() string
}

func (qry krpcQuery) encode() string {
	var ben = bencodeDict{
		"t": bencodeString(qry.transactionId),
		"y": bencodeString(KrpcTypeQuery),
		"q": bencodeString(qry.methodName),
		"a": qry.arguments,
	}

	return ben.encode()
}

func (res krpcResponse) encode() string {
	var ben = bencodeDict{
		"t": bencodeString(res.transactionId),
		"y": bencodeString(KrpcTypeReply),
		"r": res.response,
	}

	return ben.encode()
}

func (err krpcError) encode() string {
	var ben = bencodeDict{
		"t": bencodeString(err.transactionId),
		"y": bencodeString(KrpcTypeError),
		"e": bencodeList{bencodeInt(err.kind), bencodeString(err.message)},
	}

	return ben.encode()
}

func decodeKrpcMessage(data bencodeDict) (krpcMessage, error) {
	var t = data["t"]
	if t == nil || t.kind() != BencodeString {
		return nil, fmt.Errorf("KRCP transaction ID is not a string")
	}

	var y = data["y"]
	if y == nil || y.kind() != BencodeString {
		return nil, fmt.Errorf("KRPC Message type is not a string")
	}

	switch string(y.(bencodeString)) {
	case KrpcTypeQuery:
		var q = data["q"]
		if q == nil || q.kind() != BencodeString {
			return nil, fmt.Errorf("Query method is not a string")
		}

		var a = data["a"]
		if a == nil || a.kind() != BencodeDict {
			return nil, fmt.Errorf("Query arguments is not a dictionary")
		}

		return krpcQuery{
			transactionId: string(t.(bencodeString)),
			methodName:    string(q.(bencodeString)),
			arguments:     a.(bencodeDict),
		}, nil
	case KrpcTypeReply:
		var r = data["r"]
		if r == nil || r.kind() != BencodeDict {
			return nil, fmt.Errorf("Response is not a dictionary")
		}

		return krpcResponse{
			transactionId: string(t.(bencodeString)),
			response:      r.(bencodeDict),
		}, nil
	case KrpcTypeError:
		var e = data["e"]
		if e == nil || e.kind() != BencodeList {
			return nil, fmt.Errorf("Error is not a list")
		}

		var l = e.(bencodeList)
		if l == nil || len(l) != 2 {
			return nil, fmt.Errorf("Error list does not have two elements")
		}

		var code = l[0]
		if code == nil || code.kind() != BencodeInteger {
			return nil, fmt.Errorf("Error code is not an integer")
		}

		var message = l[1]
		if message == nil || message.kind() != BencodeString {
			return nil, fmt.Errorf("Error message is not a string")
		}

		return krpcError{
			transactionId: string(t.(bencodeString)),
			kind:          int(code.(bencodeInt)),
			message:       string(message.(bencodeString)),
		}, nil
	default:
		return nil, fmt.Errorf("Unknown KRPC message type")
	}
}
