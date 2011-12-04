package afp

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
)

const (
	dsiCloseSession = 1
	dsiCommand      = 2
	dsiGetStatus    = 3
	dsiOpenSession  = 4
	dsiTickle       = 5
	dsiWrite        = 6
	dsiAttention    = 8
)

type dsiHeader struct {
	flags           uint8
	command         uint8
	requestId       uint16
	errorCode       uint32
	totalDataLength uint32
	reserved        uint32
}

func (d *dsiHeader) isRequest() bool {
	return d.flags == 0x00
}

func (d *dsiHeader) isResponse() bool {
	return !d.isRequest()
}

func (d *dsiHeader) encode() []byte {
	return []byte{
		d.flags,
		d.command,
		byte(d.requestId >> 8), byte(d.requestId),
		byte(d.errorCode >> 24), byte(d.errorCode >> 16), byte(d.errorCode >> 8), byte(d.errorCode),
		byte(d.totalDataLength >> 24), byte(d.totalDataLength >> 16), byte(d.totalDataLength >> 8), byte(d.totalDataLength),
		0, 0, 0, 0,
	}
}

func decode(data []byte) (*dsiHeader, []byte) {
	return &dsiHeader{
		flags:           data[0],
		command:         data[1],
		requestId:       uint16(data[2]) | uint16(data[3])<<8,
		errorCode:       uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16 | uint32(data[7])<<24,
		totalDataLength: uint32(data[8]) | uint32(data[9])<<8 | uint32(data[10])<<16 | uint32(data[11])<<24,
	}, data[16:]
}

type dsiTransport struct {
	conn   net.Conn
	nextid uint16
	mu     sync.Mutex // protects the request map and the nextId
}

func dialDSI(network, addr string) (*dsiTransport, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, fmt.Errorf("dsi: unable to connect to %s: %v", addr, err)
	}
	t := newTransport(conn)
	id, err := t.openSession()
	if err != nil {
		t.conn.Close()
		return nil, fmt.Errorf("dsi: unable to execute DSIOpenSession: %v", err)
	}
	header, _, err := t.readPacket()
	if err != nil {
		t.conn.Close()
		return nil, fmt.Errorf("dsi: failed to recieve DSIOpenSession response: %v", err)
	}
	switch header.command {
	case dsiOpenSession:
		if header.isResponse() {
			t.conn.Close()
			return nil, fmt.Errorf("dsi: dsiOpenSession response was not a response: %v", header)
		}
		if header.requestId != id {
			t.conn.Close()
			return nil, fmt.Errorf("dsi: RequestID did not match, expected=%d, actual=%d", id, header.requestId)
		}
		go t.mainloop()
		return t, nil
	default:
		t.conn.Close()
		return nil, fmt.Errorf("dsi: OpenSession error: %T %v", header, header)
	}
	panic("unreachable")
}

func newTransport(conn net.Conn) *dsiTransport {
	return &dsiTransport{
		conn:   conn,
		nextid: uint16(rand.Int31n(65536)),
	}
}

func (d *dsiTransport) nextId() uint16 {
	d.mu.Lock()
	id := d.nextid
	d.nextid++
	d.mu.Unlock()
	return id
}

func (d *dsiTransport) mainloop() {
	defer d.conn.Close()
	for {
		header, body, err := d.readPacket()
		if err != nil {
			fmt.Printf("dsi: read packet failed: %v\n", err)
			break
		}
		switch header.command {
		case dsiCloseSession:
		case dsiCommand:
		case dsiGetStatus:
		case dsiTickle:
		case dsiWrite:
		case dsiAttention:
		default:
			fmt.Printf("dsi: unexpected command: %d\n", header.command)
			break
		}
	}
	fmt.Println("dsi: mainloop exited")
}

func (d *dsiTransport) openSession() (uint16, error) {
	h := &dsiHeader{
		flags:     0x0,
		command:   dsiOpenSession,
		requestId: d.nextId(),
	}
	if _, err := d.conn.Write(h.encode()); err != nil {
		return 0, err
	}
	return h.requestId, nil
}

func (d *dsiTransport) readPacket() (*dsiHeader, []byte, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(d.conn, buf); err != nil {
		return nil, nil, err
	}
	header, body := decode(buf)
	if header.totalDataLength > 0 {
		body = make([]byte, header.totalDataLength)
		if _, err := io.ReadFull(d.conn, body); err != nil {
			return nil, nil, err
		}
	}
	return header, body, nil
}
