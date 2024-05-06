package typescript

import (
	"SubmissionGrader/internal/common"
	"SubmissionGrader/internal/complexity/complexCommons"
	methodInfoType "SubmissionGrader/internal/complexity/methodInfo"
	"SubmissionGrader/internal/complexity/scopeInfo"
	"SubmissionGrader/internal/complexity/stateInfo"
	parser "SubmissionGrader/internal/complexity/typescript/typeScriptAntlrParser"
	"fmt"
	//"github.com/antlr4-go/antlr/v4"
	"github.com/antlr/antlr4/runtime/Go/antlr/v4"
)

func CreateTypescriptComplexityParser() complexCommons.IComplexityParser {
	return &typescriptComplexityParser{
		fileRegex: ".ts/.tsx",
	}
}

type typescriptComplexityParser struct {
	fileRegex string
}

func (c typescriptComplexityParser) GetFileRegex() string {
	return c.fileRegex
}

func (c typescriptComplexityParser) ParseComplexityOfFile(filename string, includeTokens bool) []methodInfoType.MethodInfo {
	fileText, _ := common.GetTextOfFile(filename)

	input := antlr.NewInputStream(fileText)
	lexer := parser.NewTypeScriptLexer(input)

	stateStack := stateInfo.NewStateStack()
	currentState := stateInfo.NewStateObject()
	finalStack := methodInfoType.NewMethodStack()

	bracketCount := 0
	repeatForLoopForExtraCheck := true

	prevFoundToken := ""
	potentialClassOverride := ""
	scopeCurlyBracketCheck := false
	overrideScopePop := false
	immediatelyGoOutOfScope := false
	inTernaryOperatorScope := false
	inTernaryOperatorOriginalNesting := -1

	t, b := MoveToNextToken(lexer, includeTokens, currentState.CurrentMethodInfo, currentState.InMethod)
	if !b {
		common.Warning(fmt.Sprintf("File attempting to be parsed for complexity was empty: %s", filename))
		return finalStack.ConvertToArray()
	}
	textName := t.GetText()
	symbolicName := lexer.SymbolicNames[t.GetTokenType()]
	lastItemWasRecursive := false

	for {
		prevFoundToken = symbolicName
		if !repeatForLoopForExtraCheck {
			t, b = MoveToNextToken(lexer, includeTokens, currentState.CurrentMethodInfo, currentState.InMethod)
			if !b {
				return finalStack.ConvertToArray()
			}
		} else {
			repeatForLoopForExtraCheck = false
		}
		textName = t.GetText()
		symbolicName = lexer.SymbolicNames[t.GetTokenType()]

		if immediatelyGoOutOfScope {
			currentState.CurrentScope.ScopeBracketCount = bracketCount
			immediatelyGoOutOfScope = false
		} else if symbolicName == "LeftBrace" {
			bracketCount++
		} else if scopeCurlyBracketCheck && ((prevFoundToken != "Else" && symbolicName != "If") || prevFoundToken == "Do") {
			repeatForLoopForExtraCheck = true
		} else if symbolicName == "RightBrace" {
			bracketCount--
		} else if symbolicName == "AndAnd" && currentState.InMethod {
			currentState.CurrentMethodInfo.CycCount++
		} else if symbolicName == "OrOr" && currentState.InMethod {
			currentState.CurrentMethodInfo.CycCount++
		} else if symbolicName == "Class" {
			// We need to first ensure we actually have found a class
			t, b = MoveToNextToken(lexer, includeTokens, currentState.CurrentMethodInfo, currentState.InMethod) // gets to next identifier
			if !b {
				return finalStack.ConvertToArray()
			}
			tempSN := lexer.SymbolicNames[t.GetTokenType()]
			if tempSN == "Identifier" {
				potentialClassName := t.GetText()
				t, b = MoveToNextToken(lexer, includeTokens, currentState.CurrentMethodInfo, currentState.InMethod) // gets to next identifier
				if !b {
					return finalStack.ConvertToArray()
				}
				tempSN = lexer.SymbolicNames[t.GetTokenType()]
				if tempSN == "LeftBrace" { // Yay, it is a class
					if currentState.InClass {
						newLocation := currentState.Location + "->" + currentState.ClassName
						if currentState.InMethod {
							newLocation += "." + currentState.CurrentMethodInfo.MethodName
						}
						stateStack.Push(currentState)
						currentState = stateInfo.NewStateObjectWithLocation(newLocation)
					}

					currentState.InClass = true
					currentState.ClassBracketCount = bracketCount
					currentState.ClassName = potentialClassName
					bracketCount++
				} else {
					repeatForLoopForExtraCheck = true
				}
			} else {
				repeatForLoopForExtraCheck = true
			}
		} else if symbolicName == "Identifier" && !currentState.InMethod {
			potentialMethodName := t.GetText()
			potentialParameter := ""
			t, b = MoveToNextToken(lexer, includeTokens, currentState.CurrentMethodInfo, currentState.InMethod)
			if !b {
				return finalStack.ConvertToArray()
			}
			tempSN := lexer.SymbolicNames[t.GetTokenType()]
			if tempSN == "LeftParen" { // SHOULD FIND (
				potentialCycCount, content := SkipParenthesisGetInside(lexer)
				potentialParameter = content
				if potentialCycCount == -1 {
					return finalStack.ConvertToArray()
				}
				t, b = MoveToNextToken(lexer, includeTokens, currentState.CurrentMethodInfo, currentState.InMethod)
				if !b {
					return finalStack.ConvertToArray()
				}
				tempSN = lexer.SymbolicNames[t.GetTokenType()]
				if tempSN == "LeftBrace" { // SHOULD FIND {
					// METHOD FOUND!
					if prevFoundToken == "Tilde" {
						potentialMethodName = "~" + potentialMethodName
					}
					tempClassName := currentState.ClassName
					if potentialClassOverride != "" {
						tempClassName = potentialClassOverride
						potentialClassOverride = ""
					}

					newMethod := methodInfoType.MethodInfo{
						Location:           "",
						Class:              tempClassName,
						MethodName:         potentialMethodName,
						Parameter:          potentialParameter,
						StartLine:          t.GetLine(),
						EndLine:            -1,
						TotalLine:          -1,
						LinesOfCodeCreated: -1,
						CogCount:           1,
						CycCount:           1,
					}
					currentState.CurrentMethodInfo = &newMethod
					currentState.MethodBracketCount = bracketCount
					currentState.InMethod = true
					currentState.NestingCount = 0

					bracketCount++
				}
			} else if tempSN == "DoubleColon" || tempSN == "Less" {
				potentialClassOverride = textName
			} else if tempSN == "LeftBrace" {
				repeatForLoopForExtraCheck = true
			} else if tempSN == "Identifier" {
				repeatForLoopForExtraCheck = true
			}
		} else if symbolicName == "If" && currentState.InMethod { // NESTING MATTERS
			currentState.LastScopeKeyWord = "If"
			currentState.IncCycCount(1)
			if prevFoundToken != "Else" {
				currentState.AddToComplexitiesWithNesting(0, 1)
			}
			potentialCycloCount := FindAndSkipParenthesis(lexer, lexer.NextToken()) // There should be a set of parenthesis following IF
			if potentialCycloCount == -1 {
				return finalStack.ConvertToArray()
			}
			currentState.IncCycCount(potentialCycloCount)
			scopeCurlyBracketCheck = true
			continue
		} else if symbolicName == "Else" && currentState.InMethod { // NESTING MATTERS
			currentState.LastScopeKeyWord = "Else"
			currentState.AddToComplexitiesAndIncreaseNesting(0, 1)
			scopeCurlyBracketCheck = true
			continue
		} else if symbolicName == "For" && currentState.InMethod { // NESTING MATTERS
			worked := AddItemAsScoped(&currentState, lexer, true, &scopeCurlyBracketCheck, &prevFoundToken, symbolicName, 1, 1)
			if !worked {
				return finalStack.ConvertToArray()
			}
			continue
		} else if symbolicName == "While" && currentState.InMethod { // NESTING MATTERS
			worked := AddItemAsScoped(&currentState, lexer, true, &scopeCurlyBracketCheck, &prevFoundToken, symbolicName, 1, 1)
			if !worked {
				return finalStack.ConvertToArray()
			}
			continue
		} else if symbolicName == "Do" && currentState.InMethod { // NESTING MATTERS
			currentState.LastScopeKeyWord = "Do"
			worked := AddItemAsScoped(&currentState, lexer, false, &scopeCurlyBracketCheck, &prevFoundToken, symbolicName, 1, 1)
			if !worked {
				return finalStack.ConvertToArray()
			}
			continue
		} else if symbolicName == "Switch" && currentState.InMethod { // NESTING MATTERS
			worked := AddItemAsScoped(&currentState, lexer, true, &scopeCurlyBracketCheck, &prevFoundToken, symbolicName, 0, 1)
			if !worked {
				return finalStack.ConvertToArray()
			}
			continue
		} else if symbolicName == "Case" && currentState.InMethod {
			currentState.IncCycCount(1)
		} else if symbolicName == "Default" && currentState.InMethod {
			currentState.IncCycCount(1)
		} else if symbolicName == "Break" && currentState.InMethod {

		} else if symbolicName == "Question" && currentState.InMethod { // This is for Ternary Operators
			if !inTernaryOperatorScope {
				inTernaryOperatorScope = true
				inTernaryOperatorOriginalNesting = currentState.NestingCount
			}
			currentState.AddToComplexitiesWithNesting(1, 1)

		} else if symbolicName == "LeftBracket" && currentState.InMethod { // [
			MoveToNextSpecifiedToken(lexer, "RightBracket", includeTokens, currentState.CurrentMethodInfo, currentState.InMethod)
			t = lexer.NextToken()
			tempSN := lexer.SymbolicNames[t.GetTokenType()]
			if tempSN == "LeftParen" {
				SkipParenthesis(lexer)
				t = lexer.NextToken()
				tempSN = lexer.SymbolicNames[t.GetTokenType()]
			}
			if tempSN == "LeftBrace" {
				currentState.LastScopeKeyWord = "Lambda_Function"
				symbolicName = "LeftBrace"
				bracketCount++
				currentState.AddToComplexitiesAndIncreaseNesting(1, 1)
				scopeCurlyBracketCheck = true
			}
		} else if symbolicName == "RightBracket" && currentState.InMethod { // ]

		} else if symbolicName == "Try" && currentState.InMethod {
			currentState.LastScopeKeyWord = "Try"
			scopeCurlyBracketCheck = true
			continue
		} else if symbolicName == "Catch" && currentState.InMethod { // NESTING MATTERS
			worked := AddItemAsScoped(&currentState, lexer, true, &scopeCurlyBracketCheck, &prevFoundToken, symbolicName, 1, 1)
			if !worked {
				return finalStack.ConvertToArray()
			}
			continue
		} else if symbolicName == "Throw" && currentState.InMethod {
			currentState.IncCycCount(1)
		} else if symbolicName == "Semi" && currentState.InMethod {
			if inTernaryOperatorScope {
				inTernaryOperatorScope = false
				currentState.NestingCount = inTernaryOperatorOriginalNesting
			}
			if currentState.InScope {
				potentialClassOverride = ""
				if currentState.CurrentScope.DependentScope {
					currentState.CurrentScope.ScopeBracketCount = bracketCount
				}
			}
		} else if symbolicName == "DoubleColon" && currentState.InMethod { // ::

		} else if symbolicName == "Less" { // "<"

		} else if symbolicName == "Greater" { // ">"

		} else if symbolicName == "LeftParen" { // "("
			if lastItemWasRecursive {
				currentState.IncCogCount(1)
			}
		} else if symbolicName == "RightParen" { // ")"

		} else if currentState.InMethod {
			if textName == currentState.CurrentMethodInfo.MethodName {
				lastItemWasRecursive = true
				continue
			}
		}
		lastItemWasRecursive = false

		if scopeCurlyBracketCheck { // Creates scope for item
			if currentState.InScope {
				currentState.CurrentScopeStack.Push(*currentState.CurrentScope)
			}

			deScope := false

			if symbolicName == "LeftBrace" {
				currentState.CurrentScope = scopeInfo.NewScopeInfo(false, bracketCount-1, currentState.LastScopeKeyWord, false)
			} else {

				if currentState.LastScopeKeyWord == "Do" {
					deScope = true
				}
				currentState.CurrentScope = scopeInfo.NewScopeInfo(true, -1, currentState.LastScopeKeyWord, deScope)
			}
			currentState.InScope = true
			scopeCurlyBracketCheck = false
		}

		if bracketCount == currentState.CurrentScope.ScopeBracketCount { // this Scope is over
			t, b = MoveToNextToken(lexer, includeTokens, currentState.CurrentMethodInfo, currentState.InMethod)
			if !b {
				return finalStack.ConvertToArray()
			}
			repeatForLoopForExtraCheck = true
			symbolicNameTemp := lexer.SymbolicNames[t.GetTokenType()]
			if currentState.CurrentScope.ScopeKeyword == "If" {
				if symbolicNameTemp == "Else" {
					overrideScopePop = true
				}
			} else if currentState.CurrentScope.ScopeKeyword == "Do" {
				potentialCycloCount := FindAndSkipParenthesis(lexer, t) // There should be a set of parenthesis following WHILE (if not do while)
				if potentialCycloCount == -1 {
					return finalStack.ConvertToArray()
				} else {
					currentState.IncCycCount(potentialCycloCount)
				}
				t, b = MoveToNextToken(lexer, includeTokens, currentState.CurrentMethodInfo, currentState.InMethod)
				if !b {
					return finalStack.ConvertToArray()
				}
			} else if currentState.CurrentScope.ScopeKeyword == "Try" {
				overrideScopePop = true
				currentState.NestingCount++
			}

			currentState.NestingCount--
			if !currentState.CurrentScopeStack.Empty() {
				poppingResult := -2 // SHOULD NEVER END UP WITH -2 AS FINAL RESULT
				if !overrideScopePop {
					poppingResult = currentState.PopScopeStackUntilNonDependentScope()
				} else {
					overrideScopePop = false
					poppingResult = 1
				}

				if poppingResult == -1 { // ERROR
					return finalStack.ConvertToArray()
				} else if poppingResult == 0 { // Empty stack
					currentState.FlushScope()
				} else if poppingResult == 1 || poppingResult == 2 { // Grab the previous item now
					prevScope, err := currentState.CurrentScopeStack.Front()
					if err != nil {
						common.Error(fmt.Sprintf("Failed to get previous scope: %s\n", err))
						return finalStack.ConvertToArray()
					}
					currentState.CurrentScope = &prevScope
					err = currentState.CurrentScopeStack.Pop()
					if err != nil {
						common.Error(fmt.Sprintf("Failed to pop previous scope: %s\n", err))
						return finalStack.ConvertToArray()
					}
					if poppingResult == 2 {
						immediatelyGoOutOfScope = true // THE IMPORTANT DIFFERENT
					}
				}
			} else {
				currentState.FlushScope()
			}
		} else if bracketCount == currentState.MethodBracketCount { // this method is over
			// Finishes up adding necessary items to method info object and puts it on stack
			methodEndLine := t.GetLine()
			b := complexCommons.FinishMethod(&currentState, finalStack, methodEndLine)
			if !b {
				return finalStack.ConvertToArray()
			}
			potentialClassOverride = ""
		} else if bracketCount == currentState.ClassBracketCount { // this Class is over
			b := complexCommons.RestorePreviousState(&currentState, stateStack)
			if !b {
				return finalStack.ConvertToArray()
			}
		}
	}
}
