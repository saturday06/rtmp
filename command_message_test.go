package rtmp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/zhangpeihao/goamf"
)

func TestReadConnectMessage(t *testing.T) {
	in := []byte{
		0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x97, 0x14, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x07, 0x63, // |...............c|
		0x6f, 0x6e, 0x6e, 0x65, 0x63, 0x74, 0x00, 0x3f, 0xf0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, // |onnect.?........|
		0x00, 0x03, 0x61, 0x70, 0x70, 0x02, 0x00, 0x0a, 0x6c, 0x69, 0x76, 0x65, 0x5f, 0x31, 0x30, 0x38, // |..app...live_108|
		0x30, 0x70, 0x00, 0x04, 0x74, 0x79, 0x70, 0x65, 0x02, 0x00, 0x0a, 0x6e, 0x6f, 0x6e, 0x70, 0x72, // |0p..type...nonpr|
		0x69, 0x76, 0x61, 0x74, 0x65, 0x00, 0x08, 0x66, 0x6c, 0x61, 0x73, 0x68, 0x56, 0x65, 0x72, 0x02, // |ivate..flashVer.|
		0x00, 0x24, 0x46, 0x4d, 0x4c, 0x45, 0x2f, 0x33, 0x2e, 0x30, 0x20, 0x28, 0x63, 0x6f, 0x6d, 0x70, // |.$FMLE/3.0 (comp|
		0x61, 0x74, 0x69, 0x62, 0x6c, 0x65, 0x3b, 0x20, 0x4c, 0x61, 0x76, 0x66, 0x35, 0x37, 0x2e, 0x37, // |atible; Lavf57.7|
		0x31, 0x2e, 0x31, 0x30, 0x30, 0x29, 0x00, 0x05, 0x74, 0x63, 0x55, 0x72, 0x6c, 0x02, 0x00, 0x20, // |1.100)..tcUrl.. |
		//0x72, 0x74, 0x6d, 0x70, 0x3a, 0x2f, 0x2f, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0xc3, 0x68, 0x6f, 0x73,  |rtmp://local.hos| What's a 0xc3 ... Maybe this is a bug of ffmpeg.
		0x72, 0x74, 0x6d, 0x70, 0x3a, 0x2f, 0x2f, 0x6c, 0x6f, 0x63, 0x61, 0x6c, 0x68, 0x6f, 0x73, // |rtmp://local.hos|
		0x74, 0x3a, 0x31, 0x39, 0x33, 0x35, 0x2f, 0x6c, 0x69, 0x76, 0x65, 0x5f, 0x31, 0x30, 0x38, 0x30, // |t:1935/live_1080|
		0x70, 0x00, 0x00, 0x09, //                                                                         |0)...|
	}
	inReader := bufio.NewReader(bytes.NewBuffer(in))

	ch, err := readChunkHeader(inReader, nil)
	if err != nil {
		t.Errorf("should be nil, but got %s", err)
	}
	if ch.BasicHeader.ChunkStreamID != 3 {
		t.Errorf("ChunkStreamID should be 3, but got %d", ch.BasicHeader.ChunkStreamID)
	}
	if ch.MessageHeader.MessageTypeID != 20 {
		t.Errorf("MessageTypeID should be 20, but got %d", ch.MessageHeader.MessageTypeID)
	}

	payload := make([]byte, ch.MessageHeader.MessageLength)
	_, err = io.ReadAtLeast(inReader, payload, int(ch.MessageHeader.MessageLength))
	if err != nil {
		t.Errorf("should be nil, but got %s", err)
	}

	amfBuf := bytes.NewBuffer(payload)
	for amfBuf.Len() > 0 {
		v, err := amf.ReadValue(amfBuf)
		if err != nil {
			t.Errorf("should be nil, but got %s", err)
		}
		fmt.Printf("Value: %#v\n", v)
	}
}

func TestReadReleaseStreamMessage(t *testing.T) {
	in := []byte{
		0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x01, 0x00, 0x00, 0x00, 0x00, //                         |............|
		0x00, 0x00, 0x10, 0x00, //                                                                         |....|
		0x43, 0x00, 0x00, 0x00, 0x00, 0x00, 0x25, 0x14, //                                                 |C.....%.|
		0x02, 0x00, 0x0d, 0x72, 0x65, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, // |...releaseStream|
		0x00, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x02, 0x00, 0x08, 0x6d, 0x79, 0x53, // |.@...........myS|
		0x74, 0x72, 0x65, 0x61, 0x6d, //                                                                   |tream|
	}
	inReader := bufio.NewReader(bytes.NewBuffer(in))

	var chs []*ChunkHeader
	for {
		ch, err := readChunkHeader(inReader, chs)
		if err == io.EOF {
			return
		} else if err != nil {
			t.Errorf("should be nil, but got %s", err)
		}
		chs = append(chs, ch)

		payload := make([]byte, ch.MessageHeader.MessageLength)
		_, err = io.ReadAtLeast(inReader, payload, int(ch.MessageHeader.MessageLength))
		if err != nil {
			t.Errorf("should be nil, but got %s", err)
		}
		fmt.Println("ChunkHeader")
		fmt.Printf("  BasicHeader: %#v\n", ch.BasicHeader)
		fmt.Printf("  MessageHeader: %#v\n", ch.MessageHeader)

		if ch.BasicHeader.ChunkStreamID == 3 && ch.MessageHeader.MessageTypeID == 20 {
			amfBuf := bytes.NewBuffer(payload)
			for amfBuf.Len() > 0 {
				v, err := amf.ReadValue(amfBuf)
				if err != nil {
					t.Errorf("should be nil, but got %s", err)
				}
				fmt.Printf("  Value: %#v\n", v)
			}
		} else {
			fmt.Printf("  Value: %#v\n", payload)
		}
	}
}

func TestReadFCPublishAndCreateStreamMessage(t *testing.T) {
	in := []byte{
		0x43, 0x00, 0x00, 0x00, 0x00, 0x00, 0x21, 0x14, 0x02, 0x00, 0x09, 0x46, 0x43, 0x50, 0x75, 0x62, // |C.....!....FCPub|^M
		0x6c, 0x69, 0x73, 0x68, 0x00, 0x40, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x02, 0x00, // |lish.@..........|^M
		0x08, 0x51, 0x4a, 0x67, 0x38, 0x4e, 0x54, 0x6a, 0x70, 0x43, 0x00, 0x00, 0x00, 0x00, 0x00, 0x19, // |.QJg8NTjpC......|^M
		0x14, 0x02, 0x00, 0x0c, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, // |....createStream|^M
		0x00, 0x40, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, // |.@........|^M
	}
	inReader := bufio.NewReader(bytes.NewBuffer(in))

	var chs []*ChunkHeader
	ch, err := readChunkHeader(inReader, chs)
	if err != nil {
		t.Errorf("should be nil, but got %s", err)
	}
	chs = append(chs, ch)
	if ch.BasicHeader.ChunkStreamID != 3 {
		t.Errorf("ChunkStreamID should be 3, but got %d", ch.BasicHeader.ChunkStreamID)
	}
	if ch.MessageHeader.MessageTypeID != 20 {
		t.Errorf("MessageTypeID should be 20, but got %d", ch.MessageHeader.MessageTypeID)
	}

	payload := make([]byte, ch.MessageHeader.MessageLength)
	_, err = io.ReadAtLeast(inReader, payload, int(ch.MessageHeader.MessageLength))
	if err != nil {
		t.Errorf("should be nil, but got %s", err)
	}

	amfBuf := bytes.NewBuffer(payload)
	for amfBuf.Len() > 0 {
		v, err := amf.ReadValue(amfBuf)
		if err != nil {
			t.Errorf("should be nil, but got %s", err)
		}
		fmt.Printf("Value: %#v\n", v)
	}

	ch, err = readChunkHeader(inReader, chs)
	if err != nil {
		t.Errorf("should be nil, but got %s", err)
	}
	chs = append(chs, ch)
	if ch.BasicHeader.ChunkStreamID != 3 {
		t.Errorf("ChunkStreamID should be 3, but got %d", ch.BasicHeader.ChunkStreamID)
	}
	if ch.MessageHeader.MessageTypeID != 20 {
		t.Errorf("MessageTypeID should be 20, but got %d", ch.MessageHeader.MessageTypeID)
	}

	payload = make([]byte, ch.MessageHeader.MessageLength)
	_, err = io.ReadAtLeast(inReader, payload, int(ch.MessageHeader.MessageLength))
	if err != nil {
		t.Errorf("should be nil, but got %s", err)
	}

	amfBuf = bytes.NewBuffer(payload)
	for amfBuf.Len() > 0 {
		v, err := amf.ReadValue(amfBuf)
		if err != nil {
			t.Errorf("should be nil, but got %s", err)
		}
		fmt.Printf("Value: %#v\n", v)
	}
}

func TestParseOnFCPublishAndResultMessage(t *testing.T) {
	in := []byte{
		0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x8c, 0x14, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x0b, 0x6f, // |...............o|
		0x6e, 0x46, 0x43, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // |nFCPublish......|
		0x00, 0x00, 0x00, 0x05, 0x03, 0x00, 0x05, 0x6c, 0x65, 0x76, 0x65, 0x6c, 0x02, 0x00, 0x06, 0x73, // |.......level...s|
		0x74, 0x61, 0x74, 0x75, 0x73, 0x00, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x02, 0x00, 0x17, 0x4e, 0x65, // |tatus..code...Ne|
		0x74, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x2e, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x2e, // |tStream.Publish.|
		0x53, 0x74, 0x61, 0x72, 0x74, 0x00, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, // |Start..descripti|
		0x6f, 0x6e, 0x02, 0x00, 0x1d, 0x46, 0x43, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x20, 0x74, // |on...FCPublish t|
		0x6f, 0x20, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x20, 0x51, 0x4a, 0x67, 0x38, 0x4e, 0x54, 0x6a, // |o stream QJg8NTj|
		0x70, 0x2e, 0x00, 0x08, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x69, 0x64, 0x00, 0x41, 0xc1, 0x2e, // |p...clientid.A..|
		0xb6, 0xc7, 0x80, 0x00, 0x00, 0x00, 0x00, 0x09, //                                                 |........|
	}
	inReader := bufio.NewReader(bytes.NewBuffer(in))

	var chs []*ChunkHeader
	for {
		ch, err := readChunkHeader(inReader, chs)
		if err == io.EOF {
			return
		} else if err != nil {
			t.Errorf("should be nil, but got %s", err)
		}
		chs = append(chs, ch)

		payload := make([]byte, ch.MessageHeader.MessageLength)
		_, err = io.ReadAtLeast(inReader, payload, int(ch.MessageHeader.MessageLength))
		if err != nil {
			t.Errorf("should be nil, but got %s", err)
		}
		fmt.Println("ChunkHeader")
		fmt.Printf("  BasicHeader: %#v\n", ch.BasicHeader)
		fmt.Printf("  MessageHeader: %#v\n", ch.MessageHeader)

		if ch.BasicHeader.ChunkStreamID == 3 && ch.MessageHeader.MessageTypeID == 20 {
			amfBuf := bytes.NewBuffer(payload)
			for amfBuf.Len() > 0 {
				v, err := amf.ReadValue(amfBuf)
				if err != nil {
					t.Errorf("should be nil, but got %s", err)
				}
				fmt.Printf("  Value: %#v\n", v)
			}
		} else {
			fmt.Printf("  Value: %#v\n", payload)
		}
	}
}

func TestReadUserControlMessageStreamBegin(t *testing.T) {
	in := []byte{
		0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // |................|^M
		0x00, 0x01, //                                                                                     |..|^M
	}
	inReader := bufio.NewReader(bytes.NewBuffer(in))

	var chs []*ChunkHeader
	for {
		ch, err := readChunkHeader(inReader, chs)
		if err == io.EOF {
			return
		} else if err != nil {
			t.Errorf("should be nil, but got %s", err)
		}
		chs = append(chs, ch)

		payload := make([]byte, ch.MessageHeader.MessageLength)
		_, err = io.ReadAtLeast(inReader, payload, int(ch.MessageHeader.MessageLength))
		if err != nil {
			t.Errorf("should be nil, but got %s", err)
		}
		fmt.Println("ChunkHeader")
		fmt.Printf("  BasicHeader: %#v\n", ch.BasicHeader)
		fmt.Printf("  MessageHeader: %#v\n", ch.MessageHeader)

		if ch.BasicHeader.ChunkStreamID == 3 && ch.MessageHeader.MessageTypeID == 20 {
			amfBuf := bytes.NewBuffer(payload)
			for amfBuf.Len() > 0 {
				v, err := amf.ReadValue(amfBuf)
				if err != nil {
					t.Errorf("should be nil, but got %s", err)
				}
				fmt.Printf("  Value: %#v\n", v)
			}
		} else {
			fmt.Printf("  Value: %#v\n", payload)
		}
	}
}

func TestReadPublish(t *testing.T) {
	in := []byte{
		0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x26, 0x14, 0x01, 0x00, 0x00, 0x00, //                         |......&.....|
		0x02, 0x00, 0x07, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x00, 0x40, 0x14, 0x00, 0x00, 0x00, // |...publish.@....|^M
		0x00, 0x00, 0x00, 0x05, 0x02, 0x00, 0x08, 0x51, 0x4a, 0x67, 0x38, 0x4e, 0x54, 0x6a, 0x70, 0x02, // |.......QJg8NTjp.|^M
		0x00, 0x04, 0x6c, 0x69, 0x76, 0x65, //                                                             |..live|^M
	}
	inReader := bufio.NewReader(bytes.NewBuffer(in))

	var chs []*ChunkHeader
	for {
		ch, err := readChunkHeader(inReader, chs)
		if err == io.EOF {
			return
		} else if err != nil {
			t.Errorf("should be nil, but got %s", err)
		}
		chs = append(chs, ch)

		payload := make([]byte, ch.MessageHeader.MessageLength)
		_, err = io.ReadAtLeast(inReader, payload, int(ch.MessageHeader.MessageLength))
		if err != nil {
			t.Errorf("should be nil, but got %s", err)
		}
		fmt.Println("ChunkHeader")
		fmt.Printf("  BasicHeader: %#v\n", ch.BasicHeader)
		fmt.Printf("  MessageHeader: %#v\n", ch.MessageHeader)

		if ch.BasicHeader.ChunkStreamID == 3 && ch.MessageHeader.MessageTypeID == 20 {
			amfBuf := bytes.NewBuffer(payload)
			for amfBuf.Len() > 0 {
				v, err := amf.ReadValue(amfBuf)
				if err != nil {
					t.Errorf("should be nil, but got %s", err)
				}
				fmt.Printf("  Value: %#v\n", v)
			}
		} else {
			fmt.Printf("  Value: %#v\n", payload)
		}
	}
}

func TestReadOnStatusPublish(t *testing.T) {
	in := []byte{
		0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x14, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x08, 0x6f, // |...............o|
		0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // |nStatus.........|
		0x05, 0x03, 0x00, 0x05, 0x6c, 0x65, 0x76, 0x65, 0x6c, 0x02, 0x00, 0x06, 0x73, 0x74, 0x61, 0x74, // |....level...stat|
		0x75, 0x73, 0x00, 0x04, 0x63, 0x6f, 0x64, 0x65, 0x02, 0x00, 0x17, 0x4e, 0x65, 0x74, 0x53, 0x74, // |us..code...NetSt|
		0x72, 0x65, 0x61, 0x6d, 0x2e, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x2e, 0x53, 0x74, 0x61, // |ream.Publish.Sta|
		0x72, 0x74, 0x00, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x02, // |rt..description.|
		0x00, 0x14, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x73, 0x68, 0x69, 0x6e, 0x67, 0x20, 0x51, 0x4a, 0x67, // |..Publishing QJg|
		0x38, 0x4e, 0x54, 0x6a, 0x70, 0x2e, 0x00, 0x08, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x69, 0x64, // |8NTjp...clientid|
		0x00, 0x41, 0xc1, 0x2e, 0xb6, 0xc7, 0x80, 0x00, 0x00, 0x00, 0x00, 0x09, //                         |.A..........|
	}
	inReader := bufio.NewReader(bytes.NewBuffer(in))

	var chs []*ChunkHeader
	for {
		ch, err := readChunkHeader(inReader, chs)
		if err == io.EOF {
			return
		} else if err != nil {
			t.Errorf("should be nil, but got %s", err)
		}
		chs = append(chs, ch)

		payload := make([]byte, ch.MessageHeader.MessageLength)
		_, err = io.ReadAtLeast(inReader, payload, int(ch.MessageHeader.MessageLength))
		if err != nil {
			t.Errorf("should be nil, but got %s", err)
		}
		fmt.Println("ChunkHeader")
		fmt.Printf("  BasicHeader: %#v\n", ch.BasicHeader)
		fmt.Printf("  MessageHeader: %#v\n", ch.MessageHeader)

		if ch.BasicHeader.ChunkStreamID == 3 && ch.MessageHeader.MessageTypeID == 20 {
			amfBuf := bytes.NewBuffer(payload)
			for amfBuf.Len() > 0 {
				v, err := amf.ReadValue(amfBuf)
				if err != nil {
					t.Errorf("should be nil, but got %s", err)
				}
				fmt.Printf("  Value: %#v\n", v)
			}
		} else {
			fmt.Printf("  Value: %#v\n", payload)
		}
	}
}
