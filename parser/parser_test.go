package parser

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
}

func (suite *TestSuite) TestParser() {
	for _, tc := range []struct {
		name      string
		inputPath string
		expected  int
	}{
		{
			name:      "arithmetic",
			inputPath: "../testdata/arithmetic.test",
			expected:  0,
		},
		{
			name:      "variable",
			inputPath: "../testdata/assignment.test",
			expected:  0,
		},
		{
			name:      "if",
			inputPath: "../testdata/if.test",
			expected:  0,
		},
		{
			name:      "while",
			inputPath: "../testdata/while.test",
			expected:  0,
		},
		{
			name:      "print",
			inputPath: "../testdata/print.test",
			expected:  0,
		},
		{
			name:      "operators",
			inputPath: "../testdata/operators.test",
			expected:  0,
		},
		{
			name:      "functions",
			inputPath: "../testdata/functions.test",
			expected:  0,
		},
	} {
		suite.T().Run(tc.name, func(t *testing.T) {
			file, err := os.Open(tc.inputPath)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()

			lexer := NewLexer(file)
			output := YYParse(lexer)
			suite.Assert().Equal(tc.expected, output)
			suite.Assert().Empty(lexer.Errors())

			root, ok := lexer.Root().(*Block)
			suite.Assert().True(ok, "root AST should be a *Block")
			suite.Assert().NotEmpty(root.Stmts)
		})
	}
}

func TestParser(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
