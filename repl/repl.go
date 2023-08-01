package repl

import (
	"bufio"
	"fmt"
	"io"
	"interpreter/lexer"
	"interpreter/parser"
//	"interpreter/token"
)

const PROMPT = ">> "

func Start(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)

	for {
		fmt.Printf(PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		l := lexer.New(line)


// 		Outputs tokenization 
//		for tok := l.NextToken(); tok.Type != token.EOF; tok = l.NextToken() {
//			fmt.Printf("%+v\n", tok)
//		}

//		Outputs parsed data 
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			printParserErrors(out, p.Errors())
			continue
		}

		io.WriteString(out, program.String())
		io.WriteString(out, "\n")
	}
}

func printParserErrors(out io.Writer, errors []string) {
	for _, msg := range errors {
		io.WriteString(out, "Woops! We ran into an error here!\n")
		io.WriteString(out, "parser errors:\n")
		io.WriteString(out, "\t" + msg + "\n")
	}
}