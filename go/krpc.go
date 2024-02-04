package main

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
