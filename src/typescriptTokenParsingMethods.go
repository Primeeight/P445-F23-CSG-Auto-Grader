package typescript

import (
	methodInfoType "SubmissionGrader/internal/complexity/methodInfo"
	"SubmissionGrader/internal/complexity/stateInfo"
	parser "SubmissionGrader/internal/complexity/typescript/typeScriptAntlrParser"
	//"github.com/antlr4-go/antlr/v4"
	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
)

// FindAndSkipParenthesis
// This method will begin looking for the left parenthesis
// and then go about the process of getting to the corresponding right
// parenthesis.
// The integer that is returned represents the cyclomatic count
// as the AND (&&) symbol increases it and will potentially be
// found inside a set of parenthesis
func FindAndSkipParenthesis(lexer *parser.TypeScriptLexer, t antlr.Token) int {
	for {
		if t.GetTokenType() == antlr.TokenEOF {
			return -1 // if it returns false, it was at the end of the file which would mean we should just return
		} else if lexer.SymbolicNames[t.GetTokenType()] == "LeftParen" {
			return SkipParenthesis(lexer)
		}
		t = lexer.NextToken()
	}
}

func SkipParenthesis(lexer *parser.TypeScriptLexer) int {
	cycCount := 0
	parenCount := 1
	for parenCount != 0 {
		t := lexer.NextToken()
		if t.GetTokenType() == antlr.TokenEOF {
			return -1
		} else if lexer.SymbolicNames[t.GetTokenType()] == "AndAnd" {
			cycCount++
		} else if lexer.SymbolicNames[t.GetTokenType()] == "OrOr" {
			cycCount++
		} else if lexer.SymbolicNames[t.GetTokenType()] == "RightParen" {
			parenCount--
		} else if lexer.SymbolicNames[t.GetTokenType()] == "LeftParen" {
			parenCount++
		}
	}
	return cycCount
}

func SkipParenthesisGetInside(lexer *parser.TypeScriptLexer) (int, string) {
	cycCount := 0
	parenCount := 1
	content := ""

	for parenCount != 0 {
		t := lexer.NextToken()
		if t.GetTokenType() == antlr.TokenEOF {
			return -1, ""
		} else {
			symbolicName := lexer.SymbolicNames[t.GetTokenType()]
			if symbolicName != "RightParen" {
				content += t.GetText() + " "
			}

			if lexer.SymbolicNames[t.GetTokenType()] == "AndAnd" {
				cycCount++
			} else if lexer.SymbolicNames[t.GetTokenType()] == "OrOr" {
				cycCount++
			} else if lexer.SymbolicNames[t.GetTokenType()] == "RightParen" {
				parenCount--
			} else if lexer.SymbolicNames[t.GetTokenType()] == "LeftParen" {
				parenCount++
			}
		}
	}
	return cycCount, content
}

// MoveToNextToken
// This moves the token forward while also ensure we haven't got to the end of the file
// If it did get to the end of the file, it returns false
func MoveToNextToken(lexer *parser.TypeScriptLexer, includeToken bool, m *methodInfoType.MethodInfo, inMethod bool) (antlr.Token, bool) {
	t := lexer.NextToken()
	if inMethod && includeToken && t.GetTokenType() != antlr.TokenEOF {
		m.AddTokenToMethod(t.GetLine(), -1, lexer.SymbolicNames[t.GetTokenType()], lexer.RuleNames[t.GetTokenType()], t.GetText())
	}
	if t.GetTokenType() == antlr.TokenEOF {
		return nil, false
	}
	return t, true
}

// MoveToNextSpecifiedToken
// This continues parsing tokens until it finds a given token type
func MoveToNextSpecifiedToken(lexer *parser.TypeScriptLexer, symbolicNameLook string, includeToken bool, m *methodInfoType.MethodInfo, inMethod bool) (antlr.Token, bool) {
	for {
		t := lexer.NextToken()
		if inMethod && includeToken && t.GetTokenType() != antlr.TokenEOF {
			m.AddTokenToMethod(t.GetLine(), -1, lexer.SymbolicNames[t.GetTokenType()], lexer.RuleNames[t.GetTokenType()], t.GetText())
		}
		if t.GetTokenType() == antlr.TokenEOF {
			return nil, false
		}
		symbolicName := lexer.SymbolicNames[t.GetTokenType()]
		if symbolicName == symbolicNameLook {
			return t, true
		}
	}
}

// AddItemAsScoped
// If an items needs to record its scope, it needs to do the steps outlined here
// (or more, depending on if it's IF, for instance)
// What this does is essentially change the variables all needed when the scope
// is added later on in the code.
// Also, most words which are scoped have parenthesis immediately after it which need to be skipped, so that happens here
func AddItemAsScoped(currentState *stateInfo.StateInfo, lexer *parser.TypeScriptLexer, skipParenthesis bool, scopeCurlyBracketCheck *bool, prevFoundKeyword *string, symbolicName string, cycCount int, cogCount int) bool {
	currentState.LastScopeKeyWord = symbolicName
	currentState.AddToComplexitiesWithNesting(cycCount, cogCount)
	if skipParenthesis {
		potentialCycloCount := FindAndSkipParenthesis(lexer, lexer.NextToken())
		if potentialCycloCount == -1 {
			return false
		} else {
			currentState.IncCycCount(potentialCycloCount)
		}
	}
	*scopeCurlyBracketCheck = true
	*prevFoundKeyword = symbolicName
	return true
}
