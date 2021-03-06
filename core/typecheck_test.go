package core

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("functionCheck",
	func(in, out, expected Universe) {
		Expect(functionCheck(in, out)).To(Equal(expected))
	},
	Entry(`Type ↝ Type : Type`, Type, Type, Type),
	Entry(`Kind ↝ Type : Type`, Kind, Type, Type),
	Entry(`Sort ↝ Type : Type`, Sort, Type, Type),
	Entry(`Type ↝ Kind : Kind`, Type, Kind, Kind),
	Entry(`Kind ↝ Kind : Kind`, Kind, Kind, Kind),
	Entry(`Sort ↝ Kind : Sort`, Sort, Kind, Sort),
	Entry(`Type ↝ Sort : Sort`, Type, Sort, Sort),
	Entry(`Kind ↝ Sort : Sort`, Kind, Sort, Sort),
	Entry(`Sort ↝ Sort : Sort`, Sort, Sort, Sort),
)

func typecheckTest(t Term, expectedType Value) {
	actualType, err := TypeOf(t)
	Ω(err).ShouldNot(HaveOccurred())
	Ω(judgmentallyEqualVals(actualType, expectedType)).Should(
		BeTrue())
}

var _ = Describe("TypeOf", func() {
	DescribeTable("Universe",
		typecheckTest,
		Entry("Type : Kind", Type, Kind),
		Entry("Kind : Sort", Kind, Sort),
	)
	DescribeTable("Builtin",
		typecheckTest,
		Entry(`Natural : Type`, Natural, Type),
		Entry(`List : Type -> Type`, List, NewFnTypeVal("_", Type, Type)),
	)
	DescribeTable("Lambda",
		typecheckTest,
		Entry("λ(x : Natural) → x : ∀(x : Natural) → Natural",
			NewLambda("x", Natural, NewVar("x")),
			NewPiVal("x", Natural, func(Value) Value { return Natural })),
		Entry("λ(a : Type) → ([] : List a) : ∀(a : Type) → List a -- check presence of variables in resulting type",
			NewLambda("a", Type,
				EmptyList{AppTerm{List, NewVar("a")}}),
			NewPiVal("a", Type, func(a Value) Value {
				return AppValue{List, a}
			})),
		Entry("λ(a : Natural) → assert : a ≡ a -- check presence of variables in resulting type",
			NewLambda("a", Natural,
				Assert{OpTerm{EquivOp, NewVar("a"), NewVar("a")}}),
			NewPiVal("a", Natural, func(a Value) Value {
				return opValue{EquivOp, a, a}
			})),
	)
	DescribeTable("Pi",
		typecheckTest,
		Entry(`Natural → Natural : Type`, NewAnonPi(Natural, Natural), Type),
	)
	DescribeTable("Application",
		typecheckTest,
		Entry(`List Natural : Type`, AppTerm{List, Natural}, Type),
		Entry("(λ(a : Natural) → assert : a ≡ a) 3 -- check presence of variables in resulting type",
			Apply(
				NewLambda("a", Natural,
					Assert{OpTerm{EquivOp, NewVar("a"), NewVar("a")}}),
				NaturalLit(3)),
			opValue{EquivOp, NaturalLit(3), NaturalLit(3)}),
	)
	DescribeTable("Others",
		typecheckTest,
		Entry(`3 : Natural`, NaturalLit(3), Natural),
		Entry(`[] : List Natural : List Natural`,
			EmptyList{Apply(List, Natural)}, AppValue{List, Natural}),
	)
	DescribeTable("Expected failures",
		func(t Term) {
			_, err := TypeOf(t)
			Ω(err).Should(HaveOccurred())
		},
		// Universe
		Entry(`Sort -- Sort has no type`,
			Sort),
		// EmptyList
		Entry(`[] : List 3 -- not a valid list type`,
			EmptyList{Apply(List, NaturalLit(3))}),
		Entry(`[] : Natural -- not in form "List a"`,
			EmptyList{Natural}),

		// AppTerm
		Entry(`Sort Type -- Fn of AppTerm doesn't typecheck`,
			Apply(Sort, Type)),
		Entry(`List Sort -- Arg of AppTerm doesn't typecheck`,
			Apply(List, Sort)),
		Entry(`List 3 -- Arg of AppTerm doesn't match function input type`,
			Apply(List, NaturalLit(3))),
		Entry(`Natural Natural -- Fn of AppTerm isn't of function type`,
			Apply(Natural, Natural)),
	)
})
