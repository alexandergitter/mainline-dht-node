package main

import "testing"

func TestKrpcError(t *testing.T) {
	var err = krpcError{
		transactionId: "aa",
		kind:          KrpcErrorGeneric,
		message:       "An Error",
	}

	var encoded = err.encode()
	var expected = "d1:eli201e8:An Errore1:t2:aa1:y1:ee"
	if encoded != expected {
		t.Errorf("Expected %s, got %s", expected, encoded)
	}
}
