package vm

import (
	"fmt"
	"interpreter/code"
	"interpreter/compiler"
	"interpreter/object"
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

	vm.stack[vm.sp] = o // Pushing the object onto the virtual machine stack 
	vm.sp++

	return nil
}