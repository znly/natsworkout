package natsworkout

import (
	"encoding/binary"
	"errors"
	"time"
)

// Message is a basic message for debugging.
type Message struct {
	TimestampNano uint64
	Seq           uint64
	Payload       []byte
}

func (m *Message) Reset() {
	m.TimestampNano = 0
	m.Seq = 0
	m.Payload = nil
}

func (m *Message) Timestamp() time.Time {
	return time.Unix(0, int64(m.TimestampNano))
}

func (m *Message) MarshalTo(data []byte) (int, error) {
	if len(data) < 8+len(m.Payload) {
		return 0, errors.New("buffer to small")
	}
	binary.BigEndian.PutUint64(data, m.TimestampNano)
	binary.BigEndian.PutUint64(data[4:], m.Seq)
	copy(data[8:], m.Payload)
	return 8 + len(m.Payload), nil
}

func (m *Message) MarshalBinary() ([]byte, error) {
	data := make([]byte, 8+len(m.Payload))
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *Message) UnmarshalFrom(data []byte) error {
	if len(data) < 8 {
		return errors.New("invalid data")
	}

	m.TimestampNano = binary.BigEndian.Uint64(data)
	m.Seq = binary.BigEndian.Uint64(data[4:])
	m.Payload = data[10:]
	return nil
}

func (m *Message) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return errors.New("invalid data")
	}

	m.TimestampNano = binary.BigEndian.Uint64(data)
	m.Seq = binary.BigEndian.Uint64(data[4:])
	m.Payload = make([]byte, len(data)-10)
	copy(m.Payload, data[10:])
	return nil
}
