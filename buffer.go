package afp

import (
  "encoding/binary"
  )

type buffer struct {
  contents []byte
}

func (b *buffer) WriteUint32(i uint32) {
  j := make([]byte, 4)
  binary.BigEndian.PutUint32(j, i)
  b.contents = append(b.contents, j...)
}

func (b *buffer) WriteUint16(i uint16) {
    j := make([]byte, 2)
    binary.BigEndian.PutUint16(j, i)
    b.contents = append(b.contents, j...)
}
