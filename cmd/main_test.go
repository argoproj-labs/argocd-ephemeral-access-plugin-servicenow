package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	argocd "github.com/argoproj-labs/argocd-ephemeral-access/api/argoproj/v1alpha1"
	api "github.com/argoproj-labs/argocd-ephemeral-access/api/ephemeral-access/v1alpha1"
	"github.com/argoproj-labs/argocd-ephemeral-access/pkg/plugin"

	batchv1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	testclient "k8s.io/client-go/kubernetes/fake"
)

const correctCMDBInstallStatus = "1"
const addChange = true

type HelperMethodsTestSuite struct {
	suite.Suite
}

type K8SRelatedTestSuite struct {
	suite.Suite
}

type PluginHelperMethodsTestSuite struct {
	suite.Suite
}

type PublicMethodsTestSuite struct {
	suite.Suite
}

type ServiceNowTestSuite struct {
	suite.Suite
}

type CITestSuite struct {
	suite.Suite
}

type ChangeTestSuite struct {
	suite.Suite
}

type CheckCITestSuite struct {
	suite.Suite
}

type CheckChangeTestSuite struct {
	suite.Suite
}

type MockedLogger struct {
	mock.Mock
}

func (m *MockedLogger) Log(level hclog.Level, s string, args ...interface{}) {
	m.Called(s)
}

func (m *MockedLogger) Trace(s string, i ...interface{}) {
	m.Called(s)
}

func (m *MockedLogger) Debug(s string, i ...interface{}) {
	m.Called(s)
}

func (m *MockedLogger) Info(s string, i ...interface{}) {
	m.Called(s)
}

func (m *MockedLogger) Warn(s string, i ...interface{}) {
	m.Called(s)
}

func (m *MockedLogger) Error(s string, i ...interface{}) {
	m.Called(s)
}

func (m *MockedLogger) IsTrace() bool {
	m.Called()
	return false
}

func (m *MockedLogger) IsDebug() bool {
	m.Called()
	return true
}

func (m *MockedLogger) IsInfo() bool {
	m.Called()
	return true
}

func (m *MockedLogger) IsWarn() bool {
	m.Called()
	return true
}

func (m *MockedLogger) IsError() bool {
	m.Called()
	return true
}

func (m *MockedLogger) ImpliedArgs() []interface{} {
	m.Called()
	return nil
}

func (m *MockedLogger) With(args ...interface{}) hclog.Logger {
	m.Called()
	return nil
}

func (m *MockedLogger) Name() string {
	m.Called()
	return "myLogger"
}

func (m *MockedLogger) Named(s string) hclog.Logger {
	m.Called(s)
	return nil
}

func (m *MockedLogger) ResetNamed(s string) hclog.Logger {
	m.Called(s)
	return nil
}

func (m *MockedLogger) SetLevel(level hclog.Level) {
	m.Called(level)
}

func (m *MockedLogger) GetLevel() hclog.Level {
	m.Called()
	return hclog.Level(hclog.Debug)
}

func (m *MockedLogger) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	m.Called(opts)
	return nil
}

func (m *MockedLogger) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	m.Called(opts)
	return nil
}

func testResetEnvVar() {
	_ = os.Setenv("SERVICENOW_URL", "")
	_ = os.Setenv("TIMEZONE", "")
	_ = os.Setenv("KUBERNETES_SERVICE_HOST", "")
	_ = os.Setenv("KUBERNETES_SERVICE_PORT", "")
	_ = os.Setenv("EPHEMERAL_ACCESS_EXTENSION_NAMESPACE", "")
	_ = os.Setenv("SERVICENOW_SECRET_NAME", "")
}

func testGetPlugin() (*ServiceNowPlugin, *MockedLogger) {
	loggerObj := new(MockedLogger)

	p := &ServiceNowPlugin{
		Logger: loggerObj,
	}

	return p, loggerObj
}

func (s *HelperMethodsTestSuite) TestgetEnvVarWithoutDefaultWithEnvVar() {
	t := s.T()

	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	_ = os.Setenv("SERVICENOW_URL", "https://example.com")

	serviceNowUrl, errorText := p.getEnvVarWithoutDefault("SERVICENOW_URL", "Whatever")

	s.Equal("https://example.com", serviceNowUrl, "The correct URL is retrieved")
	s.Equal("", errorText, "No error text returned")
	loggerObj.AssertExpectations(t)
}

func (s *HelperMethodsTestSuite) TestgetEnvVarWithoutDefaultWithoutEnvVar() {
	t := s.T()

	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	expectedErrorText := "Expected error"

	loggerObj.On("Error", expectedErrorText)

	_, errorText := p.getEnvVarWithoutDefault("SERVICENOW_URL", expectedErrorText)

	s.Equal(expectedErrorText, errorText, "Error text should be the expected errortext")
	loggerObj.AssertExpectations(t)
}

func (s *HelperMethodsTestSuite) TestGetEnvVarWithDefaultWithEnvVar() {
	t := s.T()

	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	_ = os.Setenv("TIMEZONE", "Amsterdam/Europe")

	timezone := p.getEnvVarWithDefault("TIMEZONE", "UTC")

	s.Equal("Amsterdam/Europe", timezone, "Environment TIMEZONE was filled, Retrieved Amsterdam/Europe correctly")
	loggerObj.AssertExpectations(t)
}

func (s *HelperMethodsTestSuite) TestGetEnvVarWithDefaultWithoutEnvVar() {
	t := s.T()

	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	loggerObj.On("Debug", "Environment variable TIMEZONE is empty, assuming UTC")

	timezone := p.getEnvVarWithDefault("TIMEZONE", "UTC")

	s.Equal("UTC", timezone, "Assumed UTC correctly")
	loggerObj.AssertExpectations(t)
}

func (s *HelperMethodsTestSuite) TestGetLocalTime() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	time.Local = time.UTC
	currentTime := time.Now()
	timezone = "Europe/Amsterdam"
	currentTimeAmsterdamSummertime := time.Now().Add(2 * time.Hour)
	currentTimeAmsterdamWintertime := time.Now().Add(1 * time.Hour)

	currentTimeAmsterdamSummertimeString := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d",
		currentTimeAmsterdamSummertime.Year(),
		currentTimeAmsterdamSummertime.Month(),
		currentTimeAmsterdamSummertime.Day(),
		currentTimeAmsterdamSummertime.Hour(),
		currentTimeAmsterdamSummertime.Minute(),
		currentTimeAmsterdamSummertime.Second())
	currentTimeAmsterdamWintertimeString := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d",
		currentTimeAmsterdamWintertime.Year(),
		currentTimeAmsterdamWintertime.Month(),
		currentTimeAmsterdamWintertime.Day(),
		currentTimeAmsterdamWintertime.Hour(),
		currentTimeAmsterdamWintertime.Minute(),
		currentTimeAmsterdamWintertime.Second())

	currentTimeString := p.getLocalTime(currentTime)

	if currentTimeString != currentTimeAmsterdamSummertimeString &&
		currentTimeString != currentTimeAmsterdamWintertimeString {
		t.Errorf("Error: %s is not equal to Amsterdam Summertime (%s) or Amsterdam Wintertime (%s)",
			currentTimeString,
			currentTimeAmsterdamSummertimeString,
			currentTimeAmsterdamWintertimeString)
	}

	loggerObj.AssertExpectations(t)
}

func (s *HelperMethodsTestSuite) TestConvertTimeCorrectTime() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	result, errorText := p.convertTime("2025-05-15 18:14:13")
	s.Equal(18, result.Hour(), "Hours match")
	s.Equal(14, result.Minute(), "Minutes match")
	s.Equal(13, result.Second(), "Seconds match")
	s.Equal("", errorText, "No errors expected")

	loggerObj.AssertExpectations(t)
}

func (s *HelperMethodsTestSuite) TestConvertTimeIncorrectTime() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	timeString := "current"
	expectedErrorText := fmt.Sprintf("Error in converting %s to go Time: parsing time \"currentZ\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"currentZ\" as \"2006\"", timeString)

	loggerObj.On("Error", expectedErrorText)
	_, errorText := p.convertTime(timeString)
	s.Equal(expectedErrorText, errorText, "Error text is correct")
	loggerObj.AssertExpectations(t)
}

func (s *HelperMethodsTestSuite) TestConvertToIntSuccess() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	str := "1"
	result := p.convertToInt("Test set", str, 0)

	s.Equal(1, result, `Assuming string "1" to be 1`)
	loggerObj.AssertExpectations(t)
}

func (s *HelperMethodsTestSuite) TestConvertToIntFail() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	expectedErrorText := "Incorrect value, test in Test set: should be a number, assuming 0"
	loggerObj.On("Error", expectedErrorText)

	str := "test"
	result := p.convertToInt("Test set", str, 0)

	s.Equal(0, result, `Assuming string "test" to become default 0`)
	loggerObj.AssertExpectations(t)
}

func TestHelperMethods(t *testing.T) {
	suite.Run(t, new(HelperMethodsTestSuite))
}

func (s *K8SRelatedTestSuite) TestGetK8sConfigInUnittest() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	// Will always return error text because the tests are not run from within a Kubernetes cluster
	unittest = true

	errorText := p.getK8sConfig()
	s.Equal("", errorText, "Run in unittest should be successful")

	loggerObj.AssertExpectations(t)
}

func (s *K8SRelatedTestSuite) TestGetK8sConfigOutsideKubernetes() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	loggerObj.On("Error", mock.Anything) // Has different output local and in the pipeline, so just ignore

	// Will always return error text because the tests are not run from within a Kubernetes cluster
	unittest = false

	_ = os.Setenv("KUBERNETES_SERVICE_HOST", "https://kubernetes.example.com")
	_ = os.Setenv("KUBERNETES_SERVICE_PORT", "6443")

	errorText := p.getK8sConfig()
	cont := s.Contains(errorText, "Error in getK8sConfig, rest.InClusterConfig: open /var/run/secrets/kubernetes.io/serviceaccount/token")
	if !cont {
		t.Error("Expected error when not run within a Kubernetes cluster")
	}
	// No check on error in logger, details will be different in pipeline compared to local
}

func (s *K8SRelatedTestSuite) TestGetCredentialsFromSecret() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	secretName := "generic-secret"
	namespace := "generic-namespace"
	genericUsername := "GenericUsername"
	genericPassword := "GenericPassword"

	k8sclientset = testclient.NewClientset()
	setSecret(namespace, secretName, genericUsername, genericPassword)

	loggerObj.On("Debug", fmt.Sprintf("Get credentials from secret [%s]%s...", namespace, secretName))

	username, password, errorText := p.getCredentialsFromSecret(namespace, secretName, "username", "password")

	s.Equal(genericUsername, username, "Username found")
	s.Equal(genericPassword, password, "Password found")
	s.Equal("", errorText, "No error text expected")

	loggerObj.AssertExpectations(t)
}

func (s *K8SRelatedTestSuite) TestGetCredentialsFromSecretSecretDoesntExist() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	secretName := "generic-secret"
	namespace := "generic-namespace"
	genericUsername := "GenericUsername"
	genericPassword := "GenericPassword"

	k8sclientset = testclient.NewClientset()
	setSecret(namespace, secretName, genericUsername, genericPassword)

	secretName = "does-not-exist"
	expectedErrorText := fmt.Sprintf(`Error getting secret %s, does secret exist in namespace %s? Error: secrets "does-not-exist" not found`,
		secretName,
		namespace)

	loggerObj.On("Debug", fmt.Sprintf("Get credentials from secret [%s]%s...", namespace, secretName))
	loggerObj.On("Error", expectedErrorText)

	_, _, errorText := p.getCredentialsFromSecret(namespace, secretName, "username", "password")

	s.Equal(expectedErrorText, errorText, "Errortext should be correct")
	loggerObj.AssertExpectations(t)
}

func (s *K8SRelatedTestSuite) TestGetExclusionsFromConfigMapWithOneExclusion() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	namespace := "argocd-ephemeral-access"
	exclusionsString := "administrator"

	loggerObj.On("Debug", fmt.Sprintf("Get exclusions from configmap [argocd-ephemeral-access]%s", ExclusionsConfigMapName))
	loggerObj.On("Debug", "Exclusions used: "+exclusionsString)

	k8sclientset = testclient.NewClientset()
	setConfigMap(namespace, ExclusionsConfigMapName, "exclusion-roles", exclusionsString)
	exclusions := p.getExclusionsFromConfigMap(namespace)

	s.Equal([]string{"administrator"}, exclusions)
	loggerObj.AssertExpectations(t)
}

func (s *K8SRelatedTestSuite) TestGetExclusionsFromConfigMapWithTwoExclusions() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	namespace := "argocd-ephemeral-access"
	exclusionsString := "administrator\nincidentmanager"

	loggerObj.On("Debug", fmt.Sprintf("Get exclusions from configmap [argocd-ephemeral-access]%s", ExclusionsConfigMapName))
	loggerObj.On("Debug", "Exclusions used: "+exclusionsString)

	k8sclientset = testclient.NewClientset()
	setConfigMap(namespace, ExclusionsConfigMapName, "exclusion-roles", exclusionsString)
	exclusions := p.getExclusionsFromConfigMap(namespace)

	s.Equal([]string{"administrator", "incidentmanager"}, exclusions)
	loggerObj.AssertExpectations(t)
}

func (s *K8SRelatedTestSuite) TestGetExclusionsFromConfigMapWithConfigMapWithoutExclusions() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	namespace := "argocd-ephemeral-access"
	exclusionsString := ""

	loggerObj.On("Debug", fmt.Sprintf("Get exclusions from configmap [argocd-ephemeral-access]%s", ExclusionsConfigMapName))
	loggerObj.On("Debug", "Exclusions used: "+exclusionsString)

	k8sclientset = testclient.NewClientset()
	setConfigMap(namespace, ExclusionsConfigMapName, "exclusion-roles", exclusionsString)
	exclusions := p.getExclusionsFromConfigMap(namespace)

	s.Equal([]string{""}, exclusions, "No exclusions")
	loggerObj.AssertExpectations(t)
}

func (s *K8SRelatedTestSuite) TestGetExclusionsFromConfigMapWithoutConfigMap() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	namespace := "argocd-ephemeral-access"

	loggerObj.On("Debug", fmt.Sprintf("Get exclusions from configmap [argocd-ephemeral-access]%s", ExclusionsConfigMapName))
	loggerObj.On("Debug", "Error getting configmap controller-cm, does configmap exist in namespace argocd-ephemeral-access?")
	loggerObj.On("Debug", "No exclusions used")

	k8sclientset = testclient.NewClientset()
	exclusions := p.getExclusionsFromConfigMap(namespace)

	s.Equal([]string{}, exclusions)
	loggerObj.AssertExpectations(t)
}

func TestK8SRelated(t *testing.T) {
	suite.Run(t, new(K8SRelatedTestSuite))
}

func (s *PluginHelperMethodsTestSuite) TestGetGlobalVars() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	exampleUrl := "https://example.com"
	_ = os.Setenv("SERVICENOW_URL", exampleUrl)

	secretName := "servicenow-secret"
	ephemeralAccessPluginNamespace = "argocd-ephemeral-access"
	testUsername := "my-username"
	testPassword := "my-password"
	k8sclientset = testclient.NewClientset()
	setConfigMap(ephemeralAccessPluginNamespace, ExclusionsConfigMapName, "exclusion-roles", "")
	setSecret(ephemeralAccessPluginNamespace, secretName, testUsername, testPassword)

	loggerObj.On("Debug", mock.Anything)

	unittest = true
	errorText := p.getGlobalVars()

	s.Equal(exampleUrl, serviceNowUrl, "serviceNowUrl should be retrieved from environment variables")
	s.Equal("UTC", timezone, "Default timezone should be UTC")
	s.Equal(testUsername, serviceNowUsername, "ServiceNow username should be correct")
	s.Equal(testPassword, serviceNowPassword, "ServiceNow password should be correct")
	s.Equal([]string{""}, exclusionRoles, "Default for exclusion roles is empty")
	s.Equal("", errorText, "Not expected error texts")
	loggerObj.AssertExpectations(t)
}

func (s *PluginHelperMethodsTestSuite) TestGetGlobalVarsExclusionGroupsWithValue() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	exampleUrl := "https://example.com"
	_ = os.Setenv("SERVICENOW_URL", exampleUrl)

	secretName := "servicenow-secret"
	namespace := "argocd-ephemeral-access"
	testUsername := "my-username"
	testPassword := "my-password"

	k8sclientset = testclient.NewClientset()
	setConfigMap(namespace, ExclusionsConfigMapName, "exclusion-roles", "incidentmanagers")
	setSecret(namespace, secretName, testUsername, testPassword)

	loggerObj.On("Debug", mock.Anything)

	unittest = true
	p.getGlobalVars()

	s.Equal([]string{"incidentmanagers"}, exclusionRoles, "Exclusion roles should be correct")
	loggerObj.AssertExpectations(t)
}

func (s *PluginHelperMethodsTestSuite) TestShowRequest() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	var ar = new(api.AccessRequest)
	var app = new(argocd.Application)

	ar.Spec.Subject.Username = "Testuser"
	ar.Spec.Role.TemplateRef.Name = "administrator"
	ar.Spec.Application.Namespace = "argocd"
	ar.Spec.Application.Name = "demoapp"
	ar.Spec.Duration.Duration = time.Hour * 4

	jsonAr, _ := json.Marshal(ar)
	jsonApp, _ := json.Marshal(app)

	loggerObj.On("Info", "Call to GrantAccess: username: Testuser, role: administrator, application: [argocd]demoapp, duration: 4h0m0s")
	loggerObj.On("Debug", "jsonAr: "+string(jsonAr))
	loggerObj.On("Debug", "jsonApp: "+string(jsonApp))

	p.showRequest(ar, app)
	loggerObj.AssertExpectations(t)
}

func testConvertTimeToString(t time.Time) string {
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}

func (s *PluginHelperMethodsTestSuite) TestCreateRevokeJobCorrect() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	namespace := "argocd"
	accessRequestName := "test-ar"
	jobStartTime := time.Now().Add(-1 * time.Minute)
	expectedJobName := "stop-" + accessRequestName

	k8sclientset = testclient.NewClientset()

	loggerObj.On("Debug", fmt.Sprintf("createRevokeJob: %s, %s", namespace, accessRequestName))
	loggerObj.On("Info", fmt.Sprintf("Created K8s job %s successfully in namespace argocd", expectedJobName))

	p.createRevokeJob(namespace, accessRequestName, jobStartTime)

	expectedSchedule := fmt.Sprintf("%d %d %d %d *",
		jobStartTime.Minute(),
		jobStartTime.Hour(),
		jobStartTime.Day(),
		jobStartTime.Month())
	expectedCommand := []string{"sh", "-c", fmt.Sprintf("kubectl delete accessrequest -n argocd %s && kubectl delete cronjob -n argocd %s", accessRequestName, expectedJobName)}
	cronjobs := k8sclientset.BatchV1().CronJobs(namespace)
	myCronJob, err := cronjobs.Get(context.TODO(), expectedJobName, metav1.GetOptions{})

	var zero int32 = 0

	s.Equal(expectedJobName, myCronJob.Name, "Name should be the correct job name")
	s.Equal(namespace, myCronJob.Namespace, "Namespace should be the correct namespace")
	s.Equal(expectedSchedule, myCronJob.Spec.Schedule, "Schedule should be the correct schedule")
	s.Equal("remove-accessrequest-job-sa", myCronJob.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName, "Service account name should be the correct service account name")
	s.Equal(expectedJobName, myCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Name, "Container name should be the correct container name")
	s.Equal("bitnami/kubectl:latest", myCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image, "Image should be the correct image")
	s.Equal(expectedCommand, myCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Command, "Image should be the correct image")
	s.Equal(coreV1.RestartPolicyNever, myCronJob.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy, "Restart policy should be never")
	s.Equal(&zero, myCronJob.Spec.JobTemplate.Spec.BackoffLimit, "BackoffLimit should be 0")

	s.Equal(nil, err, "No errors expected")

	loggerObj.AssertExpectations(t)
	_ = cronjobs.Delete(context.TODO(), expectedJobName, metav1.DeleteOptions{})
}

func (s *PluginHelperMethodsTestSuite) TestCreateRevokeJobFail() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	namespace := "argocd"
	accessRequestName := "test-ar"
	jobStartTime := time.Now().Add(-1 * time.Minute)
	expectedJobName := "stop-" + accessRequestName

	k8sclientset = testclient.NewClientset()

	loggerObj.On("Debug", fmt.Sprintf("createRevokeJob: %s, %s", namespace, accessRequestName))
	loggerObj.On("Error", fmt.Sprintf("Failed to create K8s job %s in namespace argocd: cronjobs.batch \"stop-test-ar\" already exists.", expectedJobName))

	cronjobs := k8sclientset.BatchV1().CronJobs(namespace)
	cronJobSpec := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      expectedJobName,
			Namespace: namespace,
		},
	}
	_, _ = cronjobs.Create(context.TODO(), cronJobSpec, metav1.CreateOptions{})

	p.createRevokeJob(namespace, accessRequestName, jobStartTime)

	loggerObj.AssertExpectations(t)
	_ = cronjobs.Delete(context.TODO(), expectedJobName, metav1.DeleteOptions{})
}

func (s *PluginHelperMethodsTestSuite) TestDetermineDurationAndRealEndTimeChangeTimeWins() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	var arDuration = 4 * time.Hour
	var changeRemainingTime = 1 * time.Hour
	var endDate = time.Now().Add(1 * time.Hour)

	duration, realEndTime := p.determineDurationAndRealEndTime(arDuration, changeRemainingTime, endDate)

	s.Equal(changeRemainingTime, duration, "Expected duration: 1 hour")
	s.Equal(endDate, realEndTime, "Expected end time: 1 hour from now")

	loggerObj.AssertExpectations(t)
}

func (s *PluginHelperMethodsTestSuite) TestDetermineDurationAndRealEndTimeArDurationWins() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	var arDuration = 4 * time.Hour
	var changeRemainingTime = (8 * time.Hour) + (time.Microsecond * 50)
	var endDate = time.Now().Add(8 * time.Hour)

	var expectedRealEndTime = time.Now().Add(4 * time.Hour).Truncate(time.Second)

	duration, realEndTime := p.determineDurationAndRealEndTime(arDuration, changeRemainingTime, endDate)

	s.Equal(arDuration, duration, "Expected duration: 4 hour")
	s.Equal(expectedRealEndTime, realEndTime.Truncate(time.Second), "Expected end time: 4 hour from now")

	loggerObj.AssertExpectations(t)
}

func (s *PluginHelperMethodsTestSuite) TestDetermineGrantedTextsChange() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	requesterName := "TestUser"
	requestedRole := "admin"
	var validChange Change

	validChange.Type = "1"
	validChange.Number = "CHG300300"
	validChange.ShortDescription = "unittests"
	validChange.EndDate = time.Date(2025, 5, 20, 23, 59, 59, 0, time.UTC)
	realEndDate := time.Date(2025, 5, 20, 23, 59, 59, 0, time.UTC)

	var remainingTime = 1 * time.Hour
	expectedGrantedAccessText := fmt.Sprintf("Granted access for %s: %s change %s (%s), role %s, from %s to %s",
		requesterName,
		validChange.Type,
		validChange.Number,
		validChange.ShortDescription,
		requestedRole,
		time.Now().Truncate(time.Minute),
		realEndDate.Truncate(time.Second).String())
	expectedGrantedAccessUIText := fmt.Sprintf("Granted access: change __%s__ (%s), until __%s (%s)__",
		validChange.Number,
		validChange.ShortDescription,
		p.getLocalTime(realEndDate),
		remainingTime.Truncate(time.Second).String())
	expectedGrantedAccessServiceNowText := fmt.Sprintf("ServiceNow plugin granted access to %s, for role %s, until %s (%s)",
		requesterName,
		requestedRole,
		p.getLocalTime(realEndDate),
		remainingTime.Truncate(time.Second).String())

	loggerObj.On("Info", expectedGrantedAccessText)
	loggerObj.On("Debug", expectedGrantedAccessUIText)

	grantedAccessUIText, grantedAccessServiceNowText := p.determineGrantedTextsChange(requesterName, requestedRole, validChange, remainingTime, realEndDate)

	s.Equal(expectedGrantedAccessUIText, grantedAccessUIText, "Granted access text for UI should be what is expected")
	s.Equal(expectedGrantedAccessServiceNowText, grantedAccessServiceNowText, "Granted access text for ServiceNow should be what is expected")

	loggerObj.AssertExpectations(t)
}

func (s *PluginHelperMethodsTestSuite) TestDetermineGrantedTextsExclusions() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	requesterName := "TestUser"
	requestedRole := "admin"

	var remainingTime = 1 * time.Hour
	realEndDate := time.Now().Add(remainingTime)

	expectedGrantedAccessText := fmt.Sprintf("Granted access for %s: role %s, from %s to %s (no change, %s is an exclusion role)",
		requesterName,
		requestedRole,
		time.Now().Truncate(time.Minute),
		realEndDate.Truncate(time.Minute),
		requestedRole)
	expectedGrantedAccessUIText := fmt.Sprintf("Granted access: %s is an exclusion role, until __%s (%s)__",
		requestedRole,
		p.getLocalTime(realEndDate),
		remainingTime.Truncate(time.Second).String())

	loggerObj.On("Warn", expectedGrantedAccessText)
	loggerObj.On("Debug", expectedGrantedAccessUIText)

	grantedAccessUIText := p.determineGrantedTextsExclusions(requesterName, requestedRole, remainingTime, realEndDate)

	s.Equal(expectedGrantedAccessUIText, grantedAccessUIText, "Granted access text for UI should be what is expected")
	loggerObj.AssertExpectations(t)
}

func (s *PluginHelperMethodsTestSuite) TestDenyRequest() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	reason := "whatever"
	response, err := p.denyRequest(reason)

	s.Equal(plugin.GrantStatusDenied, response.Status, response, "Access request should be denied")
	s.Equal(reason, response.Message, response, "Reason should be correct")
	s.Equal(nil, err, "No error")

	loggerObj.AssertExpectations(t)
}

func (s *PluginHelperMethodsTestSuite) TestGrantRequest() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	reason := "whatever"
	response, err := p.grantRequest(reason)

	s.Equal(plugin.GrantStatusGranted, response.Status, response, "Access request should be denied")
	s.Equal(reason, response.Message, response, "Reason should be correct")
	s.Equal(nil, err, "No error")

	loggerObj.AssertExpectations(t)
}

func setSecret(namespace string, secretName string, username string, password string) {
	secret := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: "Opaque",
		Data: map[string][]byte{
			"username": []byte(username),
			"password": []byte(password),
		},
	}

	_, _ = k8sclientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
}

func setConfigMap(namespace string, configmapName string, exclusionsListName string, exclusionsListValue string) {
	configMap := &coreV1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configmapName,
			Namespace: namespace,
		},
		Data: map[string]string{
			exclusionsListName: exclusionsListValue,
		},
	}

	_, _ = k8sclientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
}

func (s *PluginHelperMethodsTestSuite) TestGetServiceNowCredentials() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	secretName := "servicenow-secret"
	namespace := "argocd-ephemeral-access"
	serviceNowUsername := "serviceNowUsername"
	serviceNowPassword := "serviceNowPassword"
	ephemeralAccessPluginNamespace = namespace

	k8sclientset = testclient.NewClientset()
	setSecret(namespace, secretName, serviceNowUsername, serviceNowPassword)

	loggerObj.On("Debug", "Environment variable SERVICENOW_SECRET_NAME is empty, assuming servicenow-secret")
	loggerObj.On("Debug", "Get credentials from secret [argocd-ephemeral-access]servicenow-secret...")

	username, password, errorText := p.getServiceNowCredentials()

	s.Equal(serviceNowUsername, username, "Username found")
	s.Equal(serviceNowPassword, password, "Password found")
	s.Equal("", errorText, "No error expected")

	loggerObj.AssertExpectations(t)
}

func TestPluginHelperMethods(t *testing.T) {
	suite.Run(t, new(PluginHelperMethodsTestSuite))
}

func simulateHttpRequestToServiceNow(t *testing.T, responseMap map[string]string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := ""

		for uri := range responseMap {
			if uri == r.URL.RequestURI() {
				response = responseMap[uri]
				break
			}
		}
		if response == "" {
			fmt.Printf("No response found for %s, error in testset?\n", r.URL.RequestURI())
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept: application/json header, got: %s", r.Header.Get("Accept"))
		}
		usedUsername, usedPassword, ok := r.BasicAuth()
		if ok {
			assert.Equal(t, serviceNowUsername, usedUsername, "Username that is used should match username that is requested")
			assert.Equal(t, serviceNowPassword, usedPassword, "Password that is used should match username that is requested")
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	return server
}

func simulateSimpleHttpRequestWithStatusCodeRedirect(response string) (*httptest.Server, *httptest.Server) {

	var secondServer = httptest.NewServer(http.HandlerFunc(func(w2 http.ResponseWriter, r2 *http.Request) {
		w2.WriteHeader(http.StatusOK)
		_, _ = w2.Write([]byte(response))
	}))

	server := httptest.NewServer(http.HandlerFunc(func(w1 http.ResponseWriter, r1 *http.Request) {
		http.Redirect(w1, r1, secondServer.URL, http.StatusFound)
	}))
	return server, secondServer
}

func (s *ServiceNowTestSuite) TestCheckAPIResultNormalResponse() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	var resp = http.Response{
		Status:     "200 OK",
		StatusCode: 200,
	}
	var body = `{"result":[]}`

	result, errorText := p.checkAPIResult(&resp, []byte(body))

	s.Equal(body, string(result), "Body should not be changed")
	s.Equal("", errorText, "No error expected")
	loggerObj.AssertExpectations(t)
}

func (s *ServiceNowTestSuite) TestCheckAPIResultForbidden() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	expectedErrorText := "ServiceNow API changed"

	var resp = http.Response{
		Status:     "403 Forbidden",
		StatusCode: 403,
	}
	var body = `{"result":[]}`

	_, errorText := p.checkAPIResult(&resp, []byte(body))
	s.Equal(expectedErrorText, errorText, "Correct error text")
	loggerObj.AssertExpectations(t)
}

func (s *ServiceNowTestSuite) TestCheckAPIResultBadGateway() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	expectedErrorText := "ServiceNow API server is down"

	var resp = http.Response{
		Status:     "502 Bad Gateway",
		StatusCode: 502,
	}
	var body = `{"result":[]}`

	_, errorText := p.checkAPIResult(&resp, []byte(body))
	s.Equal(expectedErrorText, errorText, "Correct error text")
	loggerObj.AssertExpectations(t)
}

func (s *ServiceNowTestSuite) TestCheckAPIResultBadGatewayWith200() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	expectedErrorText := "ServiceNow API server is down"

	var resp = http.Response{
		Status:     "200 OK",
		StatusCode: 200,
	}
	responseText := "<html><body>Server down!</body></html>"

	_, errorText := p.checkAPIResult(&resp, []byte(responseText))
	s.Equal(expectedErrorText, errorText, "Correct error text")
	loggerObj.AssertExpectations(t)
}

func (s *ServiceNowTestSuite) TestgetFromServiceNowAPINormalResponse() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	responseText := "{\"results\":[]}"

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"
	requestURI := "/api/test"

	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText

	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	apiCall := fmt.Sprintf("%s%s", server.URL, requestURI)

	loggerObj.On("Debug", fmt.Sprintf("apiCall: %s", apiCall))
	loggerObj.On("Debug", responseText)

	result, errorText := p.getFromServiceNowAPI(requestURI)
	s.Equal(responseText, string(result), "Correct result from API")
	s.Equal("", errorText, "No errors expected")
	loggerObj.AssertExpectations(t)
}

func (s *ServiceNowTestSuite) TestgetFromServiceNowAPINormalResponseWithRedirect() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	responseText := "{\"results\":[]}"

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"
	requestURI := "/api/test"

	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText

	server, secondServer := simulateSimpleHttpRequestWithStatusCodeRedirect(responseText)
	defer server.Close()
	defer secondServer.Close()
	serviceNowUrl = server.URL

	apiCall := fmt.Sprintf("%s%s", server.URL, requestURI)
	loggerObj.On("Debug", fmt.Sprintf("apiCall: %s", apiCall))
	loggerObj.On("Debug", responseText)

	// Redirect is done by http, not by the plugin program. We will therefore just get the apiCall for the primary address,
	// not for the redirect server.
	//
	// It is very important that redirects works.

	result, errorText := p.getFromServiceNowAPI(requestURI)
	s.Equal(responseText, string(result), "Correct result from API")
	s.Equal("", errorText, "No errors expected")
	loggerObj.AssertExpectations(t)
}

func (s *ServiceNowTestSuite) TestGetFromServiceNowAPIErrorInApiCall() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	responseText := "{\"results\":[]}"

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"
	ciName := "app-demoapp"

	requestURI := getTestCIRequestURI(ciName)

	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText

	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	// requestURI is changed to something incorrect (contains serviceNowUrl which it shouldn't)

	requestURI = fmt.Sprintf("%s%s", serviceNowUrl, requestURI)

	// Expected apiCall contains the serviceNowUrl twice
	apiCall := fmt.Sprintf("%s%s", serviceNowUrl, requestURI)

	loggerObj.On("Debug", fmt.Sprintf("apiCall: %s", apiCall))
	loggerObj.On("Error", mock.Anything)

	_, errorText := p.getFromServiceNowAPI(requestURI)
	if !strings.Contains(errorText, "no such host") {
		t.Errorf("%s should contain text no such host", errorText)
	}

	loggerObj.AssertExpectations(t)
}

// PostNote is a very simple method, so re-use the test for PatchServiceNowAPINormalRequest for both methods

func testPatchServiceNowAPINormalRequest(s *ServiceNowTestSuite, requestURI string, data string, responseText string) {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	apiCall := fmt.Sprintf("%s%s", serviceNowUrl, requestURI)
	loggerObj.On("Debug", "apiCall: "+apiCall)
	loggerObj.On("Debug", "Data: "+data)
	loggerObj.On("Debug", "Body: "+responseText)

	result, errorText := p.patchServiceNowAPI(requestURI, data)
	s.Equal(responseText, string(result), "Correct result from API")
	s.Equal("", errorText, "No errors expected")
	loggerObj.AssertExpectations(t)
}

func (s *ServiceNowTestSuite) TestPatchServiceNowAPINormalRequest() {
	serviceNowUrl = "https://example.com"
	requestURI := "/api/test/1"
	data := `{"test": 1, "result": "success"}`
	responseText := `{"test": 1, "testText": "More results than the data that is sent", "result": "success"}`

	testPatchServiceNowAPINormalRequest(s, requestURI, data, responseText)
}

func (s *ServiceNowTestSuite) TestPatchServiceNowAPIErrorInApiCall() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	serviceNowUrl = "https://example.com"
	// Incorrect requestURI containing the serviceNowUrl
	incorrectRequestURI := fmt.Sprintf("%s/api/test/1", serviceNowUrl)
	data := `{"test": 1, "result": "success"}`

	// No need to set up simulation server, as this will never be reached

	apiCall := fmt.Sprintf("%s%s", serviceNowUrl, incorrectRequestURI)
	loggerObj.On("Debug", "apiCall: "+apiCall)
	loggerObj.On("Debug", "Data: "+data)
	loggerObj.On("Error", mock.Anything)

	_, errorText := p.patchServiceNowAPI(incorrectRequestURI, data)
	if !strings.Contains(fmt.Sprintf("%v", errorText), "no such host") {
		t.Errorf("%s should contain text no such host", errorText)
	}
	loggerObj.AssertExpectations(t)
}

func TestServiceNowMethods(t *testing.T) {
	suite.Run(t, new(ServiceNowTestSuite))
}

func (s *ServiceNowTestSuite) TestGetCINameFilled() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	var app = new(argocd.Application)
	var m = make(map[string]string)

	loggerObj.On("Debug", "Search for ci-name in the CMDB...")
	loggerObj.On("Debug", "ciLabel ci-name found: app-demoapp")

	ciLabel = "ci-name"
	m[ciLabel] = "app-demoapp"
	app.Labels = m

	ciLabel := p.getCIName(app)

	s.Equal("app-demoapp", ciLabel, "Label found, correct content")
	loggerObj.AssertExpectations(t)
}

func (s *ServiceNowTestSuite) TestGetCINameEmpty() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	var app = new(argocd.Application)
	var m map[string]string

	loggerObj.On("Debug", "Search for ci-name in the CMDB...")
	loggerObj.On("Debug", "ciLabel ci-name found: ")

	app.Labels = m
	ciLabel = "ci-name"
	ciName := p.getCIName(app)

	s.Equal("", ciName, "No label found, assume empty string")
	loggerObj.AssertExpectations(t)
}

func testPrepareGetCI(t *testing.T, ciName string, responseText string) (*httptest.Server, string) {
	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"
	requestURI := getTestCIRequestURI(ciName)

	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText

	server := simulateHttpRequestToServiceNow(t, responseMap)
	serviceNowUrl = server.URL

	apiCall := fmt.Sprintf("%s%s", server.URL, requestURI)

	return server, apiCall
}

func (s *CITestSuite) TestGetCIOneCI() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	ciName := "app-demoapp"
	responseText := fmt.Sprintf(`{"result":[{"install_status":"1", "name":"%s", "sys_id": "5"}]}`, ciName)
	server, apiCall := testPrepareGetCI(t, ciName, responseText)
	defer server.Close()

	loggerObj.On("Debug", fmt.Sprintf("apiCall: %s", apiCall))
	loggerObj.On("Debug", responseText)
	loggerObj.On("Debug", "InstallStatus: 1, CI name: app-demoapp, SysId: 5")

	cmdb, errorText := p.getCI(ciName)

	s.Equal("1", cmdb.InstallStatus, "InstallStatus should be 1")
	s.Equal(ciName, cmdb.Name, "Name should be "+ciName)
	s.Equal("5", cmdb.SysId, "SysId should be 5")
	s.Equal("", errorText, "No errors expected")
	loggerObj.AssertExpectations(t)
}

func (s *CITestSuite) TestGetCITwoCIs() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	ciName := "app-demoapp"

	responseText := `{"result":[{"install_status":"1", "name":"app-demoapp", "sys_id": "1"},
	                            {"install_status":"6", "name":"app-demoapp", "sys_id": "2"}]}`
	server, apiCall := testPrepareGetCI(t, ciName, responseText)
	defer server.Close()

	loggerObj.On("Debug", fmt.Sprintf("apiCall: %s", apiCall))
	loggerObj.On("Debug", responseText)
	loggerObj.On("Debug", "InstallStatus: 1, CI name: app-demoapp, SysId: 1")

	cmdb, errorText := p.getCI(ciName)

	s.Equal("1", cmdb.InstallStatus, "InstallStatus should be 1")
	s.Equal(ciName, cmdb.Name, "Name should be "+ciName)
	s.Equal("", errorText, "No errors expected")
	loggerObj.AssertExpectations(t)
}

func (s *CITestSuite) TestGetCINoCI() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	responseText := "{\"result\":[]}"
	expectedErrorText := "No CI with name app-demoapp found"

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"
	ciName := "app-demoapp"

	requestURI := getTestCIRequestURI(ciName)
	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()

	serviceNowUrl = server.URL
	apiCall := fmt.Sprintf("%s%s", server.URL, requestURI)

	loggerObj.On("Debug", fmt.Sprintf("apiCall: %s", apiCall))
	loggerObj.On("Debug", responseText)
	loggerObj.On("Error", expectedErrorText)

	_, errorText := p.getCI(ciName)

	s.Equal(expectedErrorText, errorText, "Expected error text is correct")
	loggerObj.AssertExpectations(t)
}

func (s *ServiceNowTestSuite) TestGetCIServerDown() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	expectedErrorText := "ServiceNow API server is down"
	responseText := "<html><body>Server down!</body></html>"

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"
	ciName := "app-demoapp"

	requestURI := getTestCIRequestURI(ciName)
	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()

	apiCall := fmt.Sprintf("%s%s", server.URL, requestURI)

	loggerObj.On("Debug", fmt.Sprintf("apiCall: %s", apiCall))
	loggerObj.On("Debug", responseText)
	defer server.Close()
	serviceNowUrl = server.URL

	_, errorText := p.getCI("app-demoapp")

	s.Equal(expectedErrorText, errorText, "Correct error")
	loggerObj.AssertExpectations(t)
}

func (s *ServiceNowTestSuite) TestGetCINoJSON() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	expectedErrorText := "Error in json.Unmarshal: invalid character '<' looking for beginning of value (<Result/>)"
	responseText := "<Result/>"

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"
	ciName := "app-demoapp"

	requestURI := getTestCIRequestURI(ciName)
	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	apiCall := fmt.Sprintf("%s%s", server.URL, requestURI)

	loggerObj.On("Debug", fmt.Sprintf("apiCall: %s", apiCall))
	loggerObj.On("Debug", responseText)
	loggerObj.On("Error", expectedErrorText)

	_, errorText := p.getCI(ciName)
	s.Equal(expectedErrorText, errorText, "Correct error text")
	loggerObj.AssertExpectations(t)
}

func TestCIMethods(t *testing.T) {
	suite.Run(t, new(CITestSuite))
}

func getHttpDateTime(t time.Time) string {
	return fmt.Sprintf("%04d%%2d%02d%%2d%02d", t.Year(), t.Month(), t.Day())
}

func getExpectedRequestURI(cmdb_ci string, startDate time.Time, endDate time.Time, sysparmOffset int) string {
	requestURIStart := "/api/now/table/change_request?sysparm_query=cmdb_ci%3d" + cmdb_ci + "%5estate%3d%2d1%5ephase%3drequested%5eapproval%3dapproved%5eactive%3dtrue"

	startDateHttpString := getHttpDateTime(startDate)
	endDateHttpString := getHttpDateTime(endDate)
	firstTimeString := "00%3a00%3a00"
	lastTimeString := "23%3a59%3a59"
	requestURIStartDatePart := "GOTOstart_date%3e" + startDateHttpString + "%20" + firstTimeString
	requestURIEndDatePart := "GOTOend_date%3c" + endDateHttpString + "%20" + lastTimeString

	sysparmOffsetString := fmt.Sprintf("%d", sysparmOffset)
	requestURIEnd := "&sysparm_fields=type,number,short_description,start_date,end_date,sys_id&sysparm_limit=5&sysparm_offset=" + sysparmOffsetString

	return requestURIStart + "%5e" + requestURIStartDatePart + "%5e" + requestURIEndDatePart + requestURIEnd
}

func (s *ChangeTestSuite) TestGetChangeRequestURIDateWindow0Days() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	cmdbCi := "id1"
	sysparmOffset := 0

	timeWindowChangesDays = 0
	startDate := time.Now() // first time of today
	endDate := time.Now()   // last time of today
	expectedRequestURI := getExpectedRequestURI(cmdbCi, startDate, endDate, sysparmOffset)

	requestURI := p.getChangeRequestURI(cmdbCi, sysparmOffset)

	s.Equal(expectedRequestURI, requestURI, "RequestURI should be correct")
	loggerObj.AssertExpectations(t)
}

func (s *ChangeTestSuite) TestGetChangeRequestURIDateWindow1Days() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	cmdbCi := "id2"
	sysparmOffset := 0

	timeWindowChangesDays = 1
	startDate := time.Now().Add(-1 * 24 * time.Hour) // first time of yesterday
	endDate := time.Now().Add(1 * 24 * time.Hour)    // last time of tomorrow
	expectedRequestURI := getExpectedRequestURI(cmdbCi, startDate, endDate, sysparmOffset)

	requestURI := p.getChangeRequestURI(cmdbCi, sysparmOffset)

	s.Equal(expectedRequestURI, requestURI, "RequestURI should be correct")
	loggerObj.AssertExpectations(t)
}

func (s *ChangeTestSuite) TestGetChangeRequestURIDateWindow7Days() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	cmdbCi := "id3"
	sysparmOffset := 0

	timeWindowChangesDays = 7
	startDate := time.Now().Add(-7 * 24 * time.Hour) // first time of last week
	endDate := time.Now().Add(7 * 24 * time.Hour)    // last time of next week
	expectedRequestURI := getExpectedRequestURI(cmdbCi, startDate, endDate, sysparmOffset)

	requestURI := p.getChangeRequestURI(cmdbCi, sysparmOffset)

	s.Equal(expectedRequestURI, requestURI, "RequestURI should be correct")
	loggerObj.AssertExpectations(t)
}

func (s *ChangeTestSuite) TestGetChangeRequestURIDateWindowDifferentSysparmOffset() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	cmdbCi := "cioffset"
	sysparmOffset := 5

	timeWindowChangesDays = 7
	startDate := time.Now().Add(-7 * 24 * time.Hour) // first time of last week
	endDate := time.Now().Add(7 * 24 * time.Hour)    // last time of next week
	expectedRequestURI := getExpectedRequestURI(cmdbCi, startDate, endDate, sysparmOffset)

	requestURI := p.getChangeRequestURI(cmdbCi, sysparmOffset)

	s.Equal(expectedRequestURI, requestURI, "RequestURI should be correct")
	loggerObj.AssertExpectations(t)
}

func (s *ChangeTestSuite) TestGetChangesOneChange() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"

	cmdbCi := "1chg"

	sysparmOffset := 0
	requestURI := getTestChangeRequestURI(cmdbCi, sysparmOffset)
	responseText := `{"result":[{"type":"1", "number":"CHG300030", "short_description":"test", "start_date":"2025-05-15 17:00:00", "end_date":"2025-05-15 17:45:00", "sys_id": "1"}]}`
	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	apiCall := fmt.Sprintf("%s%s", serviceNowUrl, requestURI)
	loggerObj.On("Debug", "apiCall: "+apiCall)
	loggerObj.On("Debug", responseText)

	changes, number, errorText := p.getChanges(cmdbCi, sysparmOffset)

	s.Equal("CHG300030", changes[0].Number, "Change number should be the same as in the API result")
	s.Equal(1, number, "Number should be incremented by the number of changes that are received")
	s.Equal("", errorText, "No errors expected")

	loggerObj.AssertExpectations(t)
}

func (s *ChangeTestSuite) TestGetChangesTwoChanges() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"

	cmdbCi := "chg2"

	sysparmOffset := 0
	requestURI := getTestChangeRequestURI(cmdbCi, sysparmOffset)
	responseText := `{"result":[{"type":"1", "number":"CHG300030", "short_description":"test", "start_date":"2025-05-15 17:00:00", "end_date":"2025-05-15 17:45:00", "sys_id": "2"},
                                {"type":"1", "number":"CHG300031", "short_description":"test2", "start_date":"2025-05-15 18:00:00", "end_date":"2025-05-15 18:45:00", "sys_id": "22"}]}`

	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText

	server := simulateHttpRequestToServiceNow(t, responseMap)
	serviceNowUrl = server.URL
	defer server.Close()

	apiCall := fmt.Sprintf("%s%s", serviceNowUrl, requestURI)
	loggerObj.On("Debug", "apiCall: "+apiCall)
	loggerObj.On("Debug", responseText)

	changes, newSysparmOffset, errorText := p.getChanges(cmdbCi, sysparmOffset)

	s.Equal("CHG300030", changes[0].Number, "Change number should be the same as in the API result")
	s.Equal(2, newSysparmOffset, "SysparmOffset should be incremented by the number of changes that are received")
	s.Equal("2", changes[0].SysId, "SysId should be incremented by the number of changes that are received")
	s.Equal("CHG300031", changes[1].Number, "Change number should be the same as in the API result")
	s.Equal("22", changes[1].SysId, "SysId should be the same as in the API result")
	s.Equal("", errorText, "No errors expected")

	loggerObj.AssertExpectations(t)
}

func (s *ChangeTestSuite) TestGetChangesExactAPIWindowSize() {
	t := s.T()

	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"

	cmdbCi := "chg5"

	sysparmOffset := 0
	requestURI := getTestChangeRequestURI(cmdbCi, sysparmOffset)
	responseText := `{"result":[{"type":"1", "number":"CHG300030", "short_description":"test", "start_date":"2025-05-15 17:00:00", "end_date":"2025-05-15 17:45:00", "sys_id": "1"},
                                {"type":"1", "number":"CHG300031", "short_description":"test2", "start_date":"2025-05-15 18:00:00", "end_date":"2025-05-15 18:45:00", "sys_id": "2"},
                                {"type":"1", "number":"CHG300032", "short_description":"test3", "start_date":"2025-05-15 19:00:00", "end_date":"2025-05-15 19:45:00", "sys_id": "3"},
                                {"type":"1", "number":"CHG300033", "short_description":"test4", "start_date":"2025-05-15 20:00:00", "end_date":"2025-05-15 20:45:00", "sys_id": "4"},
                                {"type":"1", "number":"CHG300034", "short_description":"test5", "start_date":"2025-05-15 21:00:00", "end_date":"2025-05-15 21:45:00", "sys_id": "5"}]}`
	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	loggerObj.On("Debug", mock.Anything)

	changes, newSysparmOffet, errorText := p.getChanges(cmdbCi, sysparmOffset)

	s.Equal("CHG300030", changes[0].Number, "Change number should be the same as in the API result")
	s.Equal(5, newSysparmOffet, "New sysparmOffset should be incremented by the number of changes that are received")
	s.Equal("CHG300031", changes[1].Number, "Change number should be the same as in the API result")
	s.Equal("CHG300032", changes[2].Number, "Change number should be the same as in the API result")
	s.Equal("CHG300033", changes[3].Number, "Change number should be the same as in the API result")
	s.Equal("CHG300034", changes[4].Number, "Change number should be the same as in the API result")
	s.Equal("", errorText, "No errors expected")

	loggerObj.AssertExpectations(t)
}

func (s *ChangeTestSuite) TestGetChangesNoChange() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"

	cmdbCi := "ci1"

	sysparmOffset := 0
	requestURI := getTestChangeRequestURI(cmdbCi, sysparmOffset)
	responseText := "{\"result\":[]}"
	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	apiCall := fmt.Sprintf("%s%s", server.URL, requestURI)

	loggerObj.On("Debug", fmt.Sprintf("apiCall: %s", apiCall))
	loggerObj.On("Debug", responseText)
	loggerObj.On("Info", "No changes found")

	changes, newPointer, errorText := p.getChanges(cmdbCi, 0)

	s.Equal(0, len(changes), "No changes should be found")
	s.Equal(0, newPointer, "New value for offset should be 0")
	s.Equal("No changes found", errorText, "Correct error text")
	loggerObj.AssertExpectations(t)
}

func (s *ChangeTestSuite) TestGetChangesNoJSON() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	responseText := "!"
	expectedErrorText := "Error in json.Unmarshal: invalid character '!' looking for beginning of value (!)"

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"

	cmdbCi := "ci5"
	sysparmOffset := 0
	requestURI := getTestChangeRequestURI(cmdbCi, sysparmOffset)
	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	apiCall := fmt.Sprintf("%s%s", server.URL, requestURI)

	loggerObj.On("Debug", fmt.Sprintf("apiCall: %s", apiCall))
	loggerObj.On("Debug", responseText)
	loggerObj.On("Error", expectedErrorText)

	_, _, errorText := p.getChanges(cmdbCi, 0)

	s.Equal(expectedErrorText, errorText, "Correct error text")
	loggerObj.AssertExpectations(t)
}

func (s *ChangeTestSuite) TestGetChangesAPIServerDown() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"

	sysparmOffset := 0
	cmdbCi := "ci6"
	responseText := "<html><body>Server down!</body></html>"
	requestURI := getTestChangeRequestURI(cmdbCi, sysparmOffset)
	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()

	apiCall := fmt.Sprintf("%s%s", server.URL, requestURI)
	expectedErrorText := "ServiceNow API server is down"

	loggerObj.On("Debug", fmt.Sprintf("apiCall: %s", apiCall))
	loggerObj.On("Debug", responseText)
	loggerObj.On("Error", expectedErrorText)
	defer server.Close()
	serviceNowUrl = server.URL

	_, _, errorText := p.getChanges(cmdbCi, sysparmOffset)
	s.Equal(expectedErrorText, errorText, "Correct error text")
	loggerObj.AssertExpectations(t)
}

func (s *ChangeTestSuite) TestParseChange() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	var change_servicenow = ChangeServiceNow{
		Type:             "1",
		Number:           "CHG12345",
		EndDate:          "2025-05-16 23:59:59",
		ShortDescription: "Test",
		StartDate:        "2025-05-16 08:00:00",
		SysId:            "sys_id",
	}

	debugText := fmt.Sprintf("Change: Type: %s, Number: %s, Short description: %s, Start Date: %s, End Date: %s, SysId: %s",
		change_servicenow.Type,
		change_servicenow.Number,
		change_servicenow.ShortDescription,
		change_servicenow.StartDate,
		change_servicenow.EndDate,
		change_servicenow.SysId)
	loggerObj.On("Debug", debugText)

	chg, errorText := p.parseChange(change_servicenow)

	s.Equal(change_servicenow.Type, chg.Type, "Change type should be the same")
	s.Equal(change_servicenow.Number, chg.Number, "Change number should be the same")
	s.Equal(time.Date(2025, 05, 16, 23, 59, 59, 0, time.UTC), chg.EndDate, "Change end date should be the same")
	s.Equal(change_servicenow.ShortDescription, chg.ShortDescription, "Change short description should be the same")
	s.Equal(time.Date(2025, 05, 16, 8, 0, 0, 0, time.UTC), chg.StartDate, "Change start date should be the same")
	s.Equal(change_servicenow.SysId, chg.SysId, "Change sys_id should be the same")
	s.Equal("", errorText, "No errors expected")
	loggerObj.AssertExpectations(t)
}

func TestChangeMethods(t *testing.T) {
	suite.Run(t, new(ChangeTestSuite))
}

func testAllowedCIStatus(s *CheckCITestSuite, status string) {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	var ci = CmdbServiceNow{
		InstallStatus: status,
		Name:          "whatever",
	}

	checkString := p.checkCI(ci)

	s.Equal("", checkString, "Installed state should be accepted")
	loggerObj.AssertExpectations(t)
}

func testNotAllowedCIStatus(s *CheckCITestSuite, status string) {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	var ci = CmdbServiceNow{
		InstallStatus: status,
		Name:          "whatever",
	}

	checkString := p.checkCI(ci)
	expectedCheckString := fmt.Sprintf("Invalid install status (%s) for CI whatever", status)
	s.Equal(expectedCheckString, checkString, "Other states should not be accepted")

	loggerObj.AssertExpectations(t)
}

func (s *CheckCITestSuite) TestCheckCICorrectState() {
	const installed = "1"
	const inMaintenance = "3"
	const pendingInstall = "4"
	const pendingRepair = "5"

	validStates := []string{installed, inMaintenance, pendingInstall, pendingRepair}
	for _, state := range validStates {
		testAllowedCIStatus(s, state)
	}
}

func (s *CheckCITestSuite) TestCheckCIInvalidState() {
	invalidStates := []string{"0", "2", "6"}
	for _, state := range invalidStates {
		testNotAllowedCIStatus(s, state)
	}
}

func TestCheckCI(t *testing.T) {
	suite.Run(t, new(CheckCITestSuite))
}

func (s *CheckChangeTestSuite) TestCheckChangeCorrectTime() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	timezone = "UTC"
	currentTime := time.Now()
	startDate := currentTime.Add(-5 * time.Minute)
	endDate := currentTime.Add(time.Hour * 2).Add(time.Microsecond * 50)

	var change = Change{
		Type:             "1",
		Number:           "CHG12345",
		EndDate:          endDate,
		ShortDescription: "Test",
		StartDate:        startDate,
	}

	expectedErrorText := ""

	checkString, remainingTime := p.checkChange(change)

	expectedRemainingTime := time.Duration(time.Hour * 2).Truncate(time.Second)

	s.Equal(expectedErrorText, checkString, "Change that is started between start date and end date should be accepted")
	s.Equal(expectedRemainingTime, remainingTime.Truncate(time.Second), "Remaining time should be correct")
	loggerObj.AssertExpectations(t)
}

func testChangeTimeIncorrect(s *CheckChangeTestSuite, currentTime time.Time, startDate time.Time, endDate time.Time, situation string) {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	timezone = "UTC"

	var change = Change{
		Type:             "1",
		Number:           "CHG12345",
		EndDate:          endDate,
		ShortDescription: "Test",
		StartDate:        startDate,
		SysId:            "1",
	}

	expectedErrorText := fmt.Sprintf("Change %s (%s) is not in the valid time range. start date: %s and end date: %s (current date: %s)",
		change.Number,
		change.ShortDescription,
		p.getLocalTime(change.StartDate),
		p.getLocalTime(change.EndDate),
		p.getLocalTime(currentTime))
	loggerObj.On("Debug", expectedErrorText)

	checkString, _ := p.checkChange(change)

	assert.Equal(t, expectedErrorText, checkString, "Change that is started "+situation+" should not be accepted")
	loggerObj.AssertExpectations(t)

}

func (s *CheckChangeTestSuite) TestCheckChangeTooEarly() {
	currentTime := time.Now()
	startDate := currentTime.Add(time.Hour)
	endDate := currentTime.Add(time.Hour * 2)

	testChangeTimeIncorrect(s, currentTime, startDate, endDate, "too early")
}

func (s *CheckChangeTestSuite) TestCheckChangeTooLate() {
	timezone = "UTC"
	currentTime := time.Now()
	startDate := currentTime.Add(-2 * time.Hour)
	endDate := currentTime.Add(-1 * time.Hour)

	testChangeTimeIncorrect(s, currentTime, startDate, endDate, "too late")
}

func TestCheckChange(t *testing.T) {
	suite.Run(t, new(CheckChangeTestSuite))
}

func (s *PluginHelperMethodsTestSuite) TestProcessCIWithValidCI() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	loggerObj.On("Debug", mock.Anything)

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"

	ciName := "app-demoapp"

	requestURI := getTestCIRequestURI(ciName)
	responseText := fmt.Sprintf(`{"result":[{"install_status":"1", "name":"%s", "sys_id": "1"}]}`, ciName)
	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	errorString, sysId := p.processCI(ciName)

	s.Equal("", errorString, "Errorstring should be empty")
	s.Equal("1", sysId, "sys_id should be 1")
	// Don't assert logging, is done in other tests
}

func (s *PluginHelperMethodsTestSuite) TestProcessCIWithoutValidCI() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"

	ciName := "app-demoapp"
	requestURI := getTestCIRequestURI(ciName)
	responseText := `{"result":[]}`
	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	expectedErrorText := fmt.Sprintf("No CI with name %s found", ciName)
	loggerObj.On("Debug", mock.Anything)
	loggerObj.On("Error", expectedErrorText)

	errorString, sysId := p.processCI(ciName)

	s.Equal(expectedErrorText, errorString, "Errorstring should be correct")
	s.Equal("", sysId, "sys_id should be empty")
	// Don't assert logging, is done in other tests
}

func getTestCIRequestURI(ciName string) string {
	requestURI := fmt.Sprintf(`/api/now/table/cmdb_ci?name=%s&sysparm_fields=install_status,name,sys_id`, ciName)
	return requestURI
}

func getTestChangeRequestURI(cmdbCi string, sysparmOffset int) string {

	// I know, next part is awful.
	//
	// Reason: search for start_date > 2025-05-20 00:00:00 and end_date < 2025-05-22 23:59:59 is not possible without sysparm_query,
	// and to make it possible you need to use %20 for space, %5e for ^, etc.
	//
	// When we can search for number=CHG0030002 we don't need the time window anymore and then this part will be better
	// readable.
	//
	// Sorry for now!

	startDate := time.Now().Add(-1 * time.Hour * 24 * time.Duration(timeWindowChangesDays))
	startYear := startDate.Year()
	startMonth := startDate.Month()
	startDay := startDate.Day()

	endDate := time.Now().Add(+1 * time.Hour * 24 * time.Duration(timeWindowChangesDays))
	endYear := endDate.Year()
	endMonth := endDate.Month()
	endDay := endDate.Day()

	requestURIBegin := "/api/now/table/change_request?sysparm_query=cmdb_ci%3d" + cmdbCi + "%5estate%3d%2d1%5ephase%3drequested%5eapproval%3dapproved%5eactive%3dtrue%5eGOTOstart_date%3e"
	startDateURI := fmt.Sprintf("%04d%%2d%02d%%2d%02d", startYear, startMonth, startDay)
	endDateURI := fmt.Sprintf("%04d%%2d%02d%%2d%02d", endYear, endMonth, endDay)
	requestURIEndDate := "%2000%3a00%3a00%5eGOTOend_date%3c"
	requestURIRest := "%2023%3a59%3a59&sysparm_fields=type,number,short_description,start_date,end_date,sys_id&sysparm_limit=5&sysparm_offset="

	sysparmOffsetString := fmt.Sprintf("%d", sysparmOffset)
	requestURI := requestURIBegin + startDateURI + requestURIEndDate + endDateURI + requestURIRest + sysparmOffsetString

	return requestURI
}

func (s *PluginHelperMethodsTestSuite) TestProcessChangesWithChange() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"
	timezone = "UTC"

	currentTime := time.Now()
	startDate := currentTime.Add(-5 * time.Minute)
	endDate := currentTime.Add(time.Hour * 2)

	cmdbCi := "a7b5e1"
	ciName := "app-demoapp"
	sysparmOffset := 0

	requestURI := getTestChangeRequestURI(cmdbCi, sysparmOffset)
	responseText := fmt.Sprintf(`{"result":[{"type":"1", "number":"CHG300030", "short_description":"test", "start_date":"%s", "end_date":"%s", "sys_id": "1"}]}`,
		testConvertTimeToString(startDate),
		testConvertTimeToString(endDate))
	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	loggerObj.On("Debug", mock.Anything)

	errorString, changeRemainingTime, validChange := p.processChanges(ciName, cmdbCi)

	s.Equal("", errorString, "Errorstring should be empty")
	if changeRemainingTime.Minutes() < 40 {
		s.Fail("changeRemainingTime is too small, less than 40 minutes")
	}

	s.Equal("CHG300030", validChange.Number, "Numbers must be equal")
	s.Equal("test", validChange.ShortDescription, "Short descriptions must be equal")
	// Don't assert logging, is done in other tests
}

func (s *PluginHelperMethodsTestSuite) TestProcessChangesWithoutChange() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	timezone = "UTC"
	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"

	sysparmOffset := 0
	ciName := "app-demoapp1"
	cmdbCi := "a7b5e2"

	requestURI := getTestChangeRequestURI(cmdbCi, sysparmOffset)
	responseText := `{"result":[]}`
	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	expectedInfoString := "No changes found"

	loggerObj.On("Debug", mock.Anything)
	loggerObj.On("Info", expectedInfoString)

	errorString, _, _ := p.processChanges(ciName, cmdbCi)

	s.Equal(expectedInfoString, errorString, "Errorstring should be correct")

	// Don't assert logging, is done in other tests
}

func (s *PluginHelperMethodsTestSuite) TestProcessChangesWithOneInvalidChange() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	timezone = "UTC"
	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"

	sysparmOffset := 0
	ciName := "app-demoapp1"
	cmdbCi := "a7b5e2"

	requestURI := getTestChangeRequestURI(cmdbCi, sysparmOffset)
	responseText := `{"result":[{"type":"1", "number":"CHG300030", "short_description":"test", "start_date":"2025-05-15 19:00:00", "end_date":"2025-05-15 19:45:00", "sys_id": "abc123def567"}]}`
	var responseMap = make(map[string]string)
	responseMap[requestURI] = responseText
	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	expectedInfoString := "No valid change found"

	loggerObj.On("Debug", mock.Anything)
	loggerObj.On("Info", expectedInfoString)

	errorString, _, _ := p.processChanges(ciName, cmdbCi)

	s.Equal(expectedInfoString, errorString, "Errorstring should be correct")

	// Don't assert logging, is done in other tests
}

func (s *PluginHelperMethodsTestSuite) TestProcessChangesTwoAPIWindowsWithValidChange() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"
	timezone = "UTC"

	currentTime := time.Now()
	startDate := currentTime.Add(-5 * time.Minute)
	endDate := currentTime.Add(time.Hour * 2)

	ciName := "app-demoapp2"
	cmdbCi := "ci1"

	var responseMap = make(map[string]string)

	sysparmOffset := 0
	requestURI := getTestChangeRequestURI(cmdbCi, sysparmOffset)
	responseText := `{"result":[{"type":"1", "number":"CHG300030", "short_description":"test", "start_date":"2025-05-15 17:00:00", "end_date":"2025-05-15 17:45:00","sys_id": "abc123def561"},
                                {"type":"1", "number":"CHG300031", "short_description":"test2", "start_date":"2025-05-15 18:00:00", "end_date":"2025-05-15 18:45:00","sys_id": "abc123def562"},
                                {"type":"1", "number":"CHG300032", "short_description":"test3", "start_date":"2025-05-15 19:00:00", "end_date":"2025-05-15 19:45:00","sys_id": "abc123def563"},
                                {"type":"1", "number":"CHG300033", "short_description":"test4", "start_date":"2025-05-15 20:00:00", "end_date":"2025-05-15 20:45:00","sys_id": "abc123def564"},
                                {"type":"1", "number":"CHG300034", "short_description":"test5", "start_date":"2025-05-15 21:00:00", "end_date":"2025-05-15 21:45:00","sys_id": "abc123def565"}]}`
	responseMap[requestURI] = responseText

	sysparmOffset = 5
	requestURI = getTestChangeRequestURI(cmdbCi, sysparmOffset)
	responseText = fmt.Sprintf(`{"result":[{"type":"1", "number":"CHG300040", "short_description":"test6", "start_date":"2025-05-15 17:00:00", "end_date":"2025-05-15 17:45:00","sys_id": "abc123def570"},
                                {"type":"1", "number":"CHG300041", "short_description":"test7", "start_date":"2025-05-15 18:00:00", "end_date":"2025-05-15 18:45:00","sys_id": "abc123def571"},
                                {"type":"1", "number":"CHG300042", "short_description":"test8", "start_date":"2025-05-15 19:00:00", "end_date":"2025-05-15 19:45:00","sys_id": "abc123def572"},
                                {"type":"1", "number":"CHG300043", "short_description":"test9", "start_date":"2025-05-15 20:00:00", "end_date":"2025-05-15 20:45:00","sys_id": "abc123def573"},
                                {"type":"1", "number":"CHG300044", "short_description":"test10", "start_date":"%s", "end_date":"%s","sys_id": "abc123def574"}]}`,
		testConvertTimeToString(startDate),
		testConvertTimeToString(endDate))
	responseMap[requestURI] = responseText

	server := simulateHttpRequestToServiceNow(t, responseMap)
	defer server.Close()
	serviceNowUrl = server.URL

	loggerObj.On("Debug", mock.Anything)

	errorString, changeRemainingTime, validChange := p.processChanges(ciName, cmdbCi)

	s.Equal("", errorString, "Errorstring should be empty")
	if changeRemainingTime.Minutes() < 40 {
		s.Fail("changeRemainingTime is too small, less than 40 minutes")
	}

	s.Equal("CHG300044", validChange.Number, "Numbers must be correct")
	s.Equal("test10", validChange.ShortDescription, "Short descriptions must be correct")
	s.Equal("abc123def574", validChange.SysId, "sys_id must be correct")
	loggerObj.AssertExpectations(t)
}

func (s *PluginHelperMethodsTestSuite) TestProcessChangesTwoAPIWindowsErrorInSecondBatch() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	serviceNowUsername = "testUser"
	serviceNowPassword = "testPassword"
	timezone = "UTC"

	ciName := "app-demoapp3"
	cmdbCi := "id2"

	var responseMap = make(map[string]string)

	sysparmOffset := 0
	requestURI := getTestChangeRequestURI(cmdbCi, sysparmOffset)
	responseText := `{"result":[{"type":"1", "number":"CHG300030", "short_description":"test", "start_date":"2025-05-15 17:00:00", "end_date":"2025-05-15 17:45:00"},
                                {"type":"1", "number":"CHG300031", "short_description":"test2", "start_date":"2025-05-15 18:00:00", "end_date":"2025-05-15 18:45:00"},
                                {"type":"1", "number":"CHG300032", "short_description":"test3", "start_date":"2025-05-15 19:00:00", "end_date":"2025-05-15 19:45:00"},
                                {"type":"1", "number":"CHG300033", "short_description":"test4", "start_date":"2025-05-15 20:00:00", "end_date":"2025-05-15 20:45:00"},
                                {"type":"1", "number":"CHG300034", "short_description":"test5", "start_date":"2025-05-15 21:00:00", "end_date":"2025-05-15 21:45:00"}]}`
	responseMap[requestURI] = responseText

	sysparmOffset = 5
	requestURI = getTestChangeRequestURI(cmdbCi, sysparmOffset)
	responseText = `{"result":[]}`
	responseMap[requestURI] = responseText

	server := simulateHttpRequestToServiceNow(t, responseMap)
	serviceNowUrl = server.URL
	defer server.Close()
	serviceNowUrl = server.URL

	expectedInfoString := "No changes found"

	loggerObj.On("Debug", mock.Anything)
	loggerObj.On("Info", expectedInfoString)

	errorString, changeRemainingTime, _ := p.processChanges(ciName, cmdbCi)

	s.Equal(expectedInfoString, errorString, "Errorstring should be correct")
	s.Equal(changeRemainingTime.Minutes(), 0.0, "changeRemainingTime is incorrect, different from 0")

	loggerObj.AssertExpectations(t)
}

func getTestARApp() (api.AccessRequest, argocd.Application) {
	var ar api.AccessRequest
	var requestedRole api.TargetRole
	var app argocd.Application

	requestedRole.TemplateRef.Name = "administrator"

	ar.Spec.Subject.Username = "Test User"
	ar.Spec.Role = requestedRole
	ar.Spec.Application.Namespace = "argocd"
	ar.Spec.Application.Name = "demoapp"
	ar.Spec.Duration.Duration = 4 * time.Hour

	app.Name = "demoapp"

	var m = make(map[string]string)
	m["ci-name"] = "app-demoapp"
	app.Labels = m

	return ar, app
}

func (s *ServiceNowTestSuite) TestPostNote() {
	serviceNowUrl = "https://servicenow.com"
	requestURI := "/api/now/table/change_request/CHG0030002"
	noteText := `{"work_notes": "This is the text of the note"}`
	responseText := `{"number": "CHG00300002", "other_fields": "whatever"}`

	testPatchServiceNowAPINormalRequest(s, requestURI, noteText, responseText)
}

func (s *PublicMethodsTestSuite) TestInit() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	loggerObj.On("Debug", "This is a call to the Init method")

	result := p.Init()

	s.Equal(nil, result, "Init correctly executed")
	loggerObj.AssertExpectations(t)
}

func configureTestEnvWithTestData(t *testing.T, loggerObj *MockedLogger, installStatus string, addChange bool) *httptest.Server {
	_ = os.Setenv("TIMEZONE", "UTC")
	_ = os.Setenv("SERVICENOW_URL", "https://example.com")

	secretName := "servicenow-secret"
	namespace := "argocd-ephemeral-access"
	genericUsername := "serviceNowUsername"
	genericPassword := "serviceNowPassword"
	unittest = true // don't initialize k8sconfig/k8sclientset

	k8sclientset = testclient.NewClientset()
	setConfigMap(namespace, ExclusionsConfigMapName, "exclusion-roles", "incidentmanagers")
	setSecret(namespace, secretName, genericUsername, genericPassword)

	currentTime := time.Now()
	startDate := currentTime.Add(-5 * time.Minute)
	endDate := currentTime.Add(2 * time.Hour)
	startDateString := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", startDate.Year(), startDate.Month(), startDate.Day(), startDate.Hour(), startDate.Minute(), startDate.Second())
	endDateString := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d", endDate.Year(), endDate.Month(), endDate.Day(), endDate.Hour(), endDate.Minute(), endDate.Second())

	loggerObj.On("Debug", mock.Anything)
	loggerObj.On("Info", mock.Anything)

	var responseMap = make(map[string]string)

	requestURI := getTestCIRequestURI("app-demoapp")
	responseText := fmt.Sprintf(`{"result": [{"install_status": "%s", "name": "app-demoapp", "sys_id": "5"}]}`, installStatus)
	responseMap[requestURI] = responseText

	sysparmOffset := 0
	requestURI = getTestChangeRequestURI("5", sysparmOffset)
	if addChange {
		responseText = fmt.Sprintf(`{"result":[{"type":"1", "number":"CHG300030", "short_description":"valid change", "start_date":"%s", "end_date":"%s", "sys_id":"1"}]}`, startDateString, endDateString)
	} else {
		responseText = `{"result": []}`
	}
	responseMap[requestURI] = responseText

	requestURI = "/api/now/table/change_request/1" // for post, but whatever. We don't do anything with the response...
	responseText = `{"whatever":"true"}`
	responseMap[requestURI] = responseText

	server := simulateHttpRequestToServiceNow(t, responseMap)
	_ = os.Setenv("SERVICENOW_URL", server.URL)

	return server
}

func (s *PublicMethodsTestSuite) TestGrantAccess() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	server := configureTestEnvWithTestData(t, loggerObj, correctCMDBInstallStatus, addChange)
	defer server.Close()
	loggerObj.On("Info", "Call to GrantAccess: username: Test User, role: administrator, application: [argocd]demoapp, duration: 4h0m0s")

	ar, app := getTestARApp()

	response, err := p.GrantAccess(&ar, &app)

	s.Equal(plugin.GrantStatusGranted, response.Status, "Status should be granted")
	s.Equal(nil, err, "Error should be nil")
	if !strings.Contains(response.Message, "Granted access") {
		t.Errorf("%s should contain text Granted access", response.Message)
	}
	if !strings.Contains(response.Message, "change") {
		t.Errorf("%s should contain text change", response.Message)
	}
	loggerObj.AssertExpectations(t)
}

func (s *PublicMethodsTestSuite) TestGrantAccessExclusionRole() {
	t := s.T()

	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	server := configureTestEnvWithTestData(t, loggerObj, correctCMDBInstallStatus, addChange)
	defer server.Close()

	ar, app := getTestARApp()
	ar.Spec.Role.TemplateRef.Name = "incidentmanagers"
	loggerObj.On("Warn", mock.Anything)

	response, err := p.GrantAccess(&ar, &app)

	s.Equal(plugin.GrantStatusGranted, response.Status, "Status should be granted")
	s.Equal(nil, err, "Error should be nil")
	if !strings.Contains(response.Message, "Granted access") {
		t.Errorf("%s should contain text Granted access", response.Message)
	}
	if !strings.Contains(response.Message, "exclusion role") {
		t.Errorf("%s should contain text exclusion role", response.Message)
	}
	loggerObj.AssertExpectations(t)
}

func (s *PublicMethodsTestSuite) TestGrantAccessNoCIName() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	server := configureTestEnvWithTestData(t, loggerObj, correctCMDBInstallStatus, addChange)
	defer server.Close()

	errorText := "No CI name found: expected label with name ci-name in application demoapp"
	loggerObj.On("Error", errorText)

	ar, app := getTestARApp()
	var m = make(map[string]string)
	m["ci-name"] = "\"\""
	app.Labels = m

	response, err := p.GrantAccess(&ar, &app)

	s.Equal(errorText, response.Message, "Response message should be correct")
	s.Equal(plugin.GrantStatusDenied, response.Status, "Response status should be correct")
	s.Equal(nil, err, "Error should be nil")

	loggerObj.AssertExpectations(t)
}

func (s *PublicMethodsTestSuite) TestGrantAccessNoServiceNowURL() {
	t := s.T()

	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	server := configureTestEnvWithTestData(t, loggerObj, correctCMDBInstallStatus, addChange)
	defer server.Close()
	_ = os.Setenv("SERVICENOW_URL", "")

	expectedErrorText := "No Service Now URL given (environment variable SERVICENOW_URL is empty)"
	loggerObj.On("Error", expectedErrorText)

	ar, app := getTestARApp()

	response, err := p.GrantAccess(&ar, &app)

	s.Equal(expectedErrorText, response.Message, "Response message should be correct")
	s.Equal(plugin.GrantStatusDenied, response.Status, "Response status should be correct")
	s.Equal(nil, err, "Error should be nil")

	loggerObj.AssertExpectations(t)
}

func (s *PublicMethodsTestSuite) TestGrantAccessIncorrectCI() {
	t := s.T()
	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	ar, app := getTestARApp()
	invalidInstallStatus := "-1"

	server := configureTestEnvWithTestData(t, loggerObj, invalidInstallStatus, addChange)
	defer server.Close()

	errorText := fmt.Sprintf("Invalid install status (%s) for CI app-demoapp", invalidInstallStatus)

	loggerObj.On("Error", "Access Denied for Test User : "+errorText)
	response, err := p.GrantAccess(&ar, &app)

	s.Equal(errorText, response.Message, "Response message should be correct")
	s.Equal(plugin.GrantStatusDenied, response.Status, "Response status should be correct")
	s.Equal(nil, err, "Error should be nil")

	loggerObj.AssertExpectations(t)
}

func (s *PublicMethodsTestSuite) TestGrantAccessNoChange() {
	t := s.T()

	p, loggerObj := testGetPlugin()
	testResetEnvVar()

	ar, app := getTestARApp()

	dontAddChange := !addChange
	server := configureTestEnvWithTestData(t, loggerObj, correctCMDBInstallStatus, dontAddChange)
	defer server.Close()

	loggerObj.On("Error", mock.Anything)
	loggerObj.On("Warn", "Access Denied for Test User, role administrator: No changes found")
	response, err := p.GrantAccess(&ar, &app)

	s.Equal("No changes found", response.Message, "Response message should be correct")
	s.Equal(plugin.GrantStatusDenied, response.Status, "Response status should be correct")
	s.Equal(nil, err, "Error should be nil")
}

func (s *PublicMethodsTestSuite) TestRevokeAccess() {
	p, _ := testGetPlugin()

	var ar api.AccessRequest
	var app argocd.Application

	var expectedResponse *plugin.RevokeResponse = nil

	response, err := p.RevokeAccess(&ar, &app)
	s.Equal(expectedResponse, response, "Revoke Access is not used, expect nil")
	s.Equal(nil, err, "Error should be nil")
}

func TestPublicMethods(t *testing.T) {
	suite.Run(t, new(PublicMethodsTestSuite))
}
