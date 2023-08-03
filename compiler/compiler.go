package compiler

import (
	"compiler/ast"
	"compiler/code"
	"compiler/object"
)

type Compiler struct {
	instructions code.Instructions
	constants []object.Object // Generally a read only data sturction, ensure that the constant values follow an LIFO order to keep track of each index, as they are the operands that refer to these values
}

type Bytecode struct {
	Instructions code.Instructions
	Constants []object.Object
}


func New() *Compiler {
	return &Compiler {
		instructions: code.Instructions{},
		constants: []object.Object{},
	}
}

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}
	case *ast.ExpressionStatement:
		err := c.Compile(node.Expression)
		if err != nil {
			return err
		}

	case *ast.InfixExpression: // Currently, the + (aka the operator) is ignored
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer)) // Emit is the compiler term for generate/output it translates to generate an instruction and add it to a collection of memory, returns the starting point of the just admitted instruction (the operator)
	}
	
	return nil
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode {
		Instructions: c.instructions,
		Constants: c.constants,
	}
}

func (c *Compiler) addConstant(obj object.Object) int { // The point of this function is to add it to the constants slice and return the index of the newly added object, an argument for the c.emi() method
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1 // Returns the index of the newly added object
}

func (c *Compiler) emit(op code.Opcode, operands ...int) int { // Takes in the operator and operands and adds it to the instructions slice as BYTES! The operand itself (aka the identifier) is the index to a constant pool
	ins := code.Make(op, operands...) //The index of the constant is then used as an operand for the OpConstant instruction
	pos := c.addInstruction(ins) // Returns the position of the newly added instruction (operator and operand as bytes)
	return pos // returns the position of the instruction added
}

func (c *Compiler) addInstruction(ins []byte) int { //Adds the operator and operand to the instructions slice Note** This is NOT the stack, rather, the stack is ran and updated by the VM
	posNewInstruction := len(c.instructions) // Position of the newly added instruction
	c.instructions = append(c.instructions, ins...)
	return posNewInstruction // Why not -1? -> Some LIFO data structure stuff?
}

