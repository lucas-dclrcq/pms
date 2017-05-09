package lexer_test

import (
	"testing"

	"github.com/ambientsound/pms/input/lexer"
	"github.com/stretchr/testify/assert"
)

var lexerTests = []struct {
	input    string
	expected []result
}{
	{
		"we  shall\t te|st white\"space and $quoting\"; and # comments",
		[]result{
			{class: lexer.TokenIdentifier, str: "we"},
			{class: lexer.TokenIdentifier, str: "shall"},
			{class: lexer.TokenIdentifier, str: "te"},
			{class: lexer.TokenSeparator, str: "|"},
			{class: lexer.TokenIdentifier, str: "st"},
			{class: lexer.TokenIdentifier, str: "whitespace and $quoting"},
			{class: lexer.TokenStop, str: ";"},
			{class: lexer.TokenIdentifier, str: "and"},
			{class: lexer.TokenComment, str: "# comments"},
			{class: lexer.TokenEnd, str: ""},
		},
	},
	{
		"$variables are {nice }}, ar{}en't $they?",
		[]result{
			{class: lexer.TokenVariable, str: "$"},
			{class: lexer.TokenIdentifier, str: "variables"},
			{class: lexer.TokenIdentifier, str: "are"},
			{class: lexer.TokenOpen, str: "{"},
			{class: lexer.TokenIdentifier, str: "nice"},
			{class: lexer.TokenClose, str: "}"},
			{class: lexer.TokenClose, str: "}"},
			{class: lexer.TokenIdentifier, str: ","},
			{class: lexer.TokenIdentifier, str: "ar"},
			{class: lexer.TokenOpen, str: "{"},
			{class: lexer.TokenClose, str: "}"},
			{class: lexer.TokenIdentifier, str: "en't"},
			{class: lexer.TokenVariable, str: "$"},
			{class: lexer.TokenIdentifier, str: "they?"},
			{class: lexer.TokenEnd, str: ""},
		},
	},
	{
		"$1$2 \"unter minated",
		[]result{
			{class: lexer.TokenVariable, str: "$"},
			{class: lexer.TokenIdentifier, str: "1"},
			{class: lexer.TokenVariable, str: "$"},
			{class: lexer.TokenIdentifier, str: "2"},
			{class: lexer.TokenIdentifier, str: "unter minated"},
			{class: lexer.TokenEnd, str: ""},
		},
	},
	/*
		{
			`$"quoted variable" ok`,
			[]result{
				{class: lexer.TokenVariable, str: "$"},
				{class: lexer.TokenIdentifier, str: "quoted variable"},
				{class: lexer.TokenIdentifier, str: "ok"},
				{class: lexer.TokenEnd, str: ""},
			},
		},
	*/
}

// TestLexer tests the lexer.NextToken() function, checking that it correctly
// splits up input lines into Token structs.
func TestLexer(t *testing.T) {
	var token lexer.Token

	for _, test := range lexerTests {

		i := 0
		pos := 0

		for {

			if i == len(test.expected) {
				if token.Class == lexer.TokenEnd {
					break
				}
				t.Fatalf("Tokenizer generated too many tokens!")
			}

			check := test.expected[i]
			token, npos := lexer.NextToken(test.input[pos:])
			pos += npos
			str := token.String()

			t.Logf("Token %d: pos=%d, runes='%s', input='%s'", i, pos, str, test.input)

			assert.Equal(t, token.Class, check.class,
				"Token class for token %d is wrong; expected %d but got %d", i, check.class, token.Class)
			assert.Equal(t, check.str, str,
				"String check against token %d failed; expected '%s' but got '%s'", i, check.str, str)

			i++
		}
	}
}
