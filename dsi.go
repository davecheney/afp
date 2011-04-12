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

var DSICommands = map[uint8]string{
	DSICloseSession: "DSICloseSession",
	DSICommand:      "DSICommand",
	DSIGetStatus:    "DSIGetStatus",
	DSIOpenSession:  "DSIOpenSession",
	DSITickle:       "DSITickle",
	DSIWrite:        "DSIWrite",
	DSIAttention:    "DSIAttention",
}

type DSITransport struct {
	net.Conn
	clientNextId uint16
}

func DialDSI(addr *net.TCPAddr) (*DSITransport, os.Error) {
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}
	_, err = conn.Write(buildDSIRequest(DSIOpenSession, 1, []byte{}))
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
	buf = buf[:read]
	header, rest := buf[:16], buf[16:]
	log.Println(dsiDecode(header))
	log.Println(dsiDecodeOptions(rest))
	dsi := &DSITransport{
		conn,
		1,
	}
	go dsi.reader()
	// go dsi.writer()
	return dsi, nil
}

func buildDSIRequest(command uint8, requestid uint16, data []byte) []byte {
	return buildDSIHeader(0, command, requestid, data)
}

func buildDSIResponse(command uint8, requestid uint16, data []byte) []byte {
	return buildDSIHeader(1, command, requestid, data)
}

func buildDSIHeader(flags, command uint8, id uint16, data []byte) []byte {
	buf := make([]byte, 16)
	buf[0] = flags
	buf[1] = command
	binary.BigEndian.PutUint16(buf[2:4], id)
	binary.BigEndian.PutUint32(buf[8:12], uint32(len(data)))
	return buf
}

func dsiDecode(buf []byte) string {
	switch buf[0] {
	case 0:
		// request
		return fmt.Sprintf("DSI {flags=%#x, command=%s(%d), requestID=%#x, writeOffset=%#x, totalDataLength=%#x}", buf[0], DSICommands[buf[1]], buf[1], binary.BigEndian.Uint16(buf[2:4]), binary.BigEndian.Uint32(buf[4:8]), binary.BigEndian.Uint32(buf[8:12]))
	case 1:
		//response
		return fmt.Sprintf("DSI {flags=%#x, command=%s(%d), requestID=%#x, errorCode=%s(%d), totalDataLength=%#x}", buf[0], DSICommands[buf[1]], buf[1], binary.BigEndian.Uint16(buf[2:4]), "", binary.BigEndian.Uint32(buf[4:8]), binary.BigEndian.Uint32(buf[8:12]))
	}
	panic("unreachable")
}

func dsiDecodeOptions(buf []byte) string {
	var s []string
	for len(buf) > 0 {
		optionType := buf[0]
		optionLength := buf[1]
		optionData := binary.BigEndian.Uint32(buf[2:2+optionLength])
		buf = buf[2+optionLength:]
		s = append(s, fmt.Sprintf("type=%#x, length=%#x, data=%#x", optionType, optionLength, optionData))
	}
	return fmt.Sprintf("DSI Options %v", s)
}

func (d *DSITransport) reader() {
	defer d.Conn.Close()
	for {
		buf := make([]byte, 1500)
		read, err := io.ReadAtLeast(d.Conn, buf, 16)
		if err != nil {
			panic(err)
		}
		header, rest := buf[0:16], buf[16:read]
		length := binary.BigEndian.Uint16(header[8:12])
		if len(rest) < int(length) {
			buf := make([]byte, 0, int(length))
			rest = append(buf, rest...)
			_, err := io.ReadFull(d.Conn, rest) // TODO(dfc) wrong
			if err != nil {
				panic(err)
			}
		}
		log.Println(dsiDecode(header))
		switch header[1] {
		case DSICloseSession:
		case DSICommand:
		case DSIGetStatus:

		case DSITickle:
			// tickle response
			_, err := d.Conn.Write(buildDSIResponse(DSITickle, binary.BigEndian.Uint16(header[2:4]), []byte{}))
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

func (d *DSITransport) nextClientId() uint16 {
	d.clientNextId = d.clientNextId + 1
	return d.clientNextId
}
