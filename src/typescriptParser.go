package parserTypes

import (
	"SubmissionGrader/internal/parser/parserTypes/list"
	"encoding/xml"
	"os"
	"strconv"
)

// PYTEST
// in pytest, to get give test result, do:
// pytest -v --junitxml=out.xml
//after
// python -m pip install pytest
//
// MAKE SURE THE VERSION OF PYTHON WE ARE USING IN ECS CONTAINER HAS PYTEST INSTALLED

type typescriptParser struct{}

func NewTypescriptParser() IParser {
	return &typescriptParser{}
}

type TypescriptCoverage struct {
	Chan TypescriptCoverageTotal `xml:"coverage"`
}

type TypescriptCoverageTotal struct {
	LinesValid      string `xml:"lines-valid,attr"`
	LinesCovered    string `xml:"lines-covered,attr"`
	BranchesValid   string `xml:"branches-valid,attr"`
	BranchesCovered string `xml:"branches-covered,attr"`
}

// CoverageParser
// parses the output from what is covered by testing in the assignment
func (t typescriptParser) CoverageParser(location string) (CoverageResultsRawType, error) {
	returnItem := CoverageResultsRawType{
		MissedLines:         0,
		CoveredLines:        0,
		MissedFunctions:     -1,
		CoveredFunctions:    -1,
		MissedBranches:      0,
		CoveredBranches:     0,
		MissedInstructions:  -1,
		CoveredInstructions: -1,
		MissedComplexity:    -1,
		CoveredComplexity:   -1,
	}

	content, err := os.ReadFile(location) // Gets contents of file
	if err != nil {                       // Error in reading file
		return CoverageResultsRawType{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, err
	}

	var bytes = []byte(string(content)) // converts to bytes
	var query TypescriptCoverage
	err = xml.Unmarshal(bytes, &query.Chan)
	if err != nil {
		return CoverageResultsRawType{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, err
	} else {
		queriedObject := query.Chan
		AllLines, _ := strconv.Atoi(queriedObject.LinesValid)
		CoveredLines, _ := strconv.Atoi(queriedObject.LinesCovered)
		MissedLines := AllLines - CoveredLines
		AllBranches, _ := strconv.Atoi(queriedObject.BranchesValid)
		CoveredBranches, _ := strconv.Atoi(queriedObject.BranchesCovered)
		MissedBranches := AllBranches - CoveredBranches

		returnItem.CoveredLines = CoveredLines
		returnItem.MissedLines = MissedLines
		returnItem.CoveredBranches = CoveredBranches
		returnItem.MissedBranches = MissedBranches
	}

	return returnItem, nil
}

// FileParse
//
//	This method parses one file that has been converted to a byte array
func (t typescriptParser) FileParse(bytes []byte) (list.TestsLinkedList, error) {
	var query TypescriptQuery
	resultList := list.TestsLinkedList{}

	err := xml.Unmarshal(bytes, &query.Chan)
	if err != nil {
		return list.TestsLinkedList{}, err
	} else {
		queriedObject := query.Chan

		resultList = queriedObject.ListFormat()

	}

	return resultList, nil
}

type TypescriptQuery struct {
	Chan TypescriptSuites `xml:"testsuites"`
}

type TypescriptSuites struct {
	UnitTests []TypescriptUnitTests `xml:"testsuite"`
}

type TypescriptUnitTests struct {
	Errors    string               `xml:"errors,attr"`
	Failures  string               `xml:"failures,attr"`
	Skipped   string               `xml:"skipped,attr"`
	Tests     string               `xml:"tests,attr"`
	TestCases []TypescriptTestCase `xml:"testcase"`
}

type TypescriptTestCase struct {
	ClassName string                  `xml:"classname,attr"`
	Name      string                  `xml:"name,attr"`
	Skip      TypescriptSkipStatus    `xml:"skipped"`
	Error     TypescriptErrorStatus   `xml:"error"`
	Fail      TypescriptFailureStatus `xml:"failure"`
}

type TypescriptFailureStatus struct {
	Message string `xml:"message,attr"`
}

type TypescriptSkipStatus struct {
	Message string `xml:"message,attr"`
}

type TypescriptErrorStatus struct {
	Message string `xml:"message,attr"`
}

func (s TypescriptSuites) ListFormat() list.TestsLinkedList {

	returnList := list.TestsLinkedList{}

	// Gets Suites
	for _, testSuite := range s.UnitTests {
		for _, testCase := range testSuite.TestCases {
			thisTest := testCase.TestFormat()
			returnList.AddTest(testCase.ClassName, thisTest)
		}
	}

	return returnList
}

func (s TypescriptTestCase) TestFormat() list.UnitTest {

	thisTest := list.UnitTest{}

	passFailStatus := "PASSED"

	failMessage := ""
	failMessage = s.Fail.Message
	skipMessage := ""
	skipMessage = s.Skip.Message
	errorMessage := ""
	errorMessage = s.Error.Message

	if failMessage != "" {
		passFailStatus = "FAILED"
		thisTest.Message = failMessage
	} else if skipMessage != "" {
		passFailStatus = "SKIPPED"
	} else if errorMessage != "" {
		passFailStatus = "ERRORED"
	} else {
		thisTest.Message = ""
	}

	thisTest.Outcome = passFailStatus
	thisTest.Name = s.Name

	return thisTest
}
