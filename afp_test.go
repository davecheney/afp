package afp

import (
	"testing"
)

const HOST="odessa.local:548"

func TestAFPConnect(t *testing.T) {
	afp, err := Dial("tcp", HOST)
	if err != nil {
		t.Fatalf("unable to connect dsi transport: %v", err)
	}
	afp.Close()
}

func TestAFPGetSrvrInfo(t *testing.T) {
	afp, err := Dial("tcp", HOST)
        if err != nil {
                t.Fatalf("unable to connect dsi transport: %v", err)
        }
        defer afp.Close()
	st, err := afp.GetSrvrInfo()
	if err != nil {
		t.Fatalf("FPGetSrvrInfo failed: %v", err)
	}
	t.Log(st)
}

