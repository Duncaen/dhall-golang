package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/philandstuff/dhall-golang/ast"
	"github.com/philandstuff/dhall-golang/imports"
	"github.com/philandstuff/dhall-golang/parser"
	"github.com/ugorji/go/codec"
)

//go:generate pigeon -o parser/dhall.go parser/dhall.peg

func main() {
	expr, err := parser.ParseReader("-", os.Stdin)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}
	resolvedExpr, err := imports.Load(expr.(ast.Expr))
	if err != nil {
		log.Fatalf("Import resolve error: %v", err)
	}
	inferredType, err := resolvedExpr.TypeWith(ast.EmptyContext())
	if err != nil {
		log.Fatalf("Type error: %v", err)
	}
	inferredType.WriteTo(os.Stderr)
	fmt.Fprintln(os.Stderr)
	resolvedExpr.Normalize().WriteTo(os.Stdout)
	fmt.Println()
	var ch codec.CborHandle
	var buf = new(bytes.Buffer)
	enc := codec.NewEncoder(buf, &ch)
	dec := codec.NewDecoder(buf, &ch)
	enc.Encode(resolvedExpr.Normalize())
	var final interface{}
	dec.Decode(&final)
	fmt.Printf("%+v\n", final)
}
