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
	return d.flags == 0x01
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
		requestId:       uint16(data[3]) | uint16(data[2])<<8,
		errorCode:       uint32(data[7]) | uint32(data[6])<<8 | uint32(data[5])<<16 | uint32(data[4])<<24,
		totalDataLength: uint32(data[11]) | uint32(data[10])<<8 | uint32(data[9])<<16 | uint32(data[8])<<24,
	}, data[16:]
}

type dsiTransport struct {
	conn   net.Conn
	nextid uint16
	mu     sync.Mutex // protects the request map and the nextId
	outstanding map[uint16]chan interface{}
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
		if !header.isResponse() {
			t.conn.Close()
			return nil, fmt.Errorf("dsi: dsiOpenSession response was not a response: %v", header)
		}
		if header.requestId != id {
			t.conn.Close()
			return nil, fmt.Errorf("dsi: RequestID did not match, expected=%d, actual=%d", id, header.requestId)
		}
		//fmt.Printf("dsiOpenSession presented extra data: %v\n", body)	
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
		outstanding: make(map[uint16]chan interface{}),
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
		fmt.Println(header, body)
		switch header.command {
		case dsiCloseSession:
		case dsiCommand:
		case dsiGetStatus:
                        ch, ok := d.outstanding[header.requestId]
                        if !ok {
                                fmt.Printf("dsi: unexpected response %d\n", header.requestId)
                        }
                        ch <- body
                        close(ch)
                        delete(d.outstanding, header.requestId)

		case dsiTickle:
			fmt.Printf("dsi: tickle %d\n", header.requestId)
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
	h := d.nextRequestHeader(dsiOpenSession)
	if err := d.writePacketRaw(h, nil); err != nil {
		return 0, err
	}
	return h.requestId, nil
}

func (d *dsiTransport) nextRequestHeader(command uint8) *dsiHeader {
	return &dsiHeader{
		command: command,
		requestId: d.nextId(),
	}
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

func (d *dsiTransport) writePacket(header *dsiHeader, buf []byte) chan interface{} {
	header.totalDataLength = uint32(len(buf))
	ch := make(chan interface{}, 1)
	d.outstanding[header.requestId] = ch
	if err := d.writePacketRaw(header, buf); err != nil {
		ch <- err
	}
	return ch
}

func (d *dsiTransport) writePacketRaw(header *dsiHeader, buf []byte) error {
	_, err := d.conn.Write(append(header.encode(), buf...))
	return err
}

func (d *dsiTransport) getStatus() interface{} {
	h := &dsiHeader{
		flags: 0x0,
		command: dsiGetStatus,
		requestId: d.nextId(),
	}
	return <- d.writePacket(h, []byte { 15, 0 })
}

func (d *dsiTransport) Close() error {
	err := d.writePacketRaw(d.nextRequestHeader(dsiCloseSession), nil)
	if err != nil {
		return err
	}
	return d.conn.Close()
}
