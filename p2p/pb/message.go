package pb

import (
	"encoding/binary"
	"bytes"
	"github.com/golang/protobuf/proto"
	"errors"
)

/*
	Message protocol
0               1               2               3
0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  						 Data Length                          |
 */

const (
	DATA_LENGTH_SIZE = uint32(4)
	DATA_MAX_LENGTH  = uint32(4 * 1024 * 1024)
)

func NewMessage(name string, data []byte) (*Message, error) {
	msg := &Message{
		Name: name,
		Data: data,
	}
	if msg.Length() > DATA_MAX_LENGTH {
		return nil, errors.New("Message data is too long")
	}
	return msg, nil
}

//func NewMessage(name string, data MessageData) (*Message, error) {
//	message := &Message{
//		name,
//		data,
//	}
//	if message.Length() > DATA_MAX_LENGTH {
//		return nil, errors.New(fmt.Sprintf("Message %s is too long. Max data length is %d.\n", name, DATA_MAX_LENGTH))
//	}
//	return message, nil
//}

func (msg *Message) Length() uint32 {
	return uint32(len(msg.Name) + len(msg.Data))
}

func (msg *Message) Serialize() ([]byte, error) {
	//data, err := json.Marshal(msg)
	//if err != nil {
	//	return nil, err
	//}
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	// Data length prefix
	length := uint32(len(data))
	prefix, err := Uint32ToBytes(length)
	if err != nil {
		return nil, err
	}

	data = append(prefix, data...)
	return data, nil
}

func DeserializeMsg(data []byte) (*Message, error) {
	buf := data[4:]
	msg := &Message{}
	err := proto.Unmarshal(buf, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func Uint32ToBytes(i uint32) ([]byte, error) {
	prefix := []uint32{i}
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, prefix)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func BytesToUint32(buf []byte) (uint32, error) {
	check := make([]uint32, 1)
	rbuf := bytes.NewReader(buf)
	err := binary.Read(rbuf, binary.BigEndian, &check)
	if err != nil {
		return 0, err
	}
	return check[0], nil
}
