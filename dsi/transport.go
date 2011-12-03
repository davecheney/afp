package dsi

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
)

type Transport struct {
	r        io.Reader
	w        io.Writer
	closer   io.Closer
	nextid   uint16
	mu       sync.Mutex             // protects the request map and the nextId
	requests map[uint16]chan []byte // a map of outstanding requests
}

const (
	DSICloseSession = 1
	DSICommand      = 2
	DSIGetStatus    = 3
	DSIOpenSession  = 4
	DSITickle       = 5
	DSIWrite        = 6
	DSIAttention    = 8
)

var DSICommands = map[uint8]string{
	DSICloseSession: "DSICloseSession",
	DSICommand:      "DSICommand",
	DSIGetStatus:    "DSIGetStatus",
	DSIOpenSession:  "DSIOpenSession",
	DSITickle:       "DSITickle",
	DSIWrite:        "DSIWrite",
	DSIAttention:    "DSIAttention",
}

const (
	FlagRequest  = 0x00
	FlagResponse = 0x01
)

func Dial(network, addr string) (*Transport, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, fmt.Errorf("dsi: unable to connect to %s: %v", addr, err)
	}
	t := newTransport(conn)
	id, err := t.openSession()
	if err != nil {
		t.closer.Close()
		return nil, fmt.Errorf("dsi: unable to execute DSIOpenSession: %v", err)
	}
	packet, err := t.readPacket()
	if err != nil {
		t.closer.Close()
		return nil, fmt.Errorf("dsi: failed to recieve DSIOpenSession response: %v", err)
	}
	msg, err := decode(packet)
	if err != nil {
		t.closer.Close()
		return nil, fmt.Errorf("dsi: failed to decode DSIOpenSession response: %v", err)
	}
	if msg, ok := msg.(*dsiOpenSession); ok {
		if msg.RequestId == id {
			go t.mainloop()
			return t, nil
		}
	}
	t.closer.Close()
	return nil, fmt.Errorf("dsi: OpenSession error")
}

func newTransport(conn net.Conn) *Transport {
	return &Transport{
		r:        conn,
		w:        conn,
		closer:   conn,
		nextid:   uint16(rand.Int31n(65536)),
		requests: make(map[uint16]chan []byte),
	}
}

func (t *Transport) openSession() (uint16, error) {
	id := t.nextId()
	req := &dsiOpenSession{
		Command:   DSIOpenSession,
		RequestId: id,
	}
	buf := marshal(req)
	if _, err := t.w.Write(buf); err != nil {
		return 0, err
	}
	return id, nil
}

func (t *Transport) nextId() uint16 {
	t.mu.Lock()
	id := t.nextid
	t.nextid++
	t.mu.Unlock()
	return id
}

func (t *Transport) readPacket() ([]byte, error) {
	var head [16]byte
	if _, err := io.ReadFull(t.r, head[:]); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(head[7:12])
	var buf = make([]byte, uint32(len(head))+length)
	copy(buf, head[:])
	if _, err := io.ReadFull(t.r, buf[16:]); err != nil {
		return nil, err
	}
	return buf, nil
}

func (t *Transport) mainloop() {
	defer t.Close()

}

func (t *Transport) Close() {
	t.closer.Close()
}
