package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
	"unicode/utf16"
)

const (
  C3Emelate_TCPBuffer = 2048

  C3Emelat_EndTimeout = 5 * time.Second
  C3Emelat_ReadTimeout = 1 * time.Second
)

type C3Emelate struct {
  listener *net.TCPListener

  AXIS_ACT       *Position
  POS_ACT        *Position
  
  COM_ACTION     C3VariableComActionValues
  COM_ROUNDM     C3VariableComRoundmValues
  COM_E6AXIS     *Position
  COM_E6POS      *Position
  
  POSITION       *Position

  PROXY_TYPE     string
  PROXY_VERSION  string
  PROXY_HOSTNAME string
  PROXY_ADDRESS  string
  PROXY_PORT     string

  variableMux sync.RWMutex

  shutdownChan chan struct{}
  wg           sync.WaitGroup
}

func NewC3Emelate(port uint16) *C3Emelate {
  return &C3Emelate{
    AXIS_ACT:   NewPosition(PositionType_E6AXIS),
    POS_ACT:    NewRandomPosition(PositionType_E6POS),
    
    COM_ACTION: C3Variable_COM_ACTION_EMPTY,
    COM_ROUNDM: C3Variable_COM_ROUNDM_NONE,
    COM_E6AXIS: NewPosition(PositionType_E6AXIS),
    COM_E6POS:  NewPosition(PositionType_E6POS),

    POSITION:   NewPosition(PositionType_E6POS),
    
    PROXY_TYPE:     "C3 Server Emulator",
    PROXY_VERSION:  "1.0.0",
    PROXY_HOSTNAME: "localhost",
    PROXY_ADDRESS:  "127.0.0.1",
    PROXY_PORT:     fmt.Sprintf("%d", port),
  }
}

func (c3 *C3Emelate) Address() string {
  return fmt.Sprintf("%s:%s", c3.PROXY_ADDRESS, c3.PROXY_PORT)
}

func (c3 *C3Emelate) ListenAndServe() error {
  tcpAddr, err := net.ResolveTCPAddr("tcp4", c3.Address())
  if err != nil {
    return err
  }
  
  listener, err := net.ListenTCP("tcp", tcpAddr)
  if err != nil {
    return err
  }
  c3.listener = listener

  c3.shutdownChan = make(chan struct{})

  go func() {
    for {
      conn, err := listener.Accept()
      if err != nil {
        select {
          case <-c3.shutdownChan:
            fmt.Println("[C3Emelate INFO] Server is shutting down...")
            return
          default:
            fmt.Println("Error accepting connection:", err)
            continue
        }
      }
      c3.wg.Add(1)
      go c3.handleConnection(conn)
    }
  }()

  log.Printf("[C3Emelate INFO] Server start successfully at %s\n", tcpAddr.String())
  return nil
}

func (c3 *C3Emelate) Shutdown() error {
  close(c3.shutdownChan)

  if err := c3.listener.Close(); err != nil {
    return fmt.Errorf("Error closing listener: %w", err)
  }

  doneChan := make(chan struct{})
  go func() {
    c3.wg.Wait()
    close(doneChan)
  }()

  select {
    case <-doneChan:
      log.Printf("[C3Emelate INFO] All connections gracefully closed\n")
      return nil
    case <-time.After(C3Emelat_EndTimeout):
      return fmt.Errorf("[C3Emelate INFO] Timeout waiting for connections to close")
  }
}

func (c3 *C3Emelate) handleConnection(conn net.Conn) {
  defer func() {
    conn.Close()
    c3.wg.Done()
  }()

  buffer := make([]byte, C3Emelate_TCPBuffer)
  for {
    conn.SetReadDeadline(time.Now().Add(C3Emelat_ReadTimeout))
    n, err := conn.Read(buffer)
    if err != nil {
      if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
        continue
      }
      log.Printf("[C3Emelate ERROR] Failed to read request: %v\n", err)
      break
    }
    requestMessage := make([]byte, n)
    copy(requestMessage, buffer[:n])
    // log.Printf("[C3Emelate INFO] <- %s", C3EmelateDumpMessage(requestMessage))

    responseMessage, err := c3.processMessage(requestMessage)
    if err != nil {
      log.Printf("[C3Emelate ERROR] Process message error: %v\n", err)
      break
    }

    // log.Printf("[C3Emelate INFO] -> %s", C3EmelateDumpMessage(responseMessage))

    if _, err := conn.Write(responseMessage); err != nil {
      log.Printf("[C3Emelate ERROR] Failed to write response: %v\n", err)
      break
    }
  }
}

func (c3 *C3Emelate) move(p *Position) {

}

func (c3 *C3Emelate) processMessage(request []byte) ([]byte, error) {
  requestReader := bytes.NewReader(request)
  var responseBuffer bytes.Buffer

  // Read TagID (BigEndian)
  var tagID uint16
  if err := binary.Read(requestReader, binary.BigEndian, &tagID); err != nil {
    return nil, fmt.Errorf("Read TagID error: %w", err)
  }

  // Write TagID (BigEndian)
  if err := binary.Write(&responseBuffer, binary.BigEndian, tagID); err != nil {
    return nil, fmt.Errorf("Write TagID error: %w", err)
  }
  
  // Read MessageLength
  var messageLength uint16
  if err := binary.Read(requestReader, binary.BigEndian, &messageLength); err != nil {
    return nil, fmt.Errorf("Read MessageLength error: %w", err)
  }

  // Placeholder for MessageLength, to be updated later
  if err := binary.Write(&responseBuffer, binary.BigEndian, uint16(0)); err != nil {
    return nil, fmt.Errorf("Write MessageLength Placeholder error: %w", err)
  }

  // Read MessageType (BigEndian)
  var messageType C3MessageType
  if err := binary.Read(requestReader, binary.BigEndian, &messageType); err != nil {
    return nil,fmt.Errorf("Read MessageType error: %w", err)
  }

  // Write MessageType (BigEndian)
  if err := responseBuffer.WriteByte(byte(messageType)); err != nil {
    return nil, fmt.Errorf("Write MessageType error: %w", err)
  }

  if messageType == C3Message_Command_ReadVariable || messageType == C3Message_Command_WriteVariable {
    var variableNameLength uint16
    if err := binary.Read(requestReader, binary.BigEndian, &variableNameLength); err != nil {
      return nil, fmt.Errorf("Read VariableNameLength error: %w", err)
    }

    utf16Chars := make([]uint16, variableNameLength)
    if err := binary.Read(requestReader, binary.LittleEndian, &utf16Chars); err != nil {
      return nil, fmt.Errorf("Read VariableName error: %w", err)
    }

    variableName, err := messageUTF16toString(utf16Chars)
    if err != nil {
      return nil, fmt.Errorf("Parse VariableName error: %w", err)
    }

    var c3Error C3ErrorType = C3Message_Error_Success
    if messageType == C3Message_Command_WriteVariable {
      var variableValueLength uint16
      if err := binary.Read(requestReader, binary.BigEndian, &variableValueLength); err != nil {
        return nil, fmt.Errorf("Read variableValueLength error: %w", err)
      }

      utf16Chars := make([]uint16, variableValueLength)
      if err := binary.Read(requestReader, binary.LittleEndian, &utf16Chars); err != nil {
        return nil, fmt.Errorf("Read variableValue error: %w", err)
      }

      variableValue, err := messageUTF16toString(utf16Chars)
      if err != nil {
        return nil, fmt.Errorf("Parse variableValue error: %w", err)
      }

      c3.variableMux.Lock()
      switch C3VariableType(variableName) {
        case C3Variable_COM_ACTION:
          c3.COM_ACTION = C3VariableComActionValues(variableValue)
        case C3Variable_COM_ROUNDM:
          c3.COM_ROUNDM = C3VariableComRoundmValues(variableValue)
        case C3Variable_COM_E6AXIS:
          c3.COM_E6AXIS.Parse(variableValue)
        case C3Variable_COM_E6POS:
          c3.COM_E6POS.Parse(variableValue)
        default:
          c3Error = C3Message_Error_NotImplemented
      }
      c3.variableMux.Unlock()

      // log.Printf("[C3Emelate INFO] %s <- %s", variableName, variableValue)
    }

    var variableValue string
    c3.variableMux.RLock()
    switch C3VariableType(variableName) {
      case C3Variable_AXIS_ACT:
        variableValue = c3.AXIS_ACT.ValueFull()
      case C3Variable_POS_ACT:
        variableValue = c3.POS_ACT.ValueFull()
      case C3Variable_COM_ACTION:
        variableValue = string(c3.COM_ACTION)
      case C3Variable_COM_ROUNDM:
        variableValue = string(c3.COM_ROUNDM)
      case C3Variable_COM_E6AXIS:
        variableValue = c3.COM_E6AXIS.ValueFull()
      case C3Variable_COM_E6POS:
        variableValue = c3.COM_E6POS.ValueFull()
      case C3Variable_PROXY_TYPE:
        variableValue = c3.PROXY_TYPE
      case C3Variable_PROXY_VERSION:
        variableValue = c3.PROXY_VERSION
      case C3Variable_PROXY_HOSTNAME:
        variableValue = c3.PROXY_HOSTNAME
      case C3Variable_PROXY_ADDRESS:
        variableValue = c3.PROXY_ADDRESS
      case C3Variable_PROXY_PORT:
        variableValue = c3.PROXY_PORT
      default:
        c3Error = C3Message_Error_NotImplemented
    }
    c3.variableMux.RUnlock()

    // log.Printf("[C3Emelate INFO] %s -> %s", variableName, variableValue)

    // Encode VariableValue
    encodedValue := utf16.Encode([]rune(variableValue))
    lengthOfVariableValue := uint16(len(encodedValue))

    // Write LengthofVariableValue (BigEndian)
    if err := binary.Write(&responseBuffer, binary.BigEndian, lengthOfVariableValue); err != nil {
      return nil, err
    }

    // Write VariableValue (LittleEndian)
    for _, char := range encodedValue {
      if err := binary.Write(&responseBuffer, binary.LittleEndian, char); err != nil {
        return nil, err
      }
    }

    // Write ErrorCode (BigEndian)
    if err := binary.Write(&responseBuffer, binary.BigEndian, c3Error); err != nil {
      return nil, fmt.Errorf("Write ErrorCode error: %w", err)
    }

    // Write SuccessFlag (BigEndian)
    var successFlag byte = 0
    if c3Error == C3Message_Error_Success {
      successFlag = 1
    }
    if err := responseBuffer.WriteByte(successFlag); err != nil {
      return nil, fmt.Errorf("Write SuccessFlag error: %w", err)
    }
  } else if messageType == C3Message_Command_ReadMultiple || messageType == C3Message_Command_WriteMultiple {
    var variableCount uint8
    if err := binary.Read(requestReader, binary.BigEndian, &variableCount); err != nil {
      return nil, fmt.Errorf("Read VariableCount error: %w", err)
    }

    // Write NumberofVariables
    if err := responseBuffer.WriteByte(variableCount); err != nil {
      return nil, err
    }

    for i := 0; i < int(variableCount); i++ {
      var c3Error C3ErrorType = C3Message_Error_Success

      var variableNameLength uint16
      if err := binary.Read(requestReader, binary.BigEndian, &variableNameLength); err != nil {
        return nil, fmt.Errorf("Read VariableNameLength of %d error: %w", i, err)
      }

      utf16Chars := make([]uint16, variableNameLength)
      if err := binary.Read(requestReader, binary.LittleEndian, &utf16Chars); err != nil {
        return nil, fmt.Errorf("Read VariableName of %d error: %w", i, err)
      }

      variableName, err := messageUTF16toString(utf16Chars)
      if err != nil {
        return nil, fmt.Errorf("Parse VariableName of %d error: %w", i, err)
      }

      if messageType == C3Message_Command_WriteMultiple {

        var variableValueLength uint16
        if err := binary.Read(requestReader, binary.BigEndian, &variableValueLength); err != nil {
          return nil, fmt.Errorf("Read variableValueLength of %d error: %w", i, err)
        }

        utf16Chars := make([]uint16, variableValueLength)
        if err := binary.Read(requestReader, binary.LittleEndian, &utf16Chars); err != nil {
          return nil, fmt.Errorf("Read variableValue of %d error: %w", i, err)
        }

        variableValue, err := messageUTF16toString(utf16Chars)
        if err != nil {
          return nil, fmt.Errorf("Parse variableValue of %d error: %w", i, err)
        }

        c3.variableMux.Lock()
        switch C3VariableType(variableName) {
          case C3Variable_COM_ACTION:
            c3.COM_ACTION = C3VariableComActionValues(variableValue)
          case C3Variable_COM_ROUNDM:
            c3.COM_ROUNDM = C3VariableComRoundmValues(variableValue)
          case C3Variable_COM_E6AXIS:
            c3.COM_E6AXIS.Parse(variableValue)
          case C3Variable_COM_E6POS:
            c3.COM_E6POS.Parse(variableValue)
          default:
            c3Error = C3Message_Error_NotImplemented
        }
        c3.variableMux.Unlock()

        // log.Printf("[C3Emelate INFO] %s <- %s", variableName, variableValue)
      }

      var variableValue string
      c3.variableMux.RLock()
      switch C3VariableType(variableName) {
        case C3Variable_AXIS_ACT:
          variableValue = c3.AXIS_ACT.ValueFull()
        case C3Variable_POS_ACT:
          variableValue = c3.POS_ACT.ValueFull()
        case C3Variable_COM_ACTION:
          variableValue = string(c3.COM_ACTION)
        case C3Variable_COM_ROUNDM:
          variableValue = string(c3.COM_ROUNDM)
        case C3Variable_COM_E6AXIS:
          variableValue = c3.COM_E6AXIS.ValueFull()
        case C3Variable_COM_E6POS:
          variableValue = c3.COM_E6POS.ValueFull()
        case C3Variable_PROXY_TYPE:
          variableValue = c3.PROXY_TYPE
        case C3Variable_PROXY_VERSION:
          variableValue = c3.PROXY_VERSION
        case C3Variable_PROXY_HOSTNAME:
          variableValue = c3.PROXY_HOSTNAME
        case C3Variable_PROXY_ADDRESS:
          variableValue = c3.PROXY_ADDRESS
        case C3Variable_PROXY_PORT:
          variableValue = c3.PROXY_PORT
        default:
          c3Error = C3Message_Error_NotImplemented
      }
      c3.variableMux.RUnlock()

       // log.Printf("[C3Emelate INFO] %s -> %s", variableName, variableValue)

      // Write ErrorCode (BigEndian)
      // if err := binary.Write(&responseBuffer, binary.BigEndian, c3Error); err != nil {
      //   return nil, fmt.Errorf("Write ErrorCode error: %w", err)
      // }

      // Write ErrorCode (Byte)
      if err := responseBuffer.WriteByte(byte(c3Error)); err != nil {
        return nil, fmt.Errorf("Write ErrorCode error: %w", err)
      }

      // Encode VariableValue
      encodedValue := utf16.Encode([]rune(variableValue))
      lengthOfVariableValue := uint16(len(encodedValue))

      // Write LengthofVariableValue (BigEndian)
      if err := binary.Write(&responseBuffer, binary.BigEndian, lengthOfVariableValue); err != nil {
        return nil, err
      }

      // Write VariableValue (LittleEndian)
      for _, char := range encodedValue {
        if err := binary.Write(&responseBuffer, binary.LittleEndian, char); err != nil {
          return nil, err
        }
      }
    }

    // Write ErrorCode (BigEndian)
    if err := binary.Write(&responseBuffer, binary.BigEndian, C3Message_Error_Success); err != nil {
      return nil, fmt.Errorf("Write ErrorCode error: %w", err)
    }

    // Write SuccessFlag (BigEndian)
    var successFlag byte = 1
    if err := responseBuffer.WriteByte(successFlag); err != nil {
      return nil, fmt.Errorf("Write SuccessFlag error: %w", err)
    }
  }

  // Calculate and update MessageLength
  responseMessageLength := uint16(responseBuffer.Len() - 4)
  buf := responseBuffer.Bytes()
  binary.BigEndian.PutUint16(buf[2:], responseMessageLength)

  return buf, nil
}
