package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

type C3MessageType uint8

const (
  C3Message_Command_ReadVariableASCII          C3MessageType = 0
  C3Message_Command_WriteVariableASCII         C3MessageType = 1
  C3Message_Command_ReadArrayASCII             C3MessageType = 2
  C3Message_Command_WriteArrayASCII            C3MessageType = 3
  C3Message_Command_ReadVariable               C3MessageType = 4
  C3Message_Command_WriteVariable              C3MessageType = 5
  C3Message_Command_ReadMultiple               C3MessageType = 6
  C3Message_Command_WriteMultiple              C3MessageType = 7

  C3Message_Command_ProgramControl             C3MessageType = 10
  C3Message_Command_Motion                     C3MessageType = 11
  C3Message_Command_KcpAction                  C3MessageType = 12
  C3Message_Command_ProxyInfo                  C3MessageType = 13
  C3Message_Command_ProxyFeatures              C3MessageType = 14
  C3Message_Command_ProxyInfoEx                C3MessageType = 15
  C3Message_Command_ProxyCrossInfo             C3MessageType = 16
  C3Message_Command_ProxyBenchmark             C3MessageType = 17

  C3Message_Command_FileSetAttribute           C3MessageType = 20
  C3Message_Command_FileNameList               C3MessageType = 21
  C3Message_Command_FileCreate                 C3MessageType = 22
  C3Message_Command_FileDelete                 C3MessageType = 23
  C3Message_Command_FileCopy                   C3MessageType = 24
  C3Message_Command_FileMove                   C3MessageType = 25
  C3Message_Command_FileGetProperties          C3MessageType = 26
  C3Message_Command_FileGetFullName            C3MessageType = 27
  C3Message_Command_FileGetKrcName             C3MessageType = 28
  C3Message_Command_FileWriteContent           C3MessageType = 29
  C3Message_Command_FileReadContent            C3MessageType = 30

  C3Message_Command_CrossSetInfoOn             C3MessageType = 50
  C3Message_Command_CrossSetInfoOff            C3MessageType = 51
  C3Message_Command_CrossGetRobotDirectory     C3MessageType = 52
  C3Message_Command_CrossDownloadDiskToRobot   C3MessageType = 53
  C3Message_Command_CrossDownloadMemToRobot    C3MessageType = 54
  C3Message_Command_CrossUploadFromRobotToDisk C3MessageType = 55
  C3Message_Command_CrossUploadFromRobotToMem  C3MessageType = 56
  C3Message_Command_CrossDeleteRobotProgram    C3MessageType = 57
  C3Message_Command_CrossRobotLevelStop        C3MessageType = 58
  C3Message_Command_CrossControlLevelStop      C3MessageType = 59
  C3Message_Command_CrossRunControlLevel       C3MessageType = 60
  C3Message_Command_CrossSelectModul           C3MessageType = 61
  C3Message_Command_CrossCancelModul           C3MessageType = 62
  C3Message_Command_CrossConfirmAll            C3MessageType = 63
  C3Message_Command_CrossKrcOk                 C3MessageType = 64
  C3Message_Command_CrossIoRestart             C3MessageType = 65
)

type C3ErrorType uint16

const (
  C3Message_Error_General        C3ErrorType = 0
  C3Message_Error_Success        C3ErrorType = 1
  C3Message_Error_Access         C3ErrorType = 2
  C3Message_Error_Argument       C3ErrorType = 3
  C3Message_Error_Memory         C3ErrorType = 4
  C3Message_Error_Pointer        C3ErrorType = 5
  C3Message_Error_Unexpected     C3ErrorType = 6
  C3Message_Error_NotImplemented C3ErrorType = 7
  C3Message_Error_NoInterface    C3ErrorType = 8
  C3Message_Error_Protocol       C3ErrorType = 9
  C3Message_Error_LongAnswer     C3ErrorType = 10
  C3Message_Error_NotReady       C3ErrorType = 0xFFFF
)

var C3ErrorString = [11]string{
  "General",
  "Success",
  "Access",
  "Argument",
  "Memory",
  "Pointer",
  "Unexpected",
  "NotImplemented",
  "NoInterface",
  "Protocol",
  "LongAnswer",
}

type C3Message struct {
  tagID uint16  
  messageType C3MessageType    

  variableNameList      []C3VariableType
  variableValueList     []*string
  variableErrorCodeList []C3ErrorType

  errorCode   C3ErrorType
  successFlag bool
}

func NewC3Message(tagID uint16, variables map[C3VariableType]*string) (*C3Message, error) {
  variableCount := len(variables)

  if variableCount == 0 {
    return nil, fmt.Errorf("C3Message empty variable list error")
  }

  c3 := &C3Message{
    tagID: tagID,
    variableNameList:      make([]C3VariableType, variableCount),
    variableValueList:     make([]*string, variableCount),
    variableErrorCodeList: make([]C3ErrorType, variableCount),

    errorCode: C3Message_Error_NotReady,
    successFlag: false,
  }

  var isReadMessage *bool
  var i int = 0
  for name, value := range variables {
    if isReadMessage == nil {
      isValueNil := value == nil
      isReadMessage = &isValueNil
    } else {
      if (value == nil) != *isReadMessage {
        return nil, fmt.Errorf("C3Message mixed read and write request error")
      }
    }

    c3.variableNameList[i] = name
    c3.variableValueList[i] = value
    i++
  }

  if *isReadMessage == true {
    if variableCount == 1 {
      c3.messageType = C3Message_Command_ReadVariable
    } else {
      c3.messageType = C3Message_Command_ReadMultiple
    }
  } else {
    if variableCount == 1 {
      c3.messageType = C3Message_Command_WriteVariable
    } else {
      c3.messageType = C3Message_Command_WriteMultiple
    }
  }

  return c3, nil
}

func (c3 *C3Message) TagID(value *uint16) uint16 {
  if value != nil {
    c3.tagID = *value
  }
  return c3.tagID
}

func (c3 *C3Message) Error() error {
  if c3.successFlag && c3.errorCode == C3Message_Error_Success {
    return nil
  }

  return fmt.Errorf("C3Message %s error", C3ErrorString[c3.errorCode])
}

func (c3 *C3Message) Variables() []C3Variable {
  variables := make([]C3Variable, 0)
  for i, variableName := range c3.variableNameList {
    variableValue := c3.variableValueList[i]
    errorCode := c3.variableErrorCodeList[i]
    variables = append(variables, C3Variable{
      Name: variableName,
      Value: *variableValue,
      ErrorCode: errorCode,
    })
  }
  return variables
}

func (c3 *C3Message) Request() ([]byte, error) {
  switch c3.messageType {
    case C3Message_Command_ReadVariable:
      return c3.soloReadRequest()
    case C3Message_Command_ReadMultiple:
      return c3.multipleReadRequest()
    case C3Message_Command_WriteVariable:
      return c3.soloWriteRequest()
    case C3Message_Command_WriteMultiple:
      return c3.multipleWriteRequest()
    default:
      return nil, fmt.Errorf("C3Message incorrect message type")
  }
}

func (c3 *C3Message) RequestString() (string, error) {
  data, err := c3.Request()
  if err != nil {
    return "", err
  }

  var hexStrings []string
  for _, b := range data {
    hexStrings = append(hexStrings, fmt.Sprintf("%02x", b))
  }

  return strings.Join(hexStrings, " "), nil
}

func (c3 *C3Message) soloReadRequest() ([]byte, error) {
  var buffer bytes.Buffer

  // Write TagID (BigEndian)
  if err := binary.Write(&buffer, binary.BigEndian, c3.tagID); err != nil {
    return nil, err
  }

  // Placeholder for MessageLength, to be updated later
  if err := binary.Write(&buffer, binary.BigEndian, uint16(0)); err != nil {
    return nil, err
  }

  // Write MessageType (always 4)
  if c3.messageType != C3Message_Command_ReadVariable {
    return nil, fmt.Errorf("C3Message incorrect message type, must be %d (C3Message_Command_ReadVariable)", C3Message_Command_ReadVariable)
  }
  if err := buffer.WriteByte(byte(c3.messageType)); err != nil {
    return nil, err
  }

  // Encode VariableName
  variableName := c3.variableNameList[0]
  encodedName := utf16.Encode([]rune(variableName))
  lengthOfVariableName := uint16(len(encodedName))

  // Write LengthofVariableName (BigEndian)
  if err := binary.Write(&buffer, binary.BigEndian, lengthOfVariableName); err != nil {
    return nil, err
  }

  // Write VariableName (LittleEndian)
  for _, char := range encodedName {
    if err := binary.Write(&buffer, binary.LittleEndian, char); err != nil {
      return nil, err
    }
  }

  // Calculate and update MessageLength
  messageLength := uint16(buffer.Len() - 4)
  buf := buffer.Bytes()
  binary.BigEndian.PutUint16(buf[2:], messageLength)

  return buf, nil
}

func (c3 *C3Message) multipleReadRequest() ([]byte, error) {
  var buffer bytes.Buffer

  // Write TagID (BigEndian)
  if err := binary.Write(&buffer, binary.BigEndian, c3.tagID); err != nil {
    return nil, err
  }

  // Placeholder for MessageLength, to be updated later
  if err := binary.Write(&buffer, binary.BigEndian, uint16(0)); err != nil {
    return nil, err
  }

  // Write MessageType (always 6)
  if c3.messageType != C3Message_Command_ReadMultiple {
    return nil, fmt.Errorf("C3Message incorrect message type, must be %d (C3Message_Command_ReadMultiple)", C3Message_Command_ReadMultiple)
  }
  if err := buffer.WriteByte(byte(c3.messageType)); err != nil {
    return nil, err
  }

  // Write NumberofVariables
  numberOfVariables := uint8(len(c3.variableNameList))
  if err := buffer.WriteByte(numberOfVariables); err != nil {
    return nil, err
  }

  // Write each variable name
  for _, variableName := range c3.variableNameList {
    // Encode VariableName
    encodedName := utf16.Encode([]rune(variableName))
    lengthOfVariable := uint16(len(encodedName))

    // Write LengthofVariable (BigEndian)
    if err := binary.Write(&buffer, binary.BigEndian, lengthOfVariable); err != nil {
      return nil, err
    }

    // Write VariableName (LittleEndian)
    for _, char := range encodedName {
      if err := binary.Write(&buffer, binary.LittleEndian, char); err != nil {
        return nil, err
      }
    }
  }

  // Calculate and update MessageLength
  messageLength := uint16(buffer.Len() - 4)
  buf := buffer.Bytes()
  binary.BigEndian.PutUint16(buf[2:], messageLength)

  return buf, nil
}

func (c3 *C3Message) soloWriteRequest() ([]byte, error) {
  var buffer bytes.Buffer

  // Write TagID (BigEndian)
  if err := binary.Write(&buffer, binary.BigEndian, c3.tagID); err != nil {
    return nil, err
  }

  // Placeholder for MessageLength, to be updated later
  if err := binary.Write(&buffer, binary.BigEndian, uint16(0)); err != nil {
    return nil, err
  }

  // Write MessageType (always 5)
  if c3.messageType != C3Message_Command_WriteVariable {
    return nil, fmt.Errorf("C3Message incorrect message type, must be %d (C3Message_Command_WriteVariable)", C3Message_Command_WriteVariable)
  }
  if err := buffer.WriteByte(byte(c3.messageType)); err != nil {
    return nil, err
  }

  // Encode VariableName
  variableName := c3.variableNameList[0]
  encodedName := utf16.Encode([]rune(variableName))
  lengthOfVariableName := uint16(len(encodedName))

  // Write LengthofVariableName (BigEndian)
  if err := binary.Write(&buffer, binary.BigEndian, lengthOfVariableName); err != nil {
    return nil, err
  }

  // Write VariableName (LittleEndian)
  for _, char := range encodedName {
    if err := binary.Write(&buffer, binary.LittleEndian, char); err != nil {
      return nil, err
    }
  }

  // Encode and write VariableValue
  variableValue := c3.variableValueList[0]
  encodedValue := utf16.Encode([]rune(*variableValue))
  lengthOfVariableValue := uint16(len(encodedValue))

  // Write LengthofVariableValue (BigEndian)
  if err := binary.Write(&buffer, binary.BigEndian, lengthOfVariableValue); err != nil {
    return nil, err
  }

  // Write VariableValue (LittleEndian)
  for _, char := range encodedValue {
    if err := binary.Write(&buffer, binary.LittleEndian, char); err != nil {
      return nil, err
    }
  }

  // Calculate and update MessageLength
  messageLength := uint16(buffer.Len() - 4)
  buf := buffer.Bytes()
  binary.BigEndian.PutUint16(buf[2:], messageLength)

  return buf, nil
}

func (c3 *C3Message) multipleWriteRequest() ([]byte, error) {
  var buffer bytes.Buffer

  // Write TagID (BigEndian)
  if err := binary.Write(&buffer, binary.BigEndian, c3.tagID); err != nil {
    return nil, err
  }

  // Placeholder for MessageLength, to be updated later
  if err := binary.Write(&buffer, binary.BigEndian, uint16(0)); err != nil {
    return nil, err
  }

  // Write MessageType (always 7)
  if c3.messageType != C3Message_Command_WriteMultiple {
    return nil, fmt.Errorf("C3Message incorrect message type, must be %d (C3Message_Command_WriteMultiple)", C3Message_Command_WriteMultiple)
  }
  if err := buffer.WriteByte(byte(c3.messageType)); err != nil {
    return nil, err
  }

  // Write NumberofVariableNameValue
  numberOfVariableNameValue := uint8(len(c3.variableNameList))
  if err := buffer.WriteByte(numberOfVariableNameValue); err != nil {
    return nil, err
  }

  // Write each variable name and value
  for i, variableName := range c3.variableNameList {
    // Encode and  VariableName
    encodedName := utf16.Encode([]rune(variableName))
    lengthOfVariableName := uint16(len(encodedName))

    // Write LengthofVariableName (BigEndian)
    if err := binary.Write(&buffer, binary.BigEndian, lengthOfVariableName); err != nil {
      return nil, err
    }

    // Write VariableName (LittleEndian)
    for _, char := range encodedName {
      if err := binary.Write(&buffer, binary.LittleEndian, char); err != nil {
        return nil, err
      }
    }

    // Encode and write VariableValue
    encodedValue := utf16.Encode([]rune(*c3.variableValueList[i]))
    lengthOfVariableValue := uint16(len(encodedValue))

    // Write LengthofVariableValue (BigEndian)
    if err := binary.Write(&buffer, binary.BigEndian, lengthOfVariableValue); err != nil {
      return nil, err
    }

    // Write VariableValue (LittleEndian)
    for _, char := range encodedValue {
      if err := binary.Write(&buffer, binary.LittleEndian, char); err != nil {
        return nil, err
      }
    }
  }

  // Calculate and update MessageLength
  messageLength := uint16(buffer.Len() - 4)
  buf := buffer.Bytes()
  binary.BigEndian.PutUint16(buf[2:], messageLength)

  return buf, nil
}

func (c3 *C3Message) Response(packet []byte) error {
  reader := bytes.NewReader(packet)

  var tagID uint16
  if err := binary.Read(reader, binary.BigEndian, &tagID); err != nil {
    return fmt.Errorf("Packet read TagID error: %w", err)
  }

  if tagID != c3.tagID {
    return fmt.Errorf("Message TagID[%d] not equal packet TagID[%d]", c3.tagID, tagID)
  }

  var messageLength uint16
  if err := binary.Read(reader, binary.BigEndian, &messageLength); err != nil {
    return fmt.Errorf("Packet read MessageLength error: %w", err)
  }

  var messageType C3MessageType
  if err := binary.Read(reader, binary.BigEndian, &messageType); err != nil {
    return fmt.Errorf("Packet read MessageType error: %w", err)
  }

  if messageType != c3.messageType {
    return fmt.Errorf("Message MessageType[%d] not equal packet MessageType[%d]", c3.messageType, messageType)
  }

  if messageType == C3Message_Command_ReadVariable || messageType == C3Message_Command_WriteVariable {
    var variableValueLength uint16
    if err := binary.Read(reader, binary.BigEndian, &variableValueLength); err != nil {
      return fmt.Errorf("Packet read VariableValueLength error: %w", err)
    }

    utf16Chars := make([]uint16, variableValueLength)
    if err := binary.Read(reader, binary.LittleEndian, &utf16Chars); err != nil {
      return fmt.Errorf("Packet read VariableValue error: %w", err)
    }

    var utf8Buffer bytes.Buffer
    for _, r := range utf16.Decode(utf16Chars) {
      var buf [4]byte
      n := utf8.EncodeRune(buf[:], r)
      _, err := utf8Buffer.Write(buf[:n])
      if err != nil {
        return fmt.Errorf("Packet parse VariableValue error: %w", err)
      }
    }

    variableValue := utf8Buffer.String()
    c3.variableValueList[0] = &variableValue
    c3.variableErrorCodeList[0] = C3Message_Error_Success
  
  } else if messageType == C3Message_Command_ReadMultiple || messageType == C3Message_Command_WriteMultiple {

    var variableCount uint8
    if err := binary.Read(reader, binary.BigEndian, &variableCount); err != nil {
      return fmt.Errorf("Packet read VariableCount error: %w", err)
    }

    messageVariableCount := len(c3.variableNameList)
    if variableCount != uint8(messageVariableCount) {
      return fmt.Errorf("Message VariableCount[%d] not equal packet VariableCount[%d]", messageVariableCount, variableCount)
    }

    for i := 0; i < messageVariableCount; i++ {
      var variableErrorCode uint8
      if err := binary.Read(reader, binary.BigEndian, &variableErrorCode); err != nil {
        return fmt.Errorf("Packet read Variable %d ErrorCode error: %w", i, err)
      }

      c3.variableErrorCodeList[i] = C3ErrorType(variableErrorCode)

      var variableValueLength uint16
      if err := binary.Read(reader, binary.BigEndian, &variableValueLength); err != nil {
        return fmt.Errorf("Packet read Variable %d ValueLength error: %w", i, err)
      }

      utf16Chars := make([]uint16, variableValueLength)
      if err := binary.Read(reader, binary.LittleEndian, &utf16Chars); err != nil {
        return fmt.Errorf("Packet read Variable %d Value error: %w", i, err)
      }

      var utf8Buffer bytes.Buffer
      for _, r := range utf16.Decode(utf16Chars) {
        var buf [4]byte
        n := utf8.EncodeRune(buf[:], r)
        _, err := utf8Buffer.Write(buf[:n])
        if err != nil {
          return fmt.Errorf("Packet parse Variable %d Value error: %w", i, err)
        }
      }

      variableValue := utf8Buffer.String()
      c3.variableValueList[i] = &variableValue
    }

  }

  if err := binary.Read(reader, binary.BigEndian, &c3.errorCode); err != nil {
    return fmt.Errorf("Packet read ErrorCode error: %w", err)
  }

  if err := binary.Read(reader, binary.BigEndian, &c3.successFlag); err != nil {
    return fmt.Errorf("Packet read SuccessFlag error: %w", err)
  }

  return nil 
  
}
