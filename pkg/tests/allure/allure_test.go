package allure

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2/types"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAllureTest(t *testing.T) (string, func()) {
	gomega.RegisterTestingT(t)

	tempDir, err := os.MkdirTemp("", "allure-test-*")
	require.NoError(t, err)

	os.Setenv(resultsPathEnvKey, tempDir)
	os.Unsetenv(allureResultsPathEnvKey)
	resultsPath = "" // Reset to force recreation
	createFolderOnce = sync.Once{}

	cleanup := func() {
		os.RemoveAll(tempDir)
		os.Unsetenv(resultsPathEnvKey)
	}

	return tempDir, cleanup
}

func TestResolveExtension(t *testing.T) {
	assert.Equal(t, "tar.gz", resolveExtension(MimeTypeGZIP))
	assert.Equal(t, "", resolveExtension(MimeType("unknown")))
}

func TestNewAttachment(t *testing.T) {
	content := []byte("test content")
	att := newAttachment("test-attachment", MimeTypeGZIP, content)
	assert.Equal(t, "test-attachment", att.Name)
	assert.Equal(t, MimeTypeGZIP, att.Type)
	assert.Equal(t, content, att.content)
	assert.NotEmpty(t, att.uuid)
}

func TestSaveAsJSONAttachment(t *testing.T) {
	_, cleanup := setupAllureTest(t)
	defer cleanup()

	content := []byte("test")
	att := newAttachment("test", MimeTypeGZIP, content)
	jsonBytes := saveAsJSONAttachment(att)

	var unmarshalled attachment
	err := json.Unmarshal(jsonBytes, &unmarshalled)
	require.NoError(t, err)
	assert.Equal(t, "test", unmarshalled.Name)
	assert.Equal(t, MimeTypeGZIP, unmarshalled.Type)
}

func TestAddAttachment(t *testing.T) {
	tempDir, cleanup := setupAllureTest(t)
	defer cleanup()

	content := []byte("test data")
	att, err := addAttachment("file", MimeTypeGZIP, content)
	require.NoError(t, err)
	require.NotNil(t, att)

	// Check file existence
	files, err := os.ReadDir(filepath.Join(tempDir, "allure-results"))
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.True(t, filepath.Ext(files[0].Name()) == ".gz")
}

func TestAddAttachmentAPI(t *testing.T) {
	_, cleanup := setupAllureTest(t)
	defer cleanup()

	assert.Panics(t, func() {
		AddAttachment("test-api", MimeTypeGZIP, []byte("api data"))
	})
}

func TestStepAttributes(t *testing.T) {
	step := newStep()
	assert.Empty(t, step.ChildrenSteps)
	assert.Empty(t, step.Attachments)

	step.addName("my-step")
	assert.Equal(t, "my-step", step.Name)

	att := newAttachment("att", MimeTypeGZIP, []byte{})
	step.addAttachment(att)
	require.Len(t, step.Attachments, 1)
	assert.Equal(t, "att", step.Attachments[0].Name)
}

func TestStepAddNilAttachmentPanic(t *testing.T) {
	step := newStep()
	assert.Panics(t, func() { step.addAttachment(nil) })
}

func TestGetTestStatus(t *testing.T) {
	assert.Equal(t, broken, getTestStatus(types.SpecReport{State: types.SpecStatePanicked}))
	assert.Equal(t, skipped, getTestStatus(types.SpecReport{State: types.SpecStateAborted}))
	assert.Equal(t, skipped, getTestStatus(types.SpecReport{State: types.SpecStateInterrupted}))
	assert.Equal(t, skipped, getTestStatus(types.SpecReport{State: types.SpecStateSkipped}))
	assert.Equal(t, skipped, getTestStatus(types.SpecReport{State: types.SpecStatePending}))
	assert.Equal(t, failed, getTestStatus(types.SpecReport{State: types.SpecStateFailed}))
	assert.Equal(t, passed, getTestStatus(types.SpecReport{State: types.SpecStatePassed}))
	assert.Equal(t, "", getTestStatus(types.SpecReport{State: types.SpecStateInvalid}))
}

func TestTimestamps(t *testing.T) {
	ts := getTimestampMs()
	assert.Greater(t, ts, int64(0))

	tm := time.Unix(100, 500000000) // 100.5 seconds
	assert.Equal(t, int64(100500), getTimestampMsFromTime(tm))
}

func TestResultInitialization(t *testing.T) {
	res := newResult()
	assert.NotEmpty(t, res.UUID)
	assert.Greater(t, res.Start, int64(0))
	assert.Empty(t, res.Steps)
}

func TestResultFields(t *testing.T) {
	res := newResult()
	res.addSuite("my-suite")
	assert.Equal(t, "my-suite", res.Suite)
	assert.Contains(t, res.Labels, label{Name: "suite", Value: "my-suite"})

	res.addParentSuite("parent")
	assert.Equal(t, "parent", res.ParentSuite)
	assert.Contains(t, res.Labels, label{Name: "parentSuite", Value: "parent"})

	res.addFullName("full-name")
	assert.Equal(t, "full-name", res.FullName)

	details := statusDetails{Message: "error message"}
	res.setStatusDetails(details)
	assert.Equal(t, "error message", res.StatusDetails.Message)

	att := newAttachment("att", MimeTypeGZIP, []byte{})
	res.addAttachment(att)
	assert.Len(t, res.Attachments, 1)
}

func TestResultAddNilAttachmentPanic(t *testing.T) {
	res := newResult()
	assert.Panics(t, func() { res.addAttachment(nil) })
}

func TestContainerInitialization(t *testing.T) {
	c := newTestContainer()
	assert.NotEmpty(t, c.UUID)
	assert.NotNil(t, c.Befores)
	assert.NotNil(t, c.Afters)
}

func TestFolderCreation_AllureResultsPath(t *testing.T) {
	tempDir, cleanup := setupAllureTest(t)
	defer cleanup()

	testFolder := filepath.Join(tempDir, "custom-allure-results")
	os.Setenv(allureResultsPathEnvKey, testFolder)
	defer os.Unsetenv(allureResultsPathEnvKey)

	createFolderIfNotExists()
	assert.Equal(t, testFolder, resultsPath)

	_, err := os.Stat(testFolder)
	assert.NoError(t, err)
}

func TestFolderCreation_ReportsDir(t *testing.T) {
	tempDir, cleanup := setupAllureTest(t)
	defer cleanup()

	os.Unsetenv(allureResultsPathEnvKey)
	os.Setenv(resultsPathEnvKey, tempDir)

	createFolderIfNotExists()
	assert.Equal(t, filepath.Join(tempDir, "allure-results"), resultsPath)

	_, err := os.Stat(resultsPath)
	assert.NoError(t, err)
}

func TestFolderCreation_FallbackCWD(t *testing.T) {
	_, cleanup := setupAllureTest(t)
	defer cleanup()

	os.Unsetenv(allureResultsPathEnvKey)
	os.Unsetenv(resultsPathEnvKey)

	cwd, _ := os.Getwd()
	defer os.Setenv(resultsPathEnvKey, "/tmp") // restore safety

	createFolderIfNotExists()
	assert.Equal(t, filepath.Join(cwd, "allure-results"), resultsPath)

	_, err := os.Stat(resultsPath)
	assert.NoError(t, err)

	os.RemoveAll(resultsPath)
}

func TestProcessGinkgoReport(t *testing.T) {
	tempDir, cleanup := setupAllureTest(t)
	defer cleanup()

	mockReport := types.Report{
		SuitePath:        "/path/to/suite",
		SuiteDescription: "My Suite",
		StartTime:        time.Now().Add(-1 * time.Minute),
		EndTime:          time.Now(),
		SpecReports: types.SpecReports{
			{
				LeafNodeType: types.NodeTypeBeforeSuite,
				State:        types.SpecStatePassed,
				StartTime:    time.Now().Add(-1 * time.Minute),
				EndTime:      time.Now(),
			},
			{
				LeafNodeType:            types.NodeTypeIt,
				State:                   types.SpecStatePassed,
				StartTime:               time.Now().Add(-30 * time.Second),
				EndTime:                 time.Now(),
				LeafNodeText:            "My Test",
				ContainerHierarchyTexts: []string{"Suite", "Context"},
				IsSerial:                true,
			},
			{
				LeafNodeType: types.NodeTypeIt,
				State:        types.SpecStateFailed,
				StartTime:    time.Now().Add(-20 * time.Second),
				EndTime:      time.Now(),
				LeafNodeText: "My Failing Test",
				Failure: types.Failure{
					Message: "Some error",
					Location: types.CodeLocation{
						FileName:       "test.go",
						LineNumber:     10,
						FullStackTrace: "trace",
					},
				},
				ParallelProcess: 2,
			},
			{
				LeafNodeType: types.NodeTypeAfterSuite,
				State:        types.SpecStatePassed,
				StartTime:    time.Now().Add(-10 * time.Second),
				EndTime:      time.Now(),
			},
		},
	}

	err := FromGinkgoReport(mockReport)
	require.NoError(t, err)

	files, err := os.ReadDir(filepath.Join(tempDir, "allure-results"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 2) // container and result
}

func TestSpecEventsAndSteps(t *testing.T) {
	_, cleanup := setupAllureTest(t)
	defer cleanup()

	mockReport := types.Report{
		SuitePath:        "/path/to/suite",
		SuiteDescription: "My Suite",
		StartTime:        time.Now().Add(-1 * time.Minute),
		EndTime:          time.Now(),
		SpecReports: types.SpecReports{
			{
				LeafNodeType:            types.NodeTypeIt,
				State:                   types.SpecStatePassed,
				StartTime:               time.Now().Add(-30 * time.Second),
				EndTime:                 time.Now(),
				LeafNodeText:            "My Test",
				ContainerHierarchyTexts: []string{"Suite", "Context"},
				IsSerial:                true,
				SpecEvents: types.SpecEvents{
					{
						SpecEventType:    types.SpecEventByStart,
						Message:          "Step 1",
						CodeLocation:     types.CodeLocation{LineNumber: 10},
						TimelineLocation: types.TimelineLocation{Order: 1, Time: time.Now()},
					},
					{
						SpecEventType:    types.SpecEventByEnd,
						CodeLocation:     types.CodeLocation{LineNumber: 10},
						TimelineLocation: types.TimelineLocation{Order: 3, Time: time.Now()},
					},
				},
				ReportEntries: types.ReportEntries{
					{
						Name:             attachmentReportEntryName,
						Location:         types.CodeLocation{FileName: "foo.go", LineNumber: 10},
						TimelineLocation: types.TimelineLocation{Order: 2, Time: time.Now()},
						Value: func() types.ReportEntryValue {
							att := newAttachment("step-att", MimeTypeGZIP, []byte("ok"))
							b, _ := json.Marshal(att)
							return types.WrapEntryValue(string(b))
						}(),
					},
					{
						Name:  descriptionReportEntryName,
						Value: types.WrapEntryValue("Custom description"),
					},
				},
			},
		},
	}

	err := FromGinkgoReport(mockReport)
	require.NoError(t, err)
}
