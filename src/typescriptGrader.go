package graderFactory

import (
	"SubmissionGrader/internal/common"
	parserFactory "SubmissionGrader/internal/parser"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type typescriptGrader struct {
	grader graderStruct
}

func (t typescriptGrader) GetGrader() graderStruct {
	return t.grader
}

func (t *typescriptGrader) GradeAssignment(grader graderStruct) error {
	common.Info(fmt.Sprintf("Grading students test cases"))

	if grader.data.studentTestsEnabled == "true" {
		t.grader.data.GradingStudentTestCurrently = true

		pathToPotentialTeacherTests := grader.GetLocation() + "/src/test/typescript/teacher"
		if common.CheckIfDirExist(pathToPotentialTeacherTests) {
			common.Info("Teacher test folder before student test did exist and is being deleted")
			common.RemoveDir(pathToPotentialTeacherTests)
		} else {
			common.Info("Teacher test folder before student test did not exist and is moving on")
		}

		err := t.gradeSteps()
		if err != nil {
			common.Error(fmt.Sprintf("Error in running student tests: %s", err.Error()))
		}
		t.grader.data.GradingStudentTestCurrently = false
	}

	var err error

	if grader.data.teacherUnitTestsEnabled == "true" {
		if grader.data.studentTestsEnabled == "true" {
			common.Debug(fmt.Sprintf("Removing Results from student tests"))
			common.RemoveDir(grader.GetLocation() + "/build")
		}

		common.RemoveEverythingInThisDirectory(grader.GetLocation()+"/src/test/typeset", "") //REMOVES STUDENT TEST FILES

		subdirectoryToGet := "src/test/typescript/teacher"
		subdirectoryPlacementName := "teacher"
		if grader.data.UseOriginalStudentTestsInsteadOfTeacherTests == "true" {
			subdirectoryToGet = "src/test/typescript/student"
			subdirectoryPlacementName = "student"
		}

		common.Info(fmt.Sprintf("Getting teacher tests"))
		err = grader.GetTemplateSubDirectory(grader, grader.GetLocation()+"/src/test/typescript", subdirectoryToGet, subdirectoryPlacementName) //Comment out when GRADE THINGS

		if err != nil {
			return err
		}
		common.Info(fmt.Sprintf("Grading teacher test cases"))
		err = t.gradeSteps()
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *typescriptGrader) gradeSteps() error {

	common.Debug(fmt.Sprintf("Building and Grading Assignment"))
	err := t.GradeTests(t.GetGrader())
	if err != nil {
		/*if err.Error() != "exit status 1" { // this error probably means the tests just failed to pass
			common.Error(fmt.Sprintf("Error executing commands to gradle test the assignment: %s", err))
			return err
		}*/
		common.Warning(fmt.Sprintf("Grading Assignment Produced Error, although, this error means it just failed some tests: %s", err))
	}
	common.Debug(fmt.Sprintf("Sucessfully Tested Assignment"))

	common.Debug(fmt.Sprintf("Creating Report from result of tests"))
	err = t.GetUnitTestReport(t.GetGrader())
	/*if err != nil {
		common.Error(fmt.Sprintf("Error in getting results from testing project: %s", err))
		return err
	}*/
	common.Debug(fmt.Sprintf("Got results from test"))
	return nil
}

func (t *typescriptGrader) GetUnitTestReport(grader graderStruct) error {
	repoName := grader.GetLocation() + grader.data.submissionTestPath
	common.Debug(fmt.Sprintf("Getting parser factory object"))
	parser, err := parserFactory.GetParser("typescript", repoName, grader.GetLocation(), !t.grader.data.FailedToGetCoverage)
	if err != nil {
		common.Error(fmt.Sprintf("Failed to create parser: %s", err))
		return err
	}

	common.Debug(fmt.Sprintf("Parsing Results"))
	err = parser.ParseTestResults()
	if err != nil {
		errorMsg := fmt.Sprintf("Error parsing results: %s", err)
		common.Error(errorMsg)
		if strings.Contains(errorMsg, "build/test-results/test/: no such file or directory") {
			t.grader.data.FailedToCompile = true
		}

		return err
	}

	if t.grader.data.GradingStudentTestCurrently {
		common.Debug(fmt.Sprintf("Student Test Results Gathered"))
		t.grader.data.StudentTestResults = parser.UnitTestResultsAndCoverage
	} else {
		common.Debug(fmt.Sprintf("Teacher Test Results Gathered"))
		t.grader.data.TeacherTestResults = parser.UnitTestResultsAndCoverage
	}
	return err
}

func (t typescriptGrader) BuildProject(grader graderStruct) error {

	return nil
}

func (t *typescriptGrader) GradeTests(grader graderStruct) error {
	timeoutMilliseconds := t.grader.data.maxTestingTimeMilSecs
	convertToSeconds, err := strconv.Atoi(timeoutMilliseconds)
	if err != nil {
		return err
	}
	convertToSeconds = convertToSeconds / 1000
	if convertToSeconds == 0 {
		convertToSeconds = 1
	}
	timeoutSeconds := fmt.Sprintf("--testTimeout=%d", convertToSeconds)
	cmd := exec.Command("npm", "i", "jest-junit")
	cmd.Dir = grader.data.assignmentRootPath
	err = cmd.Run()

	//cmd = exec.Command("npm", "test")
	cmd = exec.Command("npm", "test", "--verbose", timeoutSeconds, "--collectCoverage")
	cmd.Dir = grader.data.assignmentRootPath

	// Kills command if taking too long
	timeoutMillisecondsBound := t.grader.data.maxTestingTimeUpperBound
	convertToMillisecondsBound, _ := strconv.Atoi(timeoutMillisecondsBound)
	go func() {
		<-time.After(time.Millisecond * time.Duration(convertToMillisecondsBound))
		_ = cmd.Process.Kill()
	}()

	err = cmd.Run()

	if err != nil {
		if strings.Contains(err.Error(), "signal: killed") {
			t.grader.data.exceededUpperBound = true
			dir := grader.data.repoPath + grader.data.repoName + grader.data.submissionTestPath
			<-time.After(time.Millisecond * 100) // If you don't give it some breathing time, the directory does not get deleted which can lead to problems
			common.MakeDir(dir)
			message := fmt.Sprintln("pytest execution ran for too long of a period. \nCould possibly mean an infinite loop exists in the code.\nCould also mean not enough time was given for process to finish\nWe can't solve the halting problem")
			dataToWrite := fmt.Sprintf("<?xml version=\"1.0\" encoding=\"utf-8\"?><testsuites><testsuite name=\"pytest\" errors=\"0\" failures=\"1\" skipped=\"0\" tests=\"1\" time=\"0.042\" timestamp=\"2023-01-29T11:23:36.249123\" hostname=\"Laptop.lan\"><testcase classname=\"CSGRADER_TOOK_TOO_LONG_ERROR\" name=\"CANNOT PROCESS THIS PROJECT\" time=\"0.001\"><failure message=\"%s\">self = &lt;test_with_unittest.TryTesting testMethod=CSGRADER_TOOK_TOO_LONG_ERROR04&gt;\n\n    CSGRADER_TOOK_TOO_LONG_ERROR_CSGRADER_TOOK_TOO_LONG_ERROR\n\nsrc/test_with_unittest.py:8: AssertionError</failure></testcase></testsuite></testsuites>", message)
			const permission = 0777
			errWrite := os.WriteFile(dir+"/TEST-result.xml", []byte(dataToWrite), permission)
			if errWrite != nil {
				return errWrite
			}
		}
		if err.Error() != "exit status 1" { // this just means the test failed
			return err
		}
	}

	cmd = exec.Command("coverage", "xml")
	cmd.Dir = grader.data.assignmentRootPath
	err = cmd.Run()

	if err != nil {
		return err
	}

	return err
}

func (t typescriptGrader) NonCodeSubmissionEnabled(grader graderStruct) bool {
	return grader.nonCodeSubmissionEnabled(grader)
}

func (t typescriptGrader) ShouldGradeUnitTests(grader graderStruct) bool {
	return grader.shouldGradeUnitTests(grader)
}

func (t typescriptGrader) TeacherTestsEnabled(grader graderStruct) bool {
	return grader.teacherTestsEnabled(grader)
}

func (t typescriptGrader) PullTeacherTests(grader graderStruct) error {
	return grader.pullTeacherTests(grader)
}

func (t typescriptGrader) GetNonCodeSubmissions(grader graderStruct) error {
	return grader.getNonCodeSubmissions(grader)
}

func (t typescriptGrader) PullAssignment(grader graderStruct) error {
	return grader.pullAssignment(grader)
}

func (t *typescriptGrader) GetComplexity() {
	t.grader.getComplexity()
}

func (t *typescriptGrader) PullRepoAndCheckCommits() bool {
	return t.grader.PullRepoAndCheckCommits()
}

func (t *typescriptGrader) SetStatus(status string) {
	t.grader.SetStatus(status)
}

func (t *typescriptGrader) SetStatusMessage(statusMessage string) {
	t.grader.SetStatusMessage(statusMessage)
}

func newTypescriptGrader() IGrader {
	return &typescriptGrader{
		grader: graderStruct{
			data: getGraderData("/tmp/typescript/", "/reports/"),
		},
	}
}
