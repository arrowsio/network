package network
import (
	"errors"
	"encoding/binary"
)

type header struct {
	data []byte
}

func createHeader(typ uint16, chunk uint32) []byte {
	header := make([]byte, 0, 8)

	typeBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(typeBytes, typ)
	chunkBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(chunkBytes, chunk)

	header = append(header, 0x1F, 0xFA)
	header = append(header, typeBytes...)
	header = append(header, chunkBytes...)
	return header
}

func headerFrom(d []byte) (h *header, err error) {
	if len(d) < 8 {
		return nil, errors.New("Header not to size.")
	}
	return &header{d}, nil
}

func (h *header) isValid() bool {
	return h.data[0] == 0x1F && h.data[1] == 0xFA
}

func (h *header) getType() uint16 {
	return binary.BigEndian.Uint16(h.data[2:4])
}

func (h *header) chunkSize() uint32 {
	return binary.BigEndian.Uint32(h.data[4:])
}