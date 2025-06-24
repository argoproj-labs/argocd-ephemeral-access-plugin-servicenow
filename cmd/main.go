package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"encoding/json"
	"net/http"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	argocd "github.com/argoproj-labs/argocd-ephemeral-access/api/argoproj/v1alpha1"
	api "github.com/argoproj-labs/argocd-ephemeral-access/api/ephemeral-access/v1alpha1"
	"github.com/hashicorp/go-hclog"

	"github.com/argoproj-labs/argocd-ephemeral-access/pkg/log"
	"github.com/argoproj-labs/argocd-ephemeral-access/pkg/plugin"
	goPlugin "github.com/hashicorp/go-plugin"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ServiceNowPlugin struct {
	Logger hclog.Logger
}

type CmdbServiceNow struct {
	InstallStatus string `json:"install_status"`
	Name          string `json:"name"`
	SysId         string `json:"sys_id"`
}

type CmdbResultsServiceNowType struct {
	Result []*CmdbServiceNow `json:"result"`
}

type ChangeServiceNow struct {
	Type             string `json:"type"`
	Number           string `json:"number"`
	EndDate          string `json:"end_date"`
	ShortDescription string `json:"short_description"`
	StartDate        string `json:"start_date"`
	SysId            string `json:"sys_id"`
}

type Change struct {
	Type             string
	Number           string
	EndDate          time.Time
	ShortDescription string
	StartDate        time.Time
	SysId            string
}

type ChangeResultsServicenow struct {
	Result []*ChangeServiceNow `json:"result"`
}

const SysparmLimit = 5
const ExclusionsConfigMapName = "controller-cm"

var unittest = false

var serviceNowUrl string
var serviceNowUsername string
var serviceNowPassword string
var ciLabel string
var exclusionRoles []string
var timezone string
var timeWindowChangesDays int
var k8sconfig *rest.Config
var k8sclientset kubernetes.Interface

var ephemeralAccessPluginNamespace string

func (p *ServiceNowPlugin) getEnvVarWithoutDefault(envVarName string, errorTextToReturn string) (string, string) {
	errorText := ""

	returnValue := os.Getenv(envVarName)
	if returnValue == "" {
		p.Logger.Error(errorTextToReturn)
		errorText = errorTextToReturn
	}
	return returnValue, errorText
}

func (p *ServiceNowPlugin) getEnvVarWithDefault(envVarName string, envVarDefault string) string {
	returnValue := os.Getenv(envVarName)
	if returnValue == "" {
		p.Logger.Debug(fmt.Sprintf("Environment variable %s is empty, assuming %s", envVarName, envVarDefault))
		returnValue = envVarDefault
	}
	return returnValue
}

func (p *ServiceNowPlugin) getLocalTime(t time.Time) string {
	loc, _ := time.LoadLocation(timezone)

	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d",
		t.In(loc).Year(),
		t.In(loc).Month(),
		t.In(loc).Day(),
		t.In(loc).Hour(),
		t.In(loc).Minute(),
		t.In(loc).Second())
}

func (p *ServiceNowPlugin) convertTime(timestring string) (time.Time, string) {

	goTimeString := strings.ReplaceAll(timestring, " ", "T") + "Z"
	var goTime time.Time

	err := goTime.UnmarshalText([]byte(goTimeString))
	errorText := ""
	if err != nil {
		errorText = "Error in converting " + timestring + " to go Time: " + err.Error()
		p.Logger.Error(errorText)
	}

	return goTime, errorText
}

func (p *ServiceNowPlugin) convertToInt(context string, s string, def int) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		errorText := fmt.Sprintf("Incorrect value, %s in %s: should be a number, assuming %d", s, context, def)
		p.Logger.Error(errorText)
		i = def
	}
	return i
}

func (p *ServiceNowPlugin) getK8sConfig() string {
	var err error
	errorText := ""

	if !unittest {
		k8sconfig, err = rest.InClusterConfig()
		if err != nil {
			errorText = "Error in getK8sConfig, rest.InClusterConfig: " + err.Error()
		} else {

			k8sclientset, err = kubernetes.NewForConfig(k8sconfig)
			if err != nil {
				errorText = "Error in getK8sConfig, kubernetes.NewForConfig: " + err.Error()
			}
		}
	}

	return errorText
}

func (p *ServiceNowPlugin) getCredentialsFromSecret(namespace string, secretName string, usernameKey string, passwordKey string) (string, string, string) {
	p.Logger.Debug(fmt.Sprintf("Get credentials from secret [%s]%s...", namespace, secretName))
	errorText := ""

	secret, err := k8sclientset.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		errorText = fmt.Sprintf("Error getting secret %s, does secret exist in namespace %s? Error: %s", secretName, namespace, err.Error())
		p.Logger.Error(errorText)
	}

	return string(secret.Data[usernameKey]), string(secret.Data[passwordKey]), errorText
}

func (p *ServiceNowPlugin) getExclusionsFromConfigMap(namespace string) []string {
	p.Logger.Debug(fmt.Sprintf("Get exclusions from configmap [%s]%s", namespace, ExclusionsConfigMapName))

	exclusions := []string{}

	configmap, err := k8sclientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), ExclusionsConfigMapName, metav1.GetOptions{})
	if err != nil {
		debugText := fmt.Sprintf("Error getting configmap %s, does configmap exist in namespace %s?", ExclusionsConfigMapName, namespace)
		p.Logger.Debug(debugText)
		p.Logger.Debug("No exclusions used")
	} else {
		exclusions = strings.Split(configmap.Data["exclusion-roles"], "\n")
		p.Logger.Debug("Exclusions used: " + configmap.Data["exclusion-roles"])
	}

	return exclusions
}

func (p *ServiceNowPlugin) getGlobalVars() string {
	errorText := p.getK8sConfig()

	serviceNowURLError := ""
	serviceNowCredentialsError := ""

	serviceNowUrl, serviceNowURLError = p.getEnvVarWithoutDefault("SERVICENOW_URL", "No Service Now URL given (environment variable SERVICENOW_URL is empty)")
	timezone = p.getEnvVarWithDefault("TIMEZONE", "UTC")
	ciLabel = p.getEnvVarWithDefault("CI_LABEL", "ci-name")
	ephemeralAccessPluginNamespace = p.getEnvVarWithDefault("EPHEMERAL_ACCESS_EXTENSION_NAMESPACE", "argocd-ephemeral-access")
	exclusionRoles = p.getExclusionsFromConfigMap(ephemeralAccessPluginNamespace)
	timeWindowChangesDays = p.convertToInt("environment variable TIME_WINDOW_CHANGES_DAYS", p.getEnvVarWithDefault("TIME_WINDOW_CHANGES_DAYS", "7"), 7)

	serviceNowUsername, serviceNowPassword, serviceNowCredentialsError = p.getServiceNowCredentials()

	return errorText + serviceNowURLError + serviceNowCredentialsError
}

func (p *ServiceNowPlugin) showRequest(ar *api.AccessRequest, app *argocd.Application) {
	username := ar.Spec.Subject.Username
	role := ar.Spec.Role.TemplateRef.Name
	namespace := ar.Spec.Application.Namespace
	applicationName := ar.Spec.Application.Name
	duration := ar.Spec.Duration.Duration.String()

	infoText := fmt.Sprintf("Call to GrantAccess: username: %s, role: %s, application: [%s]%s, duration: %s", username, role, namespace, applicationName, duration)
	p.Logger.Info(infoText)

	jsonAr, _ := json.Marshal(ar)
	jsonApp, _ := json.Marshal(app)
	p.Logger.Debug("jsonAr: " + string(jsonAr))
	p.Logger.Debug("jsonApp: " + string(jsonApp))
}

func (p *ServiceNowPlugin) createRevokeJob(namespace string, accessrequestName string, jobStartTime time.Time) {
	p.Logger.Debug(fmt.Sprintf("createRevokeJob: %s, %s", namespace, accessrequestName))
	jobName := strings.ReplaceAll("stop-"+accessrequestName, ".", "-")
	cmd := fmt.Sprintf("kubectl delete accessrequest -n argocd %s && kubectl delete cronjob -n argocd %s", accessrequestName, jobName)
	cronjobs := k8sclientset.BatchV1().CronJobs(namespace)

	var backOffLimit int32 = 0

	cronJobSpec := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: fmt.Sprintf("%d %d %d %d *", jobStartTime.Minute(), jobStartTime.Hour(), jobStartTime.Day(), jobStartTime.Month()),
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							ServiceAccountName: "remove-accessrequest-job-sa",
							Containers: []v1.Container{
								{
									Name:    jobName,
									Image:   "bitnami/kubectl:latest",
									Command: []string{"sh", "-c", cmd},
								},
							},
							RestartPolicy: v1.RestartPolicyNever,
						},
					},
					BackoffLimit: &backOffLimit,
				},
			},
		},
	}

	_, err := cronjobs.Create(context.TODO(), cronJobSpec, metav1.CreateOptions{})
	if err != nil {
		p.Logger.Error(fmt.Sprintf("Failed to create K8s job %s in namespace %s: %s.", jobName, namespace, err.Error()))
	} else {
		p.Logger.Info(fmt.Sprintf("Created K8s job %s successfully in namespace %s", jobName, namespace))
	}
}

// Set duration to the time left for this (valid) change, unless original request was
// shorter - then we are forced to use the duration of the original request.
// In an ideal world, the enddate should always be the enddate of the change and the duration always the amount of time
// that remains until that moment.

func (p *ServiceNowPlugin) determineDurationAndRealEndTime(arDuration time.Duration, changeRemainingTime time.Duration, changeEndDate time.Time) (time.Duration, time.Time) {
	var duration time.Duration
	var realEndTime time.Time

	if arDuration > changeRemainingTime {
		duration = changeRemainingTime
		realEndTime = changeEndDate
	} else {
		duration = arDuration
		realEndTime = time.Now().Add(arDuration)
	}

	return duration, realEndTime
}

func (p *ServiceNowPlugin) determineGrantedTextsChange(requesterName string, requestedRole string, validChange Change, remainingTime time.Duration, realEndDate time.Time) (string, string) {

	grantedAccessText := fmt.Sprintf("Granted access for %s: %s change %s (%s), role %s, from %s to %s",
		requesterName,
		validChange.Type,
		validChange.Number,
		validChange.ShortDescription,
		requestedRole,
		time.Now().Truncate(time.Minute),
		realEndDate.Truncate(time.Second).String())

	grantedAccessUIText := fmt.Sprintf("Granted access: change __%s__ (%s), until __%s (%s)__",
		validChange.Number,
		validChange.ShortDescription,
		p.getLocalTime(realEndDate),
		remainingTime.Truncate(time.Second).String())

	grantedAccessServiceNowText := fmt.Sprintf("ServiceNow plugin granted access to %s, for role %s, until %s (%s)",
		requesterName,
		requestedRole,
		p.getLocalTime(realEndDate),
		remainingTime.Truncate(time.Second).String())

	p.Logger.Info(grantedAccessText)
	p.Logger.Debug(grantedAccessUIText)

	return grantedAccessUIText, grantedAccessServiceNowText
}

func (p *ServiceNowPlugin) determineGrantedTextsExclusions(requesterName string, requestedRole string, remainingTime time.Duration, realEndDate time.Time) string {

	grantedAccessText := fmt.Sprintf("Granted access for %s: role %s, from %s to %s (no change, %s is an exclusion role)",
		requesterName,
		requestedRole,
		time.Now().Truncate(time.Minute),
		realEndDate.Truncate(time.Minute),
		requestedRole)

	grantedAccessUIText := fmt.Sprintf("Granted access: %s is an exclusion role, until __%s (%s)__",
		requestedRole,
		p.getLocalTime(realEndDate),
		remainingTime.Truncate(time.Second).String())

	p.Logger.Warn(grantedAccessText)
	p.Logger.Debug(grantedAccessUIText)

	return grantedAccessUIText
}

func (p *ServiceNowPlugin) denyRequest(reason string) (*plugin.GrantResponse, error) {
	return &plugin.GrantResponse{
		Status:  plugin.GrantStatusDenied,
		Message: reason,
	}, nil
}

func (p *ServiceNowPlugin) grantRequest(reason string) (*plugin.GrantResponse, error) {
	return &plugin.GrantResponse{
		Status:  plugin.GrantStatusGranted,
		Message: reason,
	}, nil
}

func (p *ServiceNowPlugin) getServiceNowCredentials() (string, string, string) {
	secretName := p.getEnvVarWithDefault("SERVICENOW_SECRET_NAME", "servicenow-secret")

	return p.getCredentialsFromSecret(ephemeralAccessPluginNamespace, secretName, "username", "password")
}

func (p *ServiceNowPlugin) checkAPIResult(resp *http.Response, body []byte) ([]byte, string) {

	errorText := ""
	if (resp.StatusCode >= 500 && resp.StatusCode <= 599) || strings.Contains(string(body), "<html>") {
		errorText = "ServiceNow API server is down"
	}

	if resp.StatusCode >= 400 && resp.StatusCode <= 499 {
		errorText = "ServiceNow API changed"
	}

	return body, errorText
}

func (p *ServiceNowPlugin) getFromServiceNowAPI(requestURI string) ([]byte, string) {

	apiCall := fmt.Sprintf("%s%s", serviceNowUrl, requestURI)
	p.Logger.Debug("apiCall: " + apiCall)

	req, err := http.NewRequest("GET", apiCall, nil)
	if err != nil {
		errorText := "Error in NewRequest: " + err.Error()
		p.Logger.Error(errorText)
		return []byte{}, errorText
	}

	req.Header.Add("Accept", "application/json")
	req.SetBasicAuth(serviceNowUsername, serviceNowPassword)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errorText := "Error in client.Do: " + err.Error()
		p.Logger.Error(errorText)
		return []byte{}, errorText
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errorText := "Error in io.ReadAll: " + err.Error()
		p.Logger.Error(errorText)
		return []byte{}, errorText
	}

	p.Logger.Debug(string(body))
	return p.checkAPIResult(resp, body)
}

func (p *ServiceNowPlugin) patchServiceNowAPI(requestURI string, data string) ([]byte, string) {

	apiCall := fmt.Sprintf("%s%s", serviceNowUrl, requestURI)
	p.Logger.Debug("apiCall: " + apiCall)
	p.Logger.Debug("Data: " + string(data))

	req, err := http.NewRequest("PATCH", apiCall, strings.NewReader(data))
	if err != nil {
		errorText := "Error in NewRequest: " + err.Error()
		p.Logger.Error(errorText)
		return nil, errorText
	}

	req.Header.Add("Accept", "application/json")
	req.SetBasicAuth(serviceNowUsername, serviceNowPassword)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errorText := "Error in client.Do: " + err.Error()
		p.Logger.Error(errorText)
		return nil, errorText
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errorText := "Error in io.ReadAll: " + err.Error()
		p.Logger.Error(errorText)
		return nil, errorText
	}

	p.Logger.Debug("Body: " + string(body))
	return p.checkAPIResult(resp, body)
}

func (p *ServiceNowPlugin) getCIName(app *argocd.Application) string {
	p.Logger.Debug("Search for " + ciLabel + " in the CMDB...")
	ciName := string(app.Labels[ciLabel])
	p.Logger.Debug(fmt.Sprintf("ciLabel %s found: %s", ciLabel, ciName))

	return ciName
}

func (p *ServiceNowPlugin) getCI(ciName string) (*CmdbServiceNow, string) {

	requestURI := fmt.Sprintf("/api/now/table/cmdb_ci?name=%s&sysparm_fields=install_status,name,sys_id", ciName)
	response, errorText := p.getFromServiceNowAPI(requestURI)
	if errorText != "" {
		return nil, errorText
	}

	var cmdbResults CmdbResultsServiceNowType
	err := json.Unmarshal(response, &cmdbResults)
	if err != nil {
		errorText := fmt.Sprintf("Error in json.Unmarshal: %s (%s)", err.Error(), response)
		p.Logger.Error(errorText)
		return nil, errorText
	}

	if len(cmdbResults.Result) == 0 {
		errorText := fmt.Sprintf("No CI with name %s found", ciName)
		p.Logger.Error(errorText)
		return nil, errorText
	}

	debugText := fmt.Sprintf("InstallStatus: %s, CI name: %s, SysId: %s",
		cmdbResults.Result[0].InstallStatus,
		cmdbResults.Result[0].Name,
		cmdbResults.Result[0].SysId)
	p.Logger.Debug(debugText)

	return cmdbResults.Result[0], ""
}

func (p *ServiceNowPlugin) getChangeRequestURI(ciSysId string, sysparmOffset int) string {
	// See also: https://github.com/argoproj-labs/argocd-ephemeral-access/issues/109
	// Hopefully the Ephemeral Access Extension will be extended with a reference number
	// that can be used to ask directly for the change number.
	//
	// In that case the original request will be extended with the change number and the
	// request will be better readable as well (the dates can only be used with > and <
	// in the sysparam_query, that also needs to have the cmdb_ci sys_id instead of the
	// name and -1 instead of the display name of the state)
	//
	// Original requestURI without test for change number:
	// requestURI := fmt.Sprintf("/api/now/table/change_request?cmdb_ci=%s&state=Implement&phase=Requested&approval=Approved&active=true&sysparm_fields=type,number,short_description,start_date,end_date,sys_id&sysparm_limit=%d&sysparm_offset=%d", ciName, SysparmLimit, SysparmOffset)
	//
	// The reason for the window is to limit the number of changes that have to be
	// processed by the API in large environments.
	window, _ := time.ParseDuration(fmt.Sprintf("%d", timeWindowChangesDays*24) + "h")

	fromDate := time.Now().Add(-window)
	endDate := time.Now().Add(window)

	fromDateString := fmt.Sprintf(`%04d-%02d-%02d 00:00:00`,
		fromDate.Year(),
		fromDate.Month(),
		fromDate.Day())
	endDateString := fmt.Sprintf(`%04d-%02d-%02d 23:59:59`,
		endDate.Year(),
		endDate.Month(),
		endDate.Day())

	selection := fmt.Sprintf("cmdb_ci=%s&state=-1&phase=requested&approval=approved&active=true&GOTOstart_date>%s&GOTOend_date<%s",
		ciSysId,
		fromDateString,
		endDateString)

	// selection should be encoded for url (the rest doesn't matter)
	selection = strings.ReplaceAll(selection, " ", "%20")
	selection = strings.ReplaceAll(selection, "-", "%2d")
	selection = strings.ReplaceAll(selection, ":", "%3a")
	selection = strings.ReplaceAll(selection, "<", "%3c")
	selection = strings.ReplaceAll(selection, "=", "%3d")
	selection = strings.ReplaceAll(selection, ">", "%3e")

	// sysparam_query uses ^ to combine fields instead of &
	selection = strings.ReplaceAll(selection, "&", "%5e")

	otherFields := fmt.Sprintf("sysparm_fields=type,number,short_description,start_date,end_date,sys_id&sysparm_limit=%d&sysparm_offset=%d",
		SysparmLimit,
		sysparmOffset)

	requestURI := "/api/now/table/change_request?sysparm_query=" + selection + "&" + otherFields

	return requestURI
}

func (p *ServiceNowPlugin) getChanges(ciSysId string, sysparmOffset int) ([]*ChangeServiceNow, int, string) {

	requestURI := p.getChangeRequestURI(ciSysId, sysparmOffset)
	response, errorText := p.getFromServiceNowAPI(requestURI)
	if errorText != "" {
		p.Logger.Error(errorText)
		return nil, sysparmOffset, errorText
	}

	var changeResults ChangeResultsServicenow
	err := json.Unmarshal(response, &changeResults)
	if err != nil {
		errorText := fmt.Sprintf("Error in json.Unmarshal: %s (%s)", err.Error(), response)
		p.Logger.Error(errorText)
		return nil, sysparmOffset, errorText
	}

	if len(changeResults.Result) == 0 {
		errorText = "No changes found"
		p.Logger.Info(errorText)
	}

	return changeResults.Result, sysparmOffset + len(changeResults.Result), errorText
}

func (p *ServiceNowPlugin) parseChange(changeServiceNow ChangeServiceNow) (Change, string) {
	var change Change

	p.Logger.Debug(fmt.Sprintf("Change: Type: %s, Number: %s, Short description: %s, Start Date: %s, End Date: %s, SysId: %s",
		changeServiceNow.Type,
		changeServiceNow.Number,
		changeServiceNow.ShortDescription,
		changeServiceNow.StartDate,
		changeServiceNow.EndDate,
		changeServiceNow.SysId))

	errorTextStartDate := ""
	errorTextEndDate := ""

	change.Type = changeServiceNow.Type
	change.Number = changeServiceNow.Number
	change.ShortDescription = changeServiceNow.ShortDescription
	change.StartDate, errorTextStartDate = p.convertTime(changeServiceNow.StartDate)
	change.EndDate, errorTextEndDate = p.convertTime(changeServiceNow.EndDate)
	change.SysId = changeServiceNow.SysId

	return change, errorTextStartDate + errorTextEndDate
}

func (p *ServiceNowPlugin) checkCI(CI CmdbServiceNow) string {
	errorText := ""
	installStatus := CI.InstallStatus
	ciName := CI.Name

	validInstallStatus := []string{
		"1", // Installed
		"3", // In maintenance
		"4", // Pending install
		"5", // Pending repair
	}

	if !slices.Contains(validInstallStatus, installStatus) {
		errorText = fmt.Sprintf("Invalid install status (%s) for CI %s", installStatus, ciName)
	}

	return errorText
}

func (p *ServiceNowPlugin) checkChange(change Change) (string, time.Duration) {
	errorText := ""
	var remainingTime time.Duration
	remainingTime = 0

	currentTime := time.Now()

	if change.EndDate.Before(currentTime) ||
		change.StartDate.After(currentTime) {
		errorText = fmt.Sprintf("Change %s (%s) is not in the valid time range. start date: %s and end date: %s (current date: %s)",
			change.Number,
			change.ShortDescription,
			p.getLocalTime(change.StartDate),
			p.getLocalTime(change.EndDate),
			p.getLocalTime(currentTime))
		p.Logger.Debug(errorText)
	} else {
		remainingTime = time.Until(change.EndDate)
	}

	return errorText, remainingTime
}

func (p *ServiceNowPlugin) processCI(ciName string) (string, string) {
	CI, errorText := p.getCI(ciName)
	if errorText != "" {
		p.Logger.Error(errorText)
		return errorText, ""
	}

	errorText = p.checkCI(*CI)

	return errorText, CI.SysId
}

func (p *ServiceNowPlugin) processChanges(ciName string, ciSysId string) (string, time.Duration, *Change) {
	var SysparmOffset = 0

	serviceNowChanges, SysparmOffset, errorText := p.getChanges(ciSysId, SysparmOffset)
	if errorText != "" {
		var noDuration = 0 * time.Minute
		return errorText, noDuration, nil
	}

	var validChange *Change
	var changeRemainingTime time.Duration
	var remainingTime time.Duration

	for {
		for _, serviceNowChange := range serviceNowChanges {
			change, errorText := p.parseChange(*serviceNowChange)
			if errorText == "" {
				errorText, remainingTime = p.checkChange(change)
				if errorText == "" {
					validChange = &change
					changeRemainingTime = remainingTime
					break
				}
			}
		}

		if validChange != nil {
			break
		} else if len(serviceNowChanges) < SysparmLimit {
			errorText = "No valid change found"
			break
		} else {
			serviceNowChanges, SysparmOffset, errorText = p.getChanges(ciSysId, SysparmOffset)
			if errorText != "" {
				break
			}
		}
	}

	return errorText, changeRemainingTime, validChange
}

func (p *ServiceNowPlugin) postNote(sysId string, noteText string) {
	requestURI := fmt.Sprintf("/api/now/table/change_request/%s", sysId)

	p.patchServiceNowAPI(requestURI, noteText)
}

// Public methods

func (p *ServiceNowPlugin) Init() error {
	p.Logger.Debug("This is a call to the Init method")
	// p.getGlobalVars cannot be put in the Init method: the variables will be lost between different calls

	return nil
}

func (p *ServiceNowPlugin) GrantAccess(ar *api.AccessRequest, app *argocd.Application) (*plugin.GrantResponse, error) {
	p.Logger.Debug("This is a call to the GrantAccess method")
	p.showRequest(ar, app)

	requesterName := ar.Spec.Subject.Username
	requestedRole := ar.Spec.Role.TemplateRef.Name
	namespace := ar.Spec.Application.Namespace
	arName := ar.Name
	arDuration := ar.Spec.Duration.Duration
	applicationName := ar.Spec.Application.Name

	errorText := p.getGlobalVars()
	if errorText != "" {
		p.Logger.Error(errorText)
		return p.denyRequest(errorText)
	}

	if slices.Contains(exclusionRoles, requestedRole) {
		endTime := time.Now().Add(arDuration)
		grantedUIText := p.determineGrantedTextsExclusions(requesterName, requestedRole, arDuration, endTime)

		return p.grantRequest(grantedUIText)
	}

	ciName := p.getCIName(app)
	if ciName == "\"\"" {
		errorText := fmt.Sprintf("No CI name found: expected label with name %s in application %s", ciLabel, applicationName)
		p.Logger.Error(errorText)
		return p.denyRequest(errorText)
	}

	errorString, ciSysId := p.processCI(ciName)
	if errorString != "" {
		p.Logger.Error("Access Denied for " + requesterName + " : " + errorString)
		return p.denyRequest(errorString)
	}

	errorString, changeRemainingTime, validChange := p.processChanges(ciName, ciSysId)

	if errorString == "" {
		duration, endDateTime := p.determineDurationAndRealEndTime(arDuration, changeRemainingTime, validChange.EndDate)
		ar.Spec.Duration.Duration = duration

		// AbortJob is only needed when the end date of the change is earlier than the default for the access request time in
		// the future, otherwise the ArgoCD Ephemeral Access Extension will revoke the permissions
		if arDuration > changeRemainingTime {
			p.createRevokeJob(namespace, arName, validChange.EndDate)
		}

		jsonAr, _ := json.Marshal(ar)
		p.Logger.Debug(string(jsonAr))

		grantedUIText, grantedAccessServiceNowText := p.determineGrantedTextsChange(requesterName, requestedRole, *validChange, duration, endDateTime)

		note := fmt.Sprintf("{\"work_notes\":\"%s\"}", grantedAccessServiceNowText)
		p.postNote(validChange.SysId, note)
		return p.grantRequest(grantedUIText)
	} else {
		p.Logger.Warn(fmt.Sprintf("Access Denied for %s, role %s: %s", requesterName, requestedRole, errorString))
		return p.denyRequest(errorString)
	}
}

func (p *ServiceNowPlugin) RevokeAccess(ar *api.AccessRequest, app *argocd.Application) (*plugin.RevokeResponse, error) {
	return nil, nil
}

func main() {
	logger, err := log.NewPluginLogger()
	if err != nil {
		panic(fmt.Sprintf("Error creating plugin logger: %s", err))
	}

	p := &ServiceNowPlugin{
		Logger: logger,
	}

	srvConfig := plugin.NewServerConfig(p, logger)

	goPlugin.Serve(srvConfig)
}
