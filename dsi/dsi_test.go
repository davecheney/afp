package dsi

import (
	"testing"
)

const HOST="odessa.local:548"

func TestDSIConnect(t *testing.T) {
	dsi, err := Dial("tcp", HOST)
	if err != nil {
		t.Fatalf("unable to connect dsi transport: %v", err)
	}
	dsi.transport.conn.Close()
}
