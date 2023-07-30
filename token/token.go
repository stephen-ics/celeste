package token

type TokenType string

type Token struct {
	Type TokenType
	Literal string
}

const (
	ILLEGAL = "ILLEGAL"
	EOF = "EOF"

	// Identifiers + Literals
	IDENT = "IDENT" // add, foobar, x, y
	INT = "INT" // 123456

	// Operators
	ASSIGN = "="
	PLUS = "+"

	// Delimiter
	COMMA = ","
	SEMICOLON = ";"

	LPAREN = "("
	RPAREN = ")"
	LBRACE = "{"
	RBRACE = "}"

	// Keywords
	FUNCTION = "FUNCTION"
	LET = "LET"
)

var keywords = map[string]TokenType {
	"fn": FUNCTION,
	"let": LET,
}

func LookupIndent(indent string) TokenType {
	if tok, ok := keywords[indent]; ok { // If ok is true --> pattern found in map --> return value of tok
 		return tok
	}
	return IDENT
}