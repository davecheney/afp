package afp

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"fmt"
	"log"
)

const (
	DSICloseSession = 1
	DSICommand      = 2
	DSIGetStatus    = 3
	DSIOpenSession  = 4
	DSITickle       = 5
	DSIWrite        = 6
	DSIAttention    = 8
)

var DSICommands = map[uint8]string {
	DSICloseSession: "DSICloseSession", 
	DSICommand: "DSICommand",
	DSIGetStatus: "DSIGetStatus",
	DSIOpenSession: "DSIOpenSession",
	DSITickle: "DSITickle",
	DSIWrite: "DSIWrite",
	DSIAttention: "DSIAttention",
}

type DSIConn struct {
	net.Conn
	clientNextId uint16
}

type DSIRequest []byte

func (r DSIRequest) Command() uint8 {
	return r[1]
}

func (r DSIRequest) RequestID() uint16 {
	return binary.BigEndian.Uint16(r[2:4])
}

func (r DSIRequest) Length() uint32 {
	return binary.BigEndian.Uint32(r[8:12])
}

func DialDSI(addr *net.TCPAddr) (*DSIConn, os.Error) {
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}
	_, err = conn.Write(buildDSIRequest(DSIOpenSession, 1))
	if err != nil {
		conn.Close()
		return nil, err
	}
	buf := make([]byte, 1500)
	read, err := io.ReadAtLeast(conn, buf, 16)
	if err != nil {
		conn.Close()
		return nil, err
	}
	header, _ := DSIRequest(buf[0:16]), buf[16:read]
	log.Println(dsiDecode(header))
	dsi := &DSIConn {
		conn,
		1,
	}
	go dsi.reader()
	return dsi, nil
}

func buildDSIRequest(command uint8, requestid uint16) []byte {
	return buildDSIHeader(0, command, requestid)
}

func buildDSIResponse(command uint8, requestid uint16) []byte {
	return buildDSIHeader(1, command, requestid)
}

func buildDSIHeader(flags, command uint8, id uint16) []byte {
	buf := make([]byte, 16)
	buf[0] = flags
	buf[1] = command
	binary.BigEndian.PutUint16(buf[2:4], id)
	return buf
}

func dsiDecode(buf []byte) string {
	switch buf[0] {
	case 0:
		// request
		return fmt.Sprintf("DSI {flags=%#x, command=%s(%d), requestID=%#x, writeOffset=%#x, totalDataLength=%#x}", buf[0], DSICommands[buf[1]], buf[1],binary.BigEndian.Uint16(buf[2:4]), binary.BigEndian.Uint16(buf[4:8]), binary.BigEndian.Uint16(buf[8:12]))
	case 1:
		//response
		return fmt.Sprintf("DSI {flags=%#x, command=%s(%d), requestID=%#x, errorCode=%s(%d), totalDataLength=%#x}", buf[0], DSICommands[buf[1]], buf[1],binary.BigEndian.Uint16(buf[2:4]), "", binary.BigEndian.Uint16(buf[4:8]), binary.BigEndian.Uint16(buf[8:12]))
	}
	panic("unreachable")
}

func (d *DSIConn) reader() {
	defer d.Conn.Close()
	for {
		buf := make([]byte, 1500)
		read, err := io.ReadAtLeast(d.Conn, buf, 16)
		if err != nil {
			panic(err)
		}
		header, rest := DSIRequest(buf[0:16]), buf[16:read]
		
		if len(rest) < int(header.Length()) {
			buf := make([]byte, 0, int(header.Length()))
			rest = append(buf, rest...)
			_, err := io.ReadFull(d.Conn, rest) // TODO(dfc) wrong
			if err != nil {
				panic(err)
			}
		}
		log.Println(dsiDecode(header))
		switch header.Command() {
		case DSICloseSession:
		case DSICommand:
		case DSIGetStatus:
		case DSIOpenSession:
			
		case DSITickle:
			// tickle response
			_, err := d.Conn.Write(buildDSIResponse(DSITickle, header.RequestID()))
			if err != nil {
				return 
			}
			
		case DSIWrite:
		case DSIAttention:
		default:
			panic("unreachable")
		}
	}
}

func (d *DSIConn) nextClientId() uint16 {
	d.clientNextId = d.clientNextId + 1
	return d.clientNextId
}
