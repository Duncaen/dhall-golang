package main_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/philandstuff/dhall-golang/ast"
	"github.com/philandstuff/dhall-golang/parser"
	"github.com/ugorji/go/codec"
)

var expectedFailures = []string{
	"TestParserAccepts/annotationsA.dhall", // requires records, list append, optionals
	"TestParserAccepts/asTextA.dhall",
	"TestParserAccepts/builtinsA.dhall",
	"TestParserAccepts/collectionImportTypeA.dhall",
	"TestParserAccepts/constructorsA.dhall",
	// hopefully dhall-lang/dhall-lang#372 will fix this
	// although I suspect floating-point precision may still bite
	"TestParserAccepts/doubleA.dhall",
	"TestParserAccepts/doubleQuotedStringA.dhall",
	"TestParserAccepts/environmentVariablesA.dhall",
	"TestParserAccepts/escapedDoubleQuotedStringA.dhall",
	"TestParserAccepts/escapedSingleQuotedStringA.dhall",
	"TestParserAccepts/fieldsA.dhall",
	"TestParserAccepts/importAltA.dhall",
	"TestParserAccepts/interpolatedDoubleQuotedStringA.dhall",
	"TestParserAccepts/interpolatedSingleQuotedStringA.dhall",
	"TestParserAccepts/largeExpressionA.dhall",
	"TestParserAccepts/mergeA.dhall",
	"TestParserAccepts/operatorsA.dhall",
	"TestParserAccepts/parenthesizeUsingA.dhall",
	"TestParserAccepts/pathTerminationA.dhall",
	"TestParserAccepts/pathsA.dhall",
	"TestParserAccepts/quotedLabelA.dhall",
	"TestParserAccepts/quotedPathsA.dhall",
	"TestParserAccepts/recordA.dhall",
	"TestParserAccepts/singleQuotedStringA.dhall",
	"TestParserAccepts/templateA.dhall",
	"TestParserAccepts/unicodePathsA.dhall",
	"TestParserAccepts/unicodeDoubleQuotedStringA.dhall",
	"TestParserAccepts/unionA.dhall",
	"TestParserAccepts/urlsA.dhall",
	"TestTypecheckFails/combineMixedRecords.dhall",
	"TestTypecheckFails/duplicateFields.dhall",
	"TestTypecheckFails/mixedUnions.dhall",
	"TestTypecheckFails/preferMixedRecords.dhall",
}

func pass(t *testing.T) {
	for _, name := range expectedFailures {
		if t.Name() == name {
			t.Error("Expected failure, but actually passed")
		}
	}
}

func failf(t *testing.T, format string, args ...interface{}) {
	for _, name := range expectedFailures {
		if t.Name() == name {
			t.Skipf(format, args...)
			return
		}
	}
	t.Fatalf(format, args...)
}

func expectError(t *testing.T, err error) {
	if err == nil {
		failf(t, "Expected file to fail to parse, but it parsed successfully")
	}
}

func expectNoError(t *testing.T, err error) {
	if err != nil {
		failf(t, "Expected file to parse successfully, but got error %v", err)
	}
}

func expectEqual(t *testing.T, expected, actual interface{}) {
	if reflect.DeepEqual(expected, actual) {
		pass(t)
	} else {
		failf(t, "Expected %+v to equal %+v", actual, expected)
	}
}

func runTestOnEachFile(t *testing.T, dir string, test func(*testing.T, io.Reader)) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatalf("Couldn't read dhall-lang tests: %v\n(Have you pulled submodules?)\n", err)
	}

	for _, f := range files {
		t.Run(f.Name(), func(t *testing.T) {
			reader, openerr := os.Open(dir + f.Name())
			defer reader.Close()
			if openerr != nil {
				t.Fatal(openerr)
			}

			test(t, reader)
		})
	}
}

func TestParserRejects(t *testing.T) {
	runTestOnEachFile(t, "./dhall-lang/tests/parser/failure/", func(t *testing.T, reader io.Reader) {
		_, err := parser.ParseReader(t.Name(), reader)

		expectError(t, err)
	})
}

func TestParserAccepts(t *testing.T) {
	successesDir := "./dhall-lang/tests/parser/success/"
	files, err := filepath.Glob(successesDir + "*A.dhall")
	if err != nil {
		t.Fatalf("Couldn't read dhall-lang tests: %v\n(Have you pulled submodules?)\n", err)
	}
	var cbor codec.CborHandle
	var json codec.JsonHandle

	for _, aName := range files {
		name := filepath.Base(aName)
		bName := strings.Replace(aName, "A.dhall", "B.json", 1)
		aReader, err := os.Open(aName)
		defer aReader.Close()
		if err != nil {
			t.Fatal(err)
		}
		bReader, err := os.Open(bName)
		defer bReader.Close()
		if err != nil {
			t.Fatal(err)
		}
		t.Run(name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			parsed, err := parser.ParseReader(name, aReader)
			expectNoError(t, err)
			aEnc := codec.NewEncoder(buf, &cbor)
			err = aEnc.Encode(parsed)
			expectNoError(t, err)
			aDec := codec.NewDecoder(buf, &cbor)
			var actual interface{}
			err = aDec.Decode(&actual)
			expectNoError(t, err)

			bDec := codec.NewDecoder(bReader, &json)
			var expected interface{}
			err = bDec.Decode(&expected)
			expectNoError(t, err)
			expectEqual(t, expected, actual)
		})
	}
}

func TestTypecheckFails(t *testing.T) {
	runTestOnEachFile(t, "./dhall-lang/tests/typecheck/failure/", func(t *testing.T, reader io.Reader) {
		parsed, err := parser.ParseReader(t.Name(), reader)

		expectNoError(t, err)

		expr, ok := parsed.(ast.Expr)
		if !ok {
			failf(t, "Expected ast.Expr, got %+v\n", parsed)
		}

		_, err = expr.TypeWith(ast.EmptyContext())

		expectError(t, err)
	})
}
