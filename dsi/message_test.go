package dsi

import (
	"bytes"
	"testing"
	"reflect"
)

var marshalTests = []struct {
	Expected []byte
	Message  interface{}
}{
	{[]byte{0x0, 0x4, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, dsiOpenSession{Command: 4, RequestId: 1}},
}

func TestMessageMarshal(t *testing.T) {
	for _, m := range marshalTests {
		actual := marshal(m.Message)
		if !bytes.Equal(m.Expected, actual) {
			t.Fatalf("Marshal Failed, expected=%v, actual=%v", m.Expected, actual)
		}
	}
}

func TestMessageUnmarshal(t *testing.T) {
	for _, m := range marshalTests {
		actual, err := decode(m.Expected)
		if err != nil{
			t.Fatalf("Unmarshal failed: %v", err)
		}
		if !reflect.DeepEqual(actual, m.Message) {
			t.Fatalf("Unmarshal Failed, expected=%#v, actual=%#v", m.Message, actual)
		}	
	}
}
