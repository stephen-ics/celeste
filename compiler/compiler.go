package compiler

import (
	"fmt"
	"compiler/ast"
	"compiler/code"
	"compiler/object"
	"sort"
)

type EmittedInstruction struct {
	Opcode code.Opcode
	Position int
}

type Compiler struct {
	constants []object.Object // Generally a read only data sturction, ensure that the constant values follow an LIFO order to keep track of each index, as they are the operands that refer to these values
	symbolTable *SymbolTable
	scopes []CompilationScope
	scopeIndex int
}

type Bytecode struct {
	Instructions code.Instructions
	Constants []object.Object
}

type CompilationScope struct {
	instructions code.Instructions
	lastInstruction EmittedInstruction
	previousInstruction EmittedInstruction
}

func New() *Compiler {
	mainScope := CompilationScope{
		instructions: code.Instructions{},
		lastInstruction: EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	symbolTable := NewSymbolTable()

	for i, v := range object.Builtins {
		symbolTable.DefineBuiltin(i, v.Name) // Only defines index and name, the index will be of use when finding the correct function in the builtins slice
	}

	return &Compiler {
		constants: []object.Object{},
		symbolTable: symbolTable,
		scopes: []CompilationScope{mainScope},
		scopeIndex: 0,
	}
}

func NewWithState(s *SymbolTable, constants []object.Object) *Compiler {
	compiler := New()
	compiler.symbolTable = s
	compiler.constants = constants
	return compiler
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

		if c.lastInstructionIs(code.OpPop) { // This is the previous instruction but also the LAST expression in the statement as all the others were already compiled in the loop, therefore this last instruction pop is omitted as the if statement has a return value
			c.removeLastPop()
		}

		jumpPos := c.emit(code.OpJump, 9999)

		afterConsequencePos := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

		if node.Alternative == nil {
			c.emit(code.OpNull)
		} else {
			err := c.Compile(node.Alternative)
			if err != nil {
				return err
			}

			if c.lastInstructionIs(code.OpPop) {
				c.removeLastPop()
			}
		}

		afterAlternativePos := len(c.currentInstructions())
		c.changeOperand(jumpPos, afterAlternativePos)
	
	case *ast.BlockStatement:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}
	
	case *ast.LetStatement:
		symbol := c.symbolTable.Define(node.Name.Value) // Defines the name to which the function will be bound right before the function is compiled, this will allow the function's body to reference the name of the function, allowing for RECURSIVE FUNCTIONS!!!

		err := c.Compile(node.Value)
		if err != nil {
			return err
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}
	
	case *ast.Identifier:
		symbol, ok := c.symbolTable.Resolve(node.Value)
		if !ok {
			return fmt.Errorf("Undefined variable %s", node.Value)
		}

		c.loadSymbol(symbol)

	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer)) // Emit is the compiler term for generate/output it translates to generate an instruction and add it to a collection of memory, returns the starting point of the just admitted instruction (the operator)
	
	case *ast.StringLiteral:
		str := &object.String{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(str))

	case *ast.Boolean:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}
	
	case *ast.ArrayLiteral:
		for _, el := range node.Elements {
			err := c.Compile(el)
			if err != nil {
				return err
			}
		}

		c.emit(code.OpArray, len(node.Elements))
	
	case *ast.HashLiteral:
		keys := []ast.Expression{}
		for k := range node.Pairs { // In GO, when you iterate through a map with range, it automatically sets the "k" (iterating value) to the key, rather than the whole map 
			keys = append(keys, k)
		}

		sort.Slice(keys, func(i, j int) bool { // func(i, j int) bool defines the order of which the elements should be sorted
			return keys[i].String() < keys[j].String()
		})

		for _, k := range keys {
			err := c.Compile(k)
			if err != nil {
				return err
			}
			err = c.Compile(node.Pairs[k])
			if err != nil {
				return err
			}
		}

		c.emit(code.OpHash, len(node.Pairs)*2)
	
	case *ast.IndexExpression:
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Index)
		if err != nil {
			return err
		}

		c.emit(code.OpIndex)

	case *ast.FunctionLiteral:
		c.enterScope()

		for _, p := range node.Parameters {
			c.symbolTable.Define(p.Value)
		}
		
		err := c.Compile(node.Body)
		if err != nil {
			return err
		}

		if c.lastInstructionIs(code.OpPop) {
			c.replaceLastPopWithReturn()
		}

		if !c.lastInstructionIs(code.OpReturnValue) { // If we could not replace the last statement with a return value (no code.OpPop as the body is empty, there is nothing to pop)
			c.emit(code.OpReturn)
		}

		freeSymbols := c.symbolTable.FreeSymbols // Important that this is called before we leave the scope, as we would not have access to it after we leave the scope
		numLocals := c.symbolTable.numDefinitions
		instructions := c.leaveScope() // Returns compiled instructions of the scope within the function

		for _, s := range freeSymbols {
			c.loadSymbol(s) // Loading the free symbol right after leaving the scope right before the closure OpCode is emitted
		} 

		compiledFn := &object.CompiledFunction{
			Instructions: instructions,
			NumLocals: numLocals,
			NumParameters: len(node.Parameters),
		}
		
		fnIndex := c.addConstant(compiledFn) // Add constant returns the location of the added constant
		c.emit(code.OpClosure, fnIndex, len(freeSymbols)) // Length of free symbols is the amount of free symbols aka the second operand

	case *ast.ReturnStatement:
		err := c.Compile(node.ReturnValue)
		if err != nil {
			return err
		}

		c.emit(code.OpReturnValue)

	case *ast.CallExpression:
		err := c.Compile(node.Function)
		if err != nil {
			return err
		}

		for _, a := range node.Arguments {
			err := c.Compile(a)
			if err != nil {
				return err
			}
		}

		c.emit(code.OpCall, len(node.Arguments))
	}
	
	return nil
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode {
		Instructions: c.currentInstructions(),
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

func (c *Compiler) currentInstructions() code.Instructions {
	return c.scopes[c.scopeIndex].instructions
}

func (c *Compiler) addInstruction(ins []byte) int { //Adds the operator and operand to the instructions slice Note** This is NOT the stack, rather, the stack is ran and updated by the VM
	posNewInstruction := len(c.currentInstructions()) // Starting position of the newly added instruction
	updatedInstructions := append(c.currentInstructions(), ins...) // IMPORTANT*** when a slice of bytes is appended to a slice of bytes, each element is appended 1 by 1, this will NOT result in a slice of slices

	c.scopes[c.scopeIndex].instructions = updatedInstructions

	return posNewInstruction // You do not -1 because you append to the array, and this returns the STARTING POSITION of the newly added instruction
}

func(c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	previous := c.scopes[c.scopeIndex].lastInstruction
	last := EmittedInstruction{Opcode: op, Position: pos}

	c.scopes[c.scopeIndex].previousInstruction = previous
	c.scopes[c.scopeIndex].lastInstruction = last
}

func (c *Compiler) lastInstructionIs(op code.Opcode) bool {
	if len(c.currentInstructions()) == 0 {
		return false
	}

	return c.scopes[c.scopeIndex].lastInstruction.Opcode == op
}

func (c *Compiler) removeLastPop() {
	last := c.scopes[c.scopeIndex].lastInstruction
	previous := c.scopes[c.scopeIndex].previousInstruction

	old := c.currentInstructions()
	new := old[:last.Position] // Splicing the last instruction off (which will be a pop)

	c.scopes[c.scopeIndex].instructions = new
	c.scopes[c.scopeIndex].lastInstruction = previous
}

func (c *Compiler) replaceInstructions(pos int, newInstruction []byte) {
	ins := c.currentInstructions()
	for i := 0; i < len(newInstruction); i++ {
		ins[pos+i] = newInstruction[i]
	}
}

func (c *Compiler) changeOperand(opPos int, operand int) {
	ins := c.currentInstructions()
	op := code.Opcode(ins[opPos])
	newInstruction := code.Make(op, operand)

	c.replaceInstructions(opPos, newInstruction)
}

func (c *Compiler) enterScope() {
	scope := CompilationScope{
		instructions: code.Instructions{},
		lastInstruction: EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	c.scopes = append(c.scopes, scope)
	c.scopeIndex++

	c.symbolTable = NewEnclosedSymbolTable(c.symbolTable)
}

func (c *Compiler) leaveScope() code.Instructions {
	instructions := c.currentInstructions()

	c.scopes = c.scopes[:len(c.scopes)-1]
	c.scopeIndex--

	c.symbolTable = c.symbolTable.Outer

	return instructions
}

func (c *Compiler) replaceLastPopWithReturn() {
	lastPos := c.scopes[c.scopeIndex].lastInstruction.Position
	c.replaceInstructions(lastPos, code.Make(code.OpReturnValue))

	c.scopes[c.scopeIndex].lastInstruction.Opcode = code.OpReturnValue
}

func (c *Compiler) loadSymbol(s Symbol) {
	switch s.Scope {
	case GlobalScope:
		c.emit(code.OpGetGlobal, s.Index)
	case LocalScope:
		c.emit(code.OpGetLocal, s.Index)
	case BuiltinScope:
		c.emit(code.OpGetBuiltin, s.Index)
	case FreeScope:
		c.emit(code.OpGetFree, s.Index)
	}
}