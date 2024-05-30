package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
)

type OSCOutputStatus int32

const (
	OSCOutputStatus_OK 		OSCOutputStatus = 1
	OSCOutputStatus_Break OSCOutputStatus = 2
	OSCOutputStatus_Error OSCOutputStatus = 3
)

type OSCPacket struct {
	Path   string
	values []any
}

func NewOSCPacket() *OSCPacket {
	return &OSCPacket{
		Path: "",
		values: make([]any, 0),
	}
}

func (p *OSCPacket) Parse(data []byte) error {
	var start int = 0
	byteReader := bytes.NewReader(data)
	reader := bufio.NewReader(byteReader)

	// Read path string
	path, n, err := oscReadPaddedString(reader)
	if err != nil {
		return err
	}
	start += n

	// Read tags string
	typeTags, n, err := oscReadPaddedString(reader)
	if err != nil {
		return err
	}

	// Remove ',' from tags
	typeTags = typeTags[1:]

	var values []any
  for _, char := range typeTags {
  	switch char {
  		case 'i': // int32
  			var i int32
  			err = binary.Read(reader, binary.BigEndian, &i)
				if err != nil {
					return err
				}
				values = append(values, i)

			case 'h': // int64
				var i int64
				err = binary.Read(reader, binary.BigEndian, &i)
				if err != nil {
					return err
				}
				values = append(values, i)

			case 'f': // float32
				var f float32
				err = binary.Read(reader, binary.BigEndian, &f)
				if err != nil {
					return err
				}
				values = append(values, f)

			case 'd': // float64/double
				var d float64
				err = binary.Read(reader, binary.BigEndian, &d)
				if err != nil {
					return err
				}
				values = append(values, d)

			default:
				return fmt.Errorf("Unsupported type tag: %c", char)
  	}	
  }

  p.Path = path
  p.values = values
  return nil
}

func (p *OSCPacket) Bytes() ([]byte, error) {
	data := new(bytes.Buffer)

	_, err := oscWritePaddedString(p.Path, data)
	if err != nil {
		return nil, err
	}

	// Type tag string starts with ","
	lenArgs := len(p.values)
	typetags := make([]byte, lenArgs+1)
	typetags[0] = ','

	// Process the type tags and collect all arguments
	payload := new(bytes.Buffer)

	for i, value := range p.values {
		switch t := value.(type) {
			case int32:
				typetags[i + 1] = 'i'
				err = binary.Write(payload, binary.BigEndian, t)
				if err != nil {
					return nil, err
				}

			case int64:
				typetags[i + 1] = 'h'
				err = binary.Write(payload, binary.BigEndian, t)
				if err != nil {
					return nil, err
				}

			case float32:
				typetags[i+1] = 'f'
				err := binary.Write(payload, binary.BigEndian, t)
				if err != nil {
					return nil, err
				}

			case float64:
				typetags[i+1] = 'd'
				err = binary.Write(payload, binary.BigEndian, t)
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("Unsupported type: %T", t)
		}
	}

	// Write the type tag string to the data buffer
	if _, err := oscWritePaddedString(string(typetags), data); err != nil {
		return nil, err
	}

	// Write the payload (OSC arguments) to the data buffer
	if _, err := data.Write(payload.Bytes()); err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

func (p *OSCPacket) Append(arg any) error {
	switch t := arg.(type) {
		// OSC types check
		case int32, int64, float32, float64:
			p.values = append(p.values, arg)
		default:
			return fmt.Errorf("Unsupported type: %T", t)
	}

	return nil
}

func (p *OSCPacket) Values() []any {
	values := make([]any, len(p.values))
	copy(values, p.values)
	return values
}


func oscReadPaddedString(reader *bufio.Reader) (string, int, error) {
	// Read the string from the reader
	str, err := reader.ReadString(0)
	if err != nil {
		return "", 0, err
	}
	lenStr := len(str)
	n := lenStr

	// Remove the padding bytes (leaving the null delimiter)
	padLen := oscPadBytesNeeded(lenStr)
	if padLen > 0 {
		n += padLen
		padBytes := make([]byte, padLen)
		if _, err = reader.Read(padBytes); err != nil {
			return "", 0, err
		}
	}

	// Strip off the string delimiter
	return str[:lenStr-1], n, nil
}

func oscPadBytesNeeded(elementLen int) int {
	return ((4 - (elementLen % 4)) % 4)
}

func oscWritePaddedString(str string, buf *bytes.Buffer) (int, error) {
	// Truncate at the first null, just in case there is more than one present
	nullIndex := strings.Index(str, "\x00")
	if nullIndex > 0 {
		str = str[:nullIndex]
	}
	// Write the string to the buffer
	n, err := buf.WriteString(str)
	if err != nil {
		return 0, err
	}

	// Always write a null terminator, as we stripped it earlier if it existed
	buf.WriteByte(0)
	n++

	// Calculate the padding bytes needed and create a buffer for the padding bytes
	numPadBytes := oscPadBytesNeeded(n)
	if numPadBytes > 0 {
		padBytes := make([]byte, numPadBytes)
		// Add the padding bytes to the buffer
		n, err := buf.Write(padBytes)
		if err != nil {
			return 0, err
		}
		numPadBytes = n
	}

	return n + numPadBytes, nil
}