package parserTypes

import (
	"SubmissionGrader/internal/parser/parserTypes/list"
	"encoding/xml"
)

// Create a struct for use in the parser.
type rParser struct{}

// Constructor
func NewRparser() IParser { return &rParser{} }

// Coverage
func (t rParser) CoverageParser(location string) (CoverageResultsRawType, error) {
	coverageResults := CoverageResultsRawType{
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
	//Parse coverage results
	return coverageResults, nil
}

// Actual file parsiing
// Performs parsing on a file converted into a byte-array
func (t rParser) FileParse(bytes []byte) (list.TestsLinkedList, error) {
	var query RQuery
	listResults := list.TestsLinkedList{}

	err := xml.Unmarshal(bytes, &query.Chan)
	if err != nil {
		return list.TestsLinkedList{}, err
	} else {
		queriedObject := query.Chan

		listResults = queriedObject.ListFormat()

	}
	return listResults, nil
}

type RQuery struct {
	Chan RSuites `xml:"testsuites"`
}

type RSuites struct {
	UnitTests []RUnitTests `xml:"testsuite"`
}

type RUnitTests struct {
	Errors    string      `xml:"errors,attr"`
	Failures  string      `xml:"failures,attr"`
	Skipped   string      `xml:"skipped,attr"`
	Tests     string      `xml:"tests,attr"`
	TestCases []RTestCase `xml:"testcase"`
}

type RTestCase struct {
	ClassName string         `xml:"classname,attr"`
	Name      string         `xml:"name,attr"`
	Skip      RSkipStatus    `xml:"skipped"`
	Error     RErrorStatus   `xml:"error"`
	Fail      RFailureStatus `xml:"failure"`
}

type RFailureStatus struct {
	Message string `xml:"message,attr"`
}

type RSkipStatus struct {
	Message string `xml:"message,attr"`
}

type RErrorStatus struct {
	Message string `xml:"message,attr"`
}

func (s RSuites) ListFormat() list.TestsLinkedList {

	listResults := list.TestsLinkedList{}

	// Gets Suites
	for _, testSuite := range s.UnitTests {
		for _, testCase := range testSuite.TestCases {
			thisTest := testCase.TestFormat()
			listResults.AddTest(testCase.ClassName, thisTest)
		}
	}

	return listResults
}

func (s RTestCase) TestFormat() list.UnitTest {

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
