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
	code          krpcErrorType
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
		"e": bencodeList{bencodeInt(err.code), bencodeString(err.message)},
	}

	return ben.encode()
}

func decodeKrpcMessage(data bencodeDict) (krpcMessage, error) {
	var t, tValid = data["t"].(bencodeString)
	var y, yValid = data["y"].(bencodeString)

	if !tValid || !yValid {
		return nil, fmt.Errorf("Transaction id or KRPC message type are missing or invalid")
	}

	switch string(y) {
	case KrpcTypeQuery:
		var q, qValid = data["q"].(bencodeString)
		var a, aValid = data["a"].(bencodeDict)

		if !qValid || !aValid {
			return nil, fmt.Errorf("Query method or arguments are missing or invalid")
		}

		return krpcQuery{
			transactionId: string(t),
			methodName:    string(q),
			arguments:     a,
		}, nil
	case KrpcTypeReply:
		var r, rValid = data["r"].(bencodeDict)

		if !rValid {
			return nil, fmt.Errorf("Response is not a dictionary")
		}

		return krpcResponse{
			transactionId: string(t),
			response:      r,
		}, nil
	case KrpcTypeError:
		var e, eValid = data["e"].(bencodeList)

		if !eValid || len(e) != 2 {
			return nil, fmt.Errorf("Error is not a two-element list")
		}

		var code, codeValid = e[0].(bencodeInt)
		var message, messageValid = e[1].(bencodeString)

		if !codeValid || !messageValid {
			return nil, fmt.Errorf("Error code or message are missing or invalid")
		}

		return krpcError{
			transactionId: string(t),
			code:          int(code),
			message:       string(message),
		}, nil
	default:
		return nil, fmt.Errorf("Unknown KRPC message type")
	}
}
