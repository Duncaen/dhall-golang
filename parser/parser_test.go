package parser_test

import (
	"math"

	. "github.com/philandstuff/dhall-golang/ast"
	"github.com/philandstuff/dhall-golang/parser"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func ParseAndCompare(input string, expected interface{}) {
	root, err := parser.Parse("test", []byte(input))
	Expect(err).ToNot(HaveOccurred())
	Expect(root).To(Equal(expected))
}

func ParseAndFail(input string) {
	_, err := parser.Parse("test", []byte(input))
	Expect(err).To(HaveOccurred())
}

var _ = Describe("Expression", func() {
	DescribeTable("simple expressions", ParseAndCompare,
		Entry("Type", `Type`, Type),
		Entry("Kind", `Kind`, Kind),
		Entry("Sort", `Sort`, Sort),
		Entry("Double", `Double`, Double),
		Entry("DoubleLit", `3.0`, DoubleLit(3.0)),
		Entry("DoubleLit with exponent", (`3E5`), DoubleLit(3e5)),
		Entry("DoubleLit with sign", (`+3.0`), DoubleLit(3.0)),
		Entry("DoubleLit with everything", (`-5.0e1`), DoubleLit(-50.0)),
		Entry("Infinity", `Infinity`, DoubleLit(math.Inf(1))),
		Entry("-Infinity", `-Infinity`, DoubleLit(math.Inf(-1))),
		Entry("Integer", `Integer`, Integer),
		Entry("IntegerLit", `+1234`, IntegerLit(1234)),
		Entry("IntegerLit", `-3`, IntegerLit(-3)),
		Entry("Identifier", `x`, Var{"x", 0}),
		Entry("Identifier with index", `x@1`, Var{"x", 1}),
		Entry("Annotated expression", `3 : Natural`, Annot{NaturalLit(3), Natural}),
	)
	DescribeTable("naturals", ParseAndCompare,
		Entry("Natural", `Natural`, Natural),
		Entry("NaturalLit", `1234`, NaturalLit(1234)),
		Entry("NaturalLit", `3`, NaturalLit(3)),
		Entry("NaturalPlus", `3 + 5`, NaturalPlus{NaturalLit(3), NaturalLit(5)}),
		// Check that if we skip whitespace, it parses
		// correctly as function application, not natural
		// addition
		Entry("Plus without whitespace", `3 +5`, &App{NaturalLit(3), IntegerLit(5)}),
	)
	DescribeTable("lists", ParseAndCompare,
		Entry("List Natural", `List Natural`, &App{List, Natural}),
	)
	// can't test NaN using ParseAndCompare because NaN ≠ NaN
	It("handles NaN correctly", func() {
		root, err := parser.Parse("test", []byte(`NaN`))
		Expect(err).ToNot(HaveOccurred())
		f := float64(root.(DoubleLit))
		Expect(math.IsNaN(f)).To(BeTrue())
	})
	DescribeTable("lambda expressions", ParseAndCompare,
		Entry("simple λ",
			`λ(foo : bar) → baz`,
			&LambdaExpr{
				"foo", Var{"bar", 0}, Var{"baz", 0}}),
		Entry(`simple \`,
			`\(foo : bar) → baz`,
			&LambdaExpr{
				"foo", Var{"bar", 0}, Var{"baz", 0}}),
		Entry("with line comment",
			"λ(foo : bar) --asdf\n → baz",
			&LambdaExpr{
				"foo", Var{"bar", 0}, Var{"baz", 0}}),
		Entry("with block comment",
			"λ(foo : bar) {-asdf\n-} → baz",
			&LambdaExpr{
				"foo", Var{"bar", 0}, Var{"baz", 0}}),
		Entry("simple ∀",
			`∀(foo : bar) → baz`,
			&Pi{
				"foo", Var{"bar", 0}, Var{"baz", 0}}),
		Entry(`simple forall`,
			`forall(foo : bar) → baz`,
			&Pi{
				"foo", Var{"bar", 0}, Var{"baz", 0}}),
		Entry("with line comment",
			"∀(foo : bar) --asdf\n → baz",
			&Pi{
				"foo", Var{"bar", 0}, Var{"baz", 0}}),
	)
	DescribeTable("applications", ParseAndCompare,
		Entry("identifier application",
			`foo bar`,
			&App{
				Var{"foo", 0},
				Var{"bar", 0},
			}),
		Entry("lambda application",
			`(λ(foo : bar) → baz) quux`,
			&App{
				&LambdaExpr{
					"foo", Var{"bar", 0}, Var{"baz", 0}},
				Var{"quux", 0}}),
	)
	Describe("Expected failures", func() {
		// these keywords should fail to parse unless they're part of
		// a larger expression
		DescribeTable("keywords", ParseAndFail,
			Entry("if", `if`),
			Entry("then", `then`),
			Entry("else", `else`),
			Entry("let", `let`),
			Entry("in", `in`),
			Entry("as", `as`),
			Entry("using", `using`),
			Entry("merge", `merge`),
			Entry("constructors", `constructors`),
			Entry("Some", `Some`),
		)
		DescribeTable("other failures", ParseAndFail,
			Entry("annotation without required space", `3 :Natural`),
		)
	})
})
