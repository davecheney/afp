package dsi

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

type header struct {
	flags           uint8
	command         uint8
	requestId       uint16
	errorCode       uint32
	totalDataLength uint32
	reserved        uint32
}

func (d *header) isRequest() bool {
	return d.flags == 0x00
}

func (d *header) isResponse() bool {
	return d.flags == 0x01
}

func (d *header) encode() []byte {
	return []byte{
		d.flags,
		d.command,
		byte(d.requestId >> 8), byte(d.requestId),
		byte(d.errorCode >> 24), byte(d.errorCode >> 16), byte(d.errorCode >> 8), byte(d.errorCode),
		byte(d.totalDataLength >> 24), byte(d.totalDataLength >> 16), byte(d.totalDataLength >> 8), byte(d.totalDataLength),
		0, 0, 0, 0,
	}
}

func decode(data []byte) (*header, []byte) {
	return &header{
		flags:           data[0],
		command:         data[1],
		requestId:       uint16(data[3]) | uint16(data[2])<<8,
		errorCode:       uint32(data[7]) | uint32(data[6])<<8 | uint32(data[5])<<16 | uint32(data[4])<<24,
		totalDataLength: uint32(data[11]) | uint32(data[10])<<8 | uint32(data[9])<<16 | uint32(data[8])<<24,
	}, data[16:]
}

type transport struct {
	conn net.Conn
}

type Session struct {
	*transport
	nextid      uint16
	mu          sync.Mutex
	outstanding map[uint16]chan interface{}
}

func Dial(network, addr string) (*Session, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, fmt.Errorf("dsi: unable to connect to %s: %v", addr, err)
	}
	s := &Session{
		transport:   &transport{conn},
		nextid:      uint16(rand.Int31n(65536)),
		outstanding: make(map[uint16]chan interface{}),
	}
	id, err := s.openSession()
	if err != nil {
		s.transport.conn.Close()
		return nil, fmt.Errorf("dsi: unable to execute DSIOpenSession: %v", err)
	}
	header, _, err := s.transport.readPacket()
	if err != nil {
		s.transport.conn.Close()
		return nil, fmt.Errorf("dsi: failed to recieve DSIOpenSession response: %v", err)
	}
	switch header.command {
	case dsiOpenSession:
		if !header.isResponse() {
			s.transport.conn.Close()
			return nil, fmt.Errorf("dsi: dsiOpenSession response was not a response: %v", header)
		}
		if header.requestId != id {
			s.transport.conn.Close()
			return nil, fmt.Errorf("dsi: RequestID did not match, expected=%d, actual=%d", id, header.requestId)
		}
		//fmt.Printf("dsiOpenSession presented extra data: %v\n", body)	
		go s.mainloop()
		return s, nil
	default:
		s.transport.conn.Close()
		return nil, fmt.Errorf("dsi: OpenSession error: %T %v", header, header)
	}
	panic("unreachable")
}

func (s *Session) nextId() uint16 {
	s.mu.Lock()
	id := s.nextid
	s.nextid++
	s.mu.Unlock()
	return id
}

func (s *Session) mainloop() {
	ch := make(chan p, 1)
	go s.transport.mainloop(ch)
	for {
		select {
		case p, ok := <-ch:
			fmt.Println(p)
			if !ok {
				// closed, shutdown
				break
			}
			switch p.header.command {
			case dsiCloseSession:
			case dsiCommand:
			case dsiGetStatus:
				ch, ok := s.outstanding[p.header.requestId]
				if !ok {
					fmt.Printf("dsi: unexpected response %d\n", p.header.requestId)
				}
				ch <- p.body
				close(ch)
				delete(s.outstanding, p.header.requestId)

			case dsiTickle:
				fmt.Printf("dsi: tickle %d\n", p.header.requestId)
			case dsiWrite:
			case dsiAttention:
			default:
				fmt.Printf("dsi: unexpected command: %d\n", p.header.command)
				break
			}
		}
	}
	fmt.Println("dsi: mainloop exited")
}

type p struct {
	*header
	body []byte
}

func (t *transport) mainloop(ch chan p) {
	defer t.conn.Close()
	for {
		header, body, err := t.readPacket()
		if err != nil {
			close(ch)
			break
		}
		ch <- p{
			header,
			body,
		}
	}
}

func (s *Session) openSession() (uint16, error) {
	h := s.nextRequestHeader(dsiOpenSession)
	if err := s.transport.writePacket(h, nil); err != nil {
		return 0, err
	}
	return h.requestId, nil
}

func (s *Session) nextRequestHeader(command uint8) *header {
	return &header{
		command:   command,
		requestId: s.nextId(),
	}
}

func (d *transport) readPacket() (*header, []byte, error) {
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

func (s *Session) writePacket(header *header, buf []byte) chan interface{} {
	header.totalDataLength = uint32(len(buf))
	ch := make(chan interface{}, 1)
	s.outstanding[header.requestId] = ch
	if err := s.transport.writePacket(header, buf); err != nil {
		ch <- err
	}
	return ch
}

func (t *transport) writePacket(header *header, buf []byte) error {
	_, err := t.conn.Write(append(header.encode(), buf...))
	return err
}

func (s *Session) GetStatus() interface{} {
	h := &header{
		flags:     0x0,
		command:   dsiGetStatus,
		requestId: s.nextId(),
	}
	return <-s.writePacket(h, []byte{15, 0})
}

func (s *Session) Close() error {
	err := s.transport.writePacket(s.nextRequestHeader(dsiCloseSession), nil)
	if err != nil {
		return err
	}
	return s.transport.conn.Close()
}
