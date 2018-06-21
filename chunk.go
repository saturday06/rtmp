package rtmp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
)

var (
	errUnknownFMT           = errors.New("unknown fmt")
	errInvalidChunkStreamID = errors.New("invalid chunk stream id")
	errNoPreceedingChunk    = errors.New("basic header fmt is 3 but no preceeding chunk")
)

// Chunk Header

type ChunkHeader struct {
	BasicHeader       *BasicHeader
	MessageHeader     *MessageHeader
	ExtendedTimestamp uint32
}

func genChunkHeader(ch *ChunkHeader) ([]byte, error) {
	bh, err := genBasicHeader(ch.BasicHeader)
	if err != nil {
		return []byte{}, err
	}
	mh, err := genMessageHeader(ch.MessageHeader, int(ch.BasicHeader.FMT))
	if err != nil {
		return []byte{}, err
	}
	x := append(bh, mh...)

	if ch.MessageHeader.Timestamp >= 16777215 {
		y := make([]byte, 4)
		binary.BigEndian.PutUint32(y, ch.MessageHeader.Timestamp)
		x = append(x, y...)
	} else if ch.MessageHeader.TimestampDelta >= 16777215 {
		y := make([]byte, 4)
		binary.BigEndian.PutUint32(y, ch.MessageHeader.TimestampDelta)
		x = append(x, y...)
	}
	return x, nil
}

func readChunkHeader(br io.Reader, oldHeaders []*ChunkHeader) (*ChunkHeader, int, error) {
	chLen := 0
	bh, bhLen, err := readBasicHeader(br)
	if err != nil {
		return nil, 0, err
	}
	chLen += bhLen

	mh, mhLen, err := readMessageHeader(br, bh, oldHeaders)
	if err != nil {
		return nil, 0, err
	}
	chLen += mhLen

	ch := &ChunkHeader{
		BasicHeader:   bh,
		MessageHeader: mh,
	}

	if mh.Timestamp == 16777215 || mh.TimestampDelta == 16777215 {
		x := make([]byte, 4)
		_, err := io.ReadFull(br, x)
		if err != nil {
			return nil, 0, err
		}
		ch.ExtendedTimestamp = binary.BigEndian.Uint32(x)
		chLen += 4
	}
	return ch, chLen, nil
}

type BasicHeader struct {
	FMT           uint8
	ChunkStreamID uint32
}

func genBasicHeader(bh *BasicHeader) ([]byte, error) {
	if bh.ChunkStreamID < 64 {
		x := uint8(bh.ChunkStreamID&(0x3f)) + (bh.FMT << 6)
		return []byte{x}, nil
	} else if bh.ChunkStreamID < 320 {
		x := make([]byte, 2)
		x[0] = bh.FMT << 6
		x[1] = uint8(bh.ChunkStreamID - 64)
		return x, nil
	} else if bh.ChunkStreamID < 65599 {
		x := make([]byte, 3)
		x[0] = bh.FMT<<6 + uint8(0xff&0x3f)
		binary.BigEndian.PutUint16(x[1:], uint16(bh.ChunkStreamID-64))
		return x, nil
	} else {
		return []byte{}, errInvalidChunkStreamID
	}
}

func readBasicHeader(br io.Reader) (*BasicHeader, int, error) {
	var err error
	xx := make([]byte, 1)
	_, err = io.ReadFull(br, xx)
	if err != nil {
		return nil, 0, err
	}
	x := xx[0]
	fmt.Printf("basic header first byte: %v\n", x)
	if x == 255 {
		panic("oops")
	}

	h := new(BasicHeader)
	h.FMT = uint8(x) >> 6
	csid := x & (32 + 16 + 8 + 4 + 2 + 1)

	switch csid {
	case 0:
		// Chunk stream IDs: 64-319
		//  0                   1
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |fmt|     0     |   cs id - 64  |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		yy := make([]byte, 1)
		_, err := io.ReadFull(br, yy)
		if err != nil {
			return nil, 0, err
		}
		y := yy[0]
		h.ChunkStreamID = uint32(y) + 64
		return h, 2, nil
	case 1:
		// Chunk stream IDs: 64-65599
		//  0                   1                   2
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |fmt|     1     |         cs id - 64            |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		z := make([]byte, 2)
		_, err = io.ReadFull(br, z)
		if err != nil {
			return nil, 0, err
		}
		h.ChunkStreamID = uint32(binary.BigEndian.Uint16(z)) + 64
		return h, 3, nil
	default:
		// Chunk Stream IDs: 2-63
		//  0 1 2 3 4 5 6 7
		// +-+-+-+-+-+-+-+-+
		// |fmt|   cs id   |
		// +-+-+-+-+-+-+-+-+
		h.ChunkStreamID = uint32(csid)
		return h, 1, nil
	}
}

type MessageHeader struct {
	Timestamp       uint32
	TimestampDelta  uint32
	MessageLength   uint32
	MessageTypeID   uint8
	MessageStreamID uint32
}

func genMessageHeader(mh *MessageHeader, fmt int) ([]byte, error) {
	timestamp := mh.Timestamp
	timestampDelta := mh.TimestampDelta
	if timestamp > 16777215 {
		timestamp = 16777215
	}
	if timestampDelta > 16777215 {
		timestampDelta = 16777215
	}

	switch fmt {
	case 0:
		//  0                   1                   2                   3
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |             timestamp                         |message length |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |     message length (cont)     |message type id| msg stream id |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |            message stream id (cont)           |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		x := make([]byte, 11)
		x[0] = byte(timestamp >> 16)
		x[1] = byte(timestamp >> 8)
		x[2] = byte(timestamp)
		x[3] = byte(mh.MessageLength >> 16)
		x[4] = byte(mh.MessageLength >> 8)
		x[5] = byte(mh.MessageLength)
		x[6] = mh.MessageTypeID
		binary.LittleEndian.PutUint32(x[7:], mh.MessageStreamID)
		return x, nil
	case 1:
		//  0                   1                   2                   3
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                 timestamp delta               |message length |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |     message length (cont)     |message type id|
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		x := make([]byte, 7)
		x[0] = byte(timestampDelta >> 16)
		x[1] = byte(timestampDelta >> 8)
		x[2] = byte(timestampDelta)
		x[3] = byte(mh.MessageLength >> 16)
		x[4] = byte(mh.MessageLength >> 8)
		x[5] = byte(mh.MessageLength)
		x[6] = mh.MessageTypeID
		return x, nil
	case 2:
		//  0                   1                   2
		//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// | timestamp delta |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		x := make([]byte, 3)
		x[0] = byte(timestampDelta >> 16)
		x[1] = byte(timestampDelta >> 8)
		x[2] = byte(timestampDelta)
		return x, nil
	case 3:
		// chunk message header is empty
		return []byte{}, nil
	default:
		return []byte{}, errUnknownFMT
	}
}

func readMessageHeader(br io.Reader, bh *BasicHeader, oldHeaders []*ChunkHeader) (*MessageHeader, int, error) {
	mh := new(MessageHeader)
	switch bh.FMT {
	case 0:
		x := make([]byte, 11)
		_, err := io.ReadFull(br, x)
		if err != nil {
			return nil, 0, err
		}
		mh.Timestamp = binary.BigEndian.Uint32(append([]byte{0x0}, x[:3]...))
		mh.MessageLength = binary.BigEndian.Uint32(append([]byte{0x0}, x[3:6]...))
		mh.MessageTypeID = x[6]
		mh.MessageStreamID = binary.LittleEndian.Uint32(x[7:11])
		log.Printf("message stream id=%v be=%v", x[7:11], mh.MessageStreamID)
		return mh, 11, nil
	case 1:
		x := make([]byte, 7)
		_, err := io.ReadFull(br, x)
		if err != nil {
			return nil, 7, err
		}
		mh.TimestampDelta = binary.BigEndian.Uint32(append([]byte{0x0}, x[:3]...))
		mh.MessageLength = binary.BigEndian.Uint32(append([]byte{0x0}, x[3:6]...))
		mh.MessageTypeID = x[6]
		return mh, 7, nil
	case 2:
		x := make([]byte, 3)
		_, err := io.ReadFull(br, x)
		if err != nil {
			return nil, 3, err
		}
		mh.TimestampDelta = binary.BigEndian.Uint32(append([]byte{0x0}, x...))
		return mh, 3, nil
	case 3:
		var prevHeader *ChunkHeader
		for _, h := range oldHeaders {
			if h.BasicHeader.ChunkStreamID == bh.ChunkStreamID {
				prevHeader = h
				break
			}
		}
		if prevHeader == nil {
			return nil, 0, errNoPreceedingChunk
		}
		*mh = *prevHeader.MessageHeader
		return mh, 0, nil
	default:
		return nil, 0, errUnknownFMT
	}
}
