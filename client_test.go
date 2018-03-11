package go_i2cp

import (
	"testing"
)

func TestClient(t *testing.T) {
	client := NewClient(nil)
	client.Connect()
	client.Disconnect()
}
