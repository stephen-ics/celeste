package vm

import (
	"fmt"
	"compiler/code"
	"compiler/compiler"
	"compiler/object"
)

const StackSize = 2048

type VM struct {
	constants []object.Object
	instructions code.Instructions

	stack []object.Object
	sp int // Always points to the next value. Top of the stack is [sp-1] --> Maybe this is why compiler does not decrease index position
}

func New(bytecode *compiler.Bytecode) *VM {
	return &VM{
		instructions: bytecode.Instructions,
		constants: bytecode.Constants,

		stack: make([]object.Object, StackSize),
		sp: 0, // Always points to the next free slot in the stack (which is why stack[sp-1] accesses the top stack)
	}
}

func (vm *VM) Run() error {
	for ip := 0; ip < len(vm.instructions); ip++ { // ip = instruction pointer
		op := code.Opcode(vm.instructions[ip]) // opcode [byte] [byte] -> index =0 -> finds op code index += 2 -> index = 2 -> instructions[2] = byte not new operator??

		switch op {
		case code.OpConstant:
			constIndex := code.ReadUint16((vm.instructions[ip+1:]))
			ip += 2 // NOTE ** it only incrememnts by operandwidth, this is because ip++ is incremented in the for loop by 1 automatically

			err := vm.push(vm.constants[constIndex])
			if err != nil {
				return err
			}	
		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv:
			err := vm.executeBinaryOperation(op)
			if err != nil {
				return err
			}
			
			vm.push(&object.Integer{Value: result})
		case code.OpPop:
			vm.pop()
		}
	}

	return nil
}

func (vm *VM) StackTop() object.Object {
	if vm.sp == 0 {
		return nil
	}

	return vm.stack[vm.sp-1]
}

func (vm *VM) push(o object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow")
	}

	vm.stack[vm.sp] = o // Overrides the previously popped element in the stack, then increments to the next available spot in the stack
	vm.sp++

	return nil
}

func (vm *VM) pop() object.Object {
	o := vm.stack[vm.sp-1]
	vm.sp--

	return o
}

func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp] // Because it is now popped off, therefore sp-- + 1 -> sp
}

func (vm *VM) executeBinaryOperation(op code.Opcode) error {
	right := vm.pop() // This assumes the right operator is the last one to be pushed onto the stack, this will affect the result for operators like "-"
	left := vm.pop()

	leftType := left.Type()
	rightType := right.Type()

	if leftType == object.INTEGER_OBJ && rightType == object.INTEGER_OBJ {
		return vm.executeBinaryIntegerOperation(op, left, right)
	}

	return fmt.Errorf("unsupported types for binary operation: %s %s", leftType, rightType)
}

func (vm *VM) executeBinaryIntegerOperation(op code.Opcode, left, right object.Object) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	var result int64

	switch op {
	case code.OpAdd:
		result = leftValue + rightValue
	case code.OpSub:
		result = leftValue - rightValue
	case code.OpMul:
		result = leftValue * rightValue
	case code.OpDiv:
		result = leftValue / rightValue
	default: 
		return fmt.Errorf("unknown integer operator: %d", op)
	}

	return vm.push(&object.Integer{Value: result})
}