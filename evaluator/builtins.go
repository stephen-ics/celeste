package evaluator

import (
	"fmt"
	"compiler/object"
)

var builtins = map[string]*object.Builtin{
	"len": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments. got=%d, want=1", len(args))
			}
			
			switch args := args[0].(type) { //args[0] is taking the first and only argument of the len() function
			case *object.String:
				return &object.Integer{Value: int64(len(args.Value))}
			case *object.Array:
				return &object.Integer{Value: int64(len(args.Elements))}
			default:
				return newError("argument to `len` not supported, got=%s", args.Type())
			}
		},
	},
	"first": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments, want=1, got=%d", len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("arguments to `first` must be ARRAY, got=%s", args[0].Type())
			}

			arr := args[0].(*object.Array)
			if len(arr.Elements) > 0 {
				return arr.Elements[0]
			}

			return NULL
		},
	},
	"last": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong numbr of arguments, want=1, got=%d", len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("arguments to `first` must be ARRAY, got=%s", args[0].Type())
			}

			arr := args[0].(*object.Array)
			if len(arr.Elements) > 0 {
				return arr.Elements[len(arr.Elements)-1]
			}

			return NULL
		},
	},
	"rest": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return newError("wrong number of arguments, want=1, got=%d", len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("arguments to `first` must be ARRAY, got=%s", args[0].Type())
			}

			arr := args[0].(*object.Array)
			length := len(arr.Elements)

			if length > 0 {
				newElements := make([]object.Object, length-1, length-1)
				copy(newElements, arr.Elements[1:length])
				return &object.Array{Elements: newElements}
			}

			return NULL
		},
	},
	"push": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return newError("wrong number of arguments, want=2, got=%d", len(args))
			}
			if args[0].Type() != object.ARRAY_OBJ {
				return newError("arguments to `first` must be ARRAY, got=%s", args[0].Type())
			}
			
			arr := args[0].(*object.Array)
			length := len(arr.Elements)

			newElements := make([]object.Object, length+1, length+1)
			copy(newElements, arr.Elements)
			newElements[length] = args[1]

			return &object.Array{Elements: newElements}
		},
	},
	"puts": &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			for _, arg := range args {
				fmt.Println(arg.Inspect())
			}

			return NULL
		},
	},
}

