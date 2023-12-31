package code

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Instructions []byte
type Opcode byte

func (ins Instructions) String() string {
	var out bytes.Buffer

	i := 0
	for i < len(ins) {
		def, err := Lookup(ins[i])
		if err != nil {
			fmt.Fprintf(&out, "ERRORS: %s\n", err)
			continue
		}

		operand, read := ReadOperands(def, ins[i+1:])

		fmt.Fprintf(&out, "%04d %s\n", i, ins.fmtInstruction(def, operand))

		i += 1 + read
	}

	return out.String()
}

func (ins Instructions) fmtInstruction(def *Definition, operands []int) string {
	operandCount := len(def.OperandWidths)

	if len(operands) != operandCount {
		return fmt.Sprintf("ERROR: operand len %d does not match defined %d\n",
			len(operands), operandCount)
	}

	switch operandCount{
	case 0:
		return def.Name
	case 1:
		return fmt.Sprintf("%s %d", def.Name, operands[0])
	case 2:
		return fmt.Sprintf("%s %d %d", def.Name, operands[0], operands[1])
	}

	return fmt.Sprintf("ERROR: unhandled operandCount for %s\n", def.Name)
}

const (
	OpConstant Opcode = iota // Push constant onto stack opcode number (0)
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpPop
	OpTrue
	OpFalse
	OpEqual
	OpNotEqual
	OpGreaterThan // No less than, will implement with compiler reordering
	OpMinus
	OpBang
	OpJumpNotTruthy
	OpJump
	OpNull
	OpSetGlobal
	OpGetGlobal
	OpArray
	OpHash
	OpIndex
	OpCall
	OpReturnValue
	OpReturn
	OpSetLocal
	OpGetLocal
	OpGetBuiltin
	OpClosure
	OpGetFree
)

type Definition struct {
	Name string
	OperandWidths []int
}

var definitions = map[Opcode]*Definition { // This associates the operator (by opcode) to its definition -> operandwidth and name 
	OpConstant: {"OpConstant", []int{2}},
	OpAdd: {"OpAdd", []int{}}, // None of these operators have operands, instead they operate on the top two elements on the stack 
	OpSub: {"OpSub", []int{}},
	OpMul: {"OpMul", []int{}},
	OpDiv: {"OpDiv", []int{}},
	OpPop: {"OpPop", []int{}},
	OpTrue: {"OpTrue", []int{}},
	OpFalse: {"OpFalse", []int{}},
	OpEqual: {"OpEqual", []int{}},
	OpNotEqual: {"OpNotEqual", []int{}},
	OpGreaterThan: {"OpGreaterThan", []int{}}, 
	OpMinus: {"OpMinus", []int{}},
	OpBang: {"OpBang", []int{}},
	OpJumpNotTruthy: {"OpJumpNotTruthy", []int{2}},
	OpJump: {"OpJump", []int{2}},
	OpNull: {"OpNull", []int{}},
	OpSetGlobal: {"OpSetGlobal", []int{2}},
	OpGetGlobal: {"OpGetGlobal", []int{2}},
	OpArray: {"OpArray", []int{2}},
	OpHash: {"OpHash", []int{2}},
	OpIndex: {"OpIndex", []int{}},
	OpCall: {"OpCall", []int{1}},
	OpReturnValue: {"OpReturnValue",[]int{}},
	OpReturn: {"OpReturn", []int{}},
	OpSetLocal: {"OpSetLocal", []int{1}},
	OpGetLocal: {"OpGetLocal", []int{1}},
	OpGetBuiltin: {"OpGetBuiltin", []int{1}},
	OpClosure: {"OpClosure", []int{2, 1}},
	OpGetFree: {"OpGetFree", []int{1}},
}

func Lookup(op byte) (*Definition, error) {
	def, ok := definitions[Opcode(op)] // Converts op from type byte to type Opcode so that it is able to refer to values in the definitions map which only accepts type Opcode
	if !ok {
		return nil, fmt.Errorf("opcode %d undefined", op)
	}

	return def, nil
}

func Make(op Opcode, operands ...int) []byte {
	def, ok := definitions[op]
	if !ok {
		return []byte{}
	}

	instructionLen := 1
	for _, w := range def.OperandWidths { // For each width in operand widths, add that width to instructionLen
		instructionLen += w
	}

	instruction := make([]byte, instructionLen) // Create a byte slice that is that long (to contain all the bytes)
	instruction[0] = byte(op) // Make the first byte the opconstant --> you can byte it because it is a number and go byte() can byte integers

	offset := 1 // Offset 1 because op code takes the 1st byte in the function
	for i, o := range operands {
		width := def.OperandWidths[i]
		switch width {
		case 1:
			instruction[offset] = byte(o)
		case 2: // Will loop through the operands, in this clase in the case the width is 2 (requires 2 bytes to represent it will represent it in big endian style)
			binary.BigEndian.PutUint16(instruction[offset:], uint16(o)) // Stores these two bytes inside of the instructions byte offsetting the op code byte
		}

		offset += width // Increments the offset by the width to make sure next operand is stored at the correct position
	}

	return instruction // returns the final instruction slide with the op and all operands in order (case 2 ordered by big endian)
}

func ReadOperands(def *Definition, ins Instructions) ([]int, int) {
	operands := make([]int, len(def.OperandWidths))
	offset := 0

	for i, width := range def.OperandWidths {
		switch width {
		case 1:
			operands[i] = int(ReadUint8(ins[offset:])) // offset: means you are starting from the offset index :offset means ending at the offset index
		case 2:
			operands[i] = int(ReadUint16(ins[offset:]))
		}

		offset += width
	}

	return operands, offset
}

func ReadUint8(ins Instructions) uint8 { 
	return uint8(ins[0]) // Though the remaining list of instructions is passed in as an argument ins[0] knows to only take the next byte as we know the operand is one byte
}

func ReadUint16(ins Instructions) uint16 {
	return binary.BigEndian.Uint16(ins) // Likewise as mentioned above but binary.BigEndian.Uint16(ins) knows only to take the next two bytes
}