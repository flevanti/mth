package mth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	tr "github.com/flevanti/timeranges"
)

type Client struct {
	url         string
	apiUser     string
	apiPassword string
}

type TaskT struct {
	ID              int          `json:"id"`
	Type            string       `json:"type"`
	CustomerID      int          `json:"customerID"`
	GroupName       string       `json:"groupName"`
	ProjectID       int          `json:"projectID"`
	ProjectName     string       `json:"projectName"`
	VersionID       int          `json:"versionID"`
	VersionName     string       `json:"versionName"`
	JobID           int          `json:"jobID"`
	JobName         string       `json:"jobName"`
	EnvironmentID   int          `json:"environmentID"`
	EnvironmentName string       `json:"environmentName"`
	State           string       `json:"state"`
	EnqueuedTime    int64        `json:"enqueuedTime"`
	StartTime       int64        `json:"startTime"`
	EndTime         int64        `json:"endTime"`
	Message         string       `json:"message"`
	OriginatorID    string       `json:"originatorID"`
	RowCount        int          `json:"rowCount"`
	HasHistoricJobs bool         `json:"hasHistoricJobs"`
	JobNames        []string     `json:"jobNames"`
	Tasks           []TaskChildT `json:"tasks"`
}

type TaskChildT struct {
	TaskID        int    `json:"taskID"`
	ParentID      int    `json:"parentID"`
	Type          string `json:"type"`
	JobID         int    `json:"jobID"`
	JobName       string `json:"jobName"`
	JobRevision   int    `json:"jobRevision"`
	JobTimestamp  int64  `json:"jobTimestamp"`
	ComponentID   int    `json:"componentID"`
	ComponentName string `json:"componentName"`
	State         string `json:"state"`
	RowCount      int    `json:"rowCount"`
	StartTime     int64  `json:"startTime"`
	EndTime       int64  `json:"endTime"`
	Message       string `json:"message"`
	TaskBatchID   int    `json:"taskBatchID"`
}

type TasksHistoryT []TaskT

type TaskWrapperT struct {
	Task               TaskT
	Err                error
	TimeRangeSequence  int
	TimeRangesTotal    int
	TimeRangeStartDate string
	TimeRangeStartTime string
	TimeRangeEndDate   string
	TimeRangeEndTime   string
}

const (
	URLCHECKCONN                  = "rest/v1/group"
	URLGETGROUPS                  = "rest/v1/group"
	URLGETPROJECTS                = "rest/v1/group/name/%s/project"
	URLGETHISTORYBYENDDATERANGE   = "rest/v1/group/name/%s/project/name/%s/task/filter/by/end/range/date/%s/time/%s/to/date/%s/time/%s"
	URLGETHISTORYBYSTARTDATERANGE = "rest/v1/group/name/%s/project/name/%s/task/filter/by/start/range/date/%s/time/%s/to/date/%s/time/%s"
	URLGETHISTORYBYTASKID         = "rest/v1/task/id/%d"
)

func New(baseUrl string, apiUser string, apiPassword string) Client {
	baseUrl = fmt.Sprintf("%s/", strings.TrimRight(baseUrl, "/"))
	return Client{url: baseUrl, apiUser: apiUser, apiPassword: apiPassword}
}

func (c *Client) CheckConnection() error {
	url := fmt.Sprintf("%s%s", c.url, URLCHECKCONN)
	_, err := c.getUrlBody(url)
	return err
}

func (c *Client) getUrlBody(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.apiUser, c.apiPassword)
	cli := &http.Client{}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New(fmt.Sprintf("status code is not 2xx but %d", resp.StatusCode))
	}
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bodyText, nil

}

func (c *Client) GetHistoryByTaskId(taskId int64) TaskWrapperT {
	var url string = fmt.Sprintf("%s%s", c.url, fmt.Sprintf(URLGETHISTORYBYTASKID, taskId))
	var task TaskT

	body, err := c.getUrlBody(url)
	if err != nil {
		return TaskWrapperT{Err: err}
	}
	err = json.Unmarshal(body, &task)
	if err != nil {
		return TaskWrapperT{Err: err}
	}

	return TaskWrapperT{Task: task}

}

func (c *Client) GetHistoryByRange(group string, project string, rangeStart time.Time, rangeEnd time.Time, step time.Duration, useEndDate bool) (chan TaskWrapperT, error) {
	var ch = make(chan TaskWrapperT)
	var timeRanges []tr.TimeRangeT
	var tasks TasksHistoryT
	var task TaskT
	timeRanges = tr.Generate(rangeStart, rangeEnd, step)
	if len(timeRanges) == 0 {
		return nil, errors.New("Unable to generate date start-end ranges")
	}
	go func() {
		defer close(ch)
		// todo check if we want to use CTX
		var i int = 0
		var urlToUse string
		if useEndDate {
			urlToUse = URLGETHISTORYBYENDDATERANGE
		} else {
			urlToUse = URLGETHISTORYBYSTARTDATERANGE
		}
		for _, r := range timeRanges {
			i++
			urlStartDate := r.S.Format("2006-01-02")
			urlStartTime := r.S.Format("15:04")
			urlEndDate := r.E.Format("2006-01-02")
			urlEndTime := r.E.Format("15:04")
			url := fmt.Sprintf("%s%s", c.url, fmt.Sprintf(urlToUse, group, project, urlStartDate, urlStartTime, urlEndDate, urlEndTime))
			body, err := c.getUrlBody(url)
			if err != nil {
				ch <- TaskWrapperT{
					Task: TaskT{},
					Err:  err,
				}
				return
			}
			err = json.Unmarshal(body, &tasks)
			if err != nil {
				ch <- TaskWrapperT{
					Task: TaskT{},
					Err:  err,
				}
				return
			}
			for _, task = range tasks {
				ch <- TaskWrapperT{
					Task:               task,
					TimeRangeSequence:  i,
					TimeRangesTotal:    len(timeRanges),
					TimeRangeStartDate: urlStartDate,
					TimeRangeStartTime: urlStartTime,
					TimeRangeEndDate:   urlEndDate,
					TimeRangeEndTime:   urlEndTime,
					Err:                nil,
				}
			}

		}

		return
	}()

	return ch, nil
}

func (c *Client) GetGroups() ([]string, error) {
	var groups []string

	url := fmt.Sprintf("%s%s", c.url, URLGETGROUPS)
	body, err := c.getUrlBody(url)
	if err != nil {
		return []string{}, err
	}
	err = json.Unmarshal(body, &groups)
	if err != nil {
		return []string{}, err
	}

	return groups, nil
}

func (c *Client) GetProjects(group string) ([]string, error) {
	var projects []string

	url := fmt.Sprintf("%s%s", c.url, fmt.Sprintf(URLGETPROJECTS, group))
	body, err := c.getUrlBody(url)
	if err != nil {
		return []string{}, err
	}
	err = json.Unmarshal(body, &projects)
	if err != nil {
		return []string{}, err
	}

	return projects, nil
}
