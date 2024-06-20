package main

type C3VariableType string

const (
	C3Variable_AXIS_ACT C3VariableType = "$AXIS_ACT"
	C3Variable_POS_ACT  C3VariableType = "$POS_ACT"

	C3Variable_COM_ACTION C3VariableType = "COM_ACTION"
	C3Variable_COM_E6AXIS C3VariableType = "COM_E6AXIS"
	C3Variable_COM_E6POS  C3VariableType = "COM_E6POS"
	C3Variable_COM_ROUNDM C3VariableType = "COM_ROUNDM"
	
	C3Variable_PROXY_TYPE     C3VariableType = "@PROXY_TYPE"
	C3Variable_PROXY_VERSION  C3VariableType = "@PROXY_VERSION"
	C3Variable_PROXY_HOSTNAME C3VariableType = "@PROXY_HOSTNAME"
	C3Variable_PROXY_ADDRESS  C3VariableType = "@PROXY_ADDRESS"
	C3Variable_PROXY_PORT     C3VariableType = "@PROXY_PORT"

	C3Variable_COM_VALUE1 C3VariableType = "COM_VALUE1" // $VEL.CP
	C3Variable_COM_VALUE2 C3VariableType = "COM_VALUE2" // $VEL_AXIS
	C3Variable_COM_VALUE3 C3VariableType = "COM_VALUE3" // $ACC.CP
	C3Variable_COM_VALUE4 C3VariableType = "COM_VALUE4" // $ACC_AXIS
)

type C3VariableComActionValues string 

const (
	C3Variable_COM_ACTION_EMPTY    C3VariableComActionValues = "1" // Empty command
	C3Variable_COM_ACTION_E6AXIS   C3VariableComActionValues = "2" // Move Joints
	C3Variable_COM_ACTION_E6POS    C3VariableComActionValues = "3" // Move Linear
	C3Variable_COM_ACTION_VELCP    C3VariableComActionValues = "6" // Set Speed
	C3Variable_COM_ACTION_VEL_AXIS C3VariableComActionValues = "7" // Set Speed Advanced
)

type C3VariableComRoundmValues string 

const (
	C3Variable_COM_ROUNDM_NONE C3VariableComRoundmValues = "-1" // None value
)

type C3Variable struct {
  Name      C3VariableType
  Value     string
  ErrorCode C3ErrorType
}
