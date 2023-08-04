package compiler

import (
	"fmt"
	"compiler/ast"
	"compiler/code"
	"compiler/object"
)

type EmittedInstruction struct {
	Opcode code.Opcode
	Position int
}

type Compiler struct {
	instructions code.Instructions
	constants []object.Object // Generally a read only data sturction, ensure that the constant values follow an LIFO order to keep track of each index, as they are the operands that refer to these values
	lastInstruction EmittedInstruction
	previousInstruction EmittedInstruction
}

type Bytecode struct {
	Instructions code.Instructions
	Constants []object.Object
}


func New() *Compiler {
	return &Compiler {
		instructions: code.Instructions{},
		constants: []object.Object{},
		lastInstruction: EmittedInstruction{},
		previousInstruction: EmittedInstruction{}, // The instruction before the last instruction
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

		c.emit(code.OpPop)

	case *ast.InfixExpression: // Currently, the + (aka the operator) is ignored
		if node.Operator == "<" {
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}

			err = c.Compile(node.Left)
			if err != nil {
				return err
			}

			c.emit(code.OpGreaterThan)
			return nil
		}

		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "+":
			c.emit(code.OpAdd)
		case "-":
			c.emit(code.OpSub)
		case "*":
			c.emit(code.OpMul)
		case "/":
			c.emit(code.OpDiv)
		case ">":
			c.emit(code.OpGreaterThan)
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}
	
	case *ast.PrefixExpression:
		err := c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "!":
			c.emit(code.OpBang)
		case "-":
			c.emit(code.OpMinus)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
		}
	
	case *ast.IfExpression:
		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999) // Bogus offset that will be back-patched once node.Consequence is compiled

		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}

		if c.lastInstructionIsPop() {
			c.removeLastPop()
		}

		if node.Alternative == nil {
			afterConsequencePos := len(c.instructions)
			c.changeOperand(jumpNotTruthyPos, afterConsequencePos)
		} else {
			jumpPos := c.emit(code.OpJump, 9999)
			afterConsequencePos := len(c.instructions)
			c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

			err := c.Compile(node.Alternative)
			if err != nil {
				return err
			}

			if c.lastInstructionIsPop() {
				c.removeLastPop()
			}

			afterAlternativePos := len(c.instructions)
			c.changeOperand(jumpPos, afterAlternativePos)
		}
	
	case *ast.BlockStatement:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}

	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer)) // Emit is the compiler term for generate/output it translates to generate an instruction and add it to a collection of memory, returns the starting point of the just admitted instruction (the operator)
	
	case *ast.Boolean:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}
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

	c.setLastInstruction(op, pos)

	return pos // returns the position of the instruction added
}

func (c *Compiler) addInstruction(ins []byte) int { //Adds the operator and operand to the instructions slice Note** This is NOT the stack, rather, the stack is ran and updated by the VM
	posNewInstruction := len(c.instructions) // Position of the newly added instruction
	c.instructions = append(c.instructions, ins...)
	return posNewInstruction // Why not -1? -> Some LIFO data structure stuff?
}

func(c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	previous := c.lastInstruction
	last := EmittedInstruction{Opcode: op, Position: pos}

	c.previousInstruction = previous
	c.lastInstruction = last
}

func (c *Compiler) lastInstructionIsPop() bool {
	return c.lastInstruction.Opcode == code.OpPop
}

func (c *Compiler) removeLastPop() {
	c.instructions = c.instructions[:c.lastInstruction.Position]
	c.lastInstruction = c.previousInstruction
}

func (c *Compiler) replaceInstructions(pos int, newInstruction []byte) {
	for i := 0; i < len(newInstruction); i++ {
		c.instructions[pos+i] = newInstruction[i]
	}
}

func (c *Compiler) changeOperand(opPos int, operand int) {
	op := code.Opcode(c.instructions[opPos])
	newInstruction := code.Make(op, operand)

	c.replaceInstructions(opPos, newInstruction)
}