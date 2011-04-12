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
	conn net.Conn
	clientNextId uint16
	writeChan chan []byte
	readChan chan []byte
}

func DialDSI(addr *net.TCPAddr) (*DSITransport, os.Error) {
	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}
	buf := buildDSIRequest(DSIOpenSession, 1, []byte{})
	log.Println("Client:", dsiDecode(buf))
	_, err = conn.Write(buf)
	if err != nil {
		conn.Close()
		return nil, err
	}
	buf = make([]byte, 1500)
	read, err := io.ReadAtLeast(conn, buf, 16)
	if err != nil {
		conn.Close()
		return nil, err
	}
	buf = buf[:read]
	header, rest := buf[:16], buf[16:]
	log.Println("Server:", dsiDecode(header))
	log.Println("Server:",dsiDecodeOptions(rest))
	dsi := &DSITransport{
		conn: conn,
		clientNextId: 1,
		writeChan: make(chan []byte, 1),
		readChan: make(chan []byte, 1),
	}
	go dsi.reader()
	go dsi.writer()
	go dsi.service()
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

func (d *DSITransport) service() {
	for {
 		select {
		case buf := <- d.readChan:
			header := buf[:16]
			switch header[1] {
			case DSICloseSession:
			case DSICommand:
			case DSIGetStatus:

			case DSITickle:
				// tickle response
				d.writeChan <- buildDSIResponse(DSITickle, binary.BigEndian.Uint16(header[2:4]), []byte{})
			case DSIWrite:
			case DSIAttention:
			}
		}
	}
}

func (d *DSITransport) reader() {
	defer d.conn.Close()
	for {
		buf := make([]byte, 1500)
		_, err := io.ReadAtLeast(d.conn, buf, 16)
		if err != nil {
			log.Fatal(err)
		}
		length := int(binary.BigEndian.Uint16(buf[8:12]))
		_, err = io.ReadAtLeast(d.conn, buf[16:], length)
		log.Println("Server:",dsiDecode(buf))
		d.readChan <- buf
	}
}

func (d *DSITransport) writer() {
	for {
		buf := <- d.writeChan
		log.Println("Client:",dsiDecode(buf))
		for len(buf) > 0 {
			written, err := d.conn.Write(buf)
			if err != nil {
				log.Fatal(err)
			}
			buf = buf[written:]
		}
	}
}

func (d *DSITransport) nextClientId() uint16 {
	d.clientNextId = d.clientNextId + 1
	return d.clientNextId
}
