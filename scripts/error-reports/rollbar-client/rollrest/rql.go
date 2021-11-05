package rollrest

import (
	"fmt"
	"time"

	"github.com/davidji99/simpleresty"
)

type RQLService service

type CreateJobRequest struct {
	QueryString string `json:"query_string"`
}

type CreateJobResponse struct {
	Err    int `json:"err"`
	Result struct {
		ID int `json:"id"`
	} `json:"result"`
}

func (i *RQLService) CreateJob(query string) (*CreateJobResponse, *simpleresty.Response, error) {
	var result *CreateJobResponse
	urlStr := i.client.http.RequestURL("/rql/jobs/")

	// Set the correct authentication header
	i.client.setAuthTokenHeader(i.client.accountAccessToken)

	req := CreateJobRequest{query}
	resp, err := i.client.http.Post(urlStr, &result, req)

	return result, resp, err
}

type RqlJobResult struct {
	Rows     [][]interface{} `json:"rows"`
	Columns  []string        `json:"columns"`
	RowCount int             `json:"rowcount"`
}

type CheckJobResponse struct {
	Err    int `json:"err"`
	Result struct {
		Status string       `json:"status"`
		Result RqlJobResult `json:"result"`
	} `json:"result"`
}

func (i *RQLService) CheckJob(id int) (*CheckJobResponse, *simpleresty.Response, error) {
	var result *CheckJobResponse
	urlStr := i.client.http.RequestURL(fmt.Sprintf("/rql/job/%d?expand=result", id))

	// Set the correct authentication header
	i.client.setAuthTokenHeader(i.client.accountAccessToken)

	resp, err := i.client.http.Get(urlStr, &result, nil)

	return result, resp, err
}

type GetJobResponse struct {
	Err    int `json:"err"`
	Result struct {
		Result RqlJobResult `json:"result"`
	} `json:"result"`
}

func (i *RQLService) GetJob(id int) (*GetJobResponse, *simpleresty.Response, error) {
	var result *GetJobResponse
	urlStr := i.client.http.RequestURL(fmt.Sprintf("/rql/job/%d/result?expand=result", id))

	// Set the correct authentication header
	i.client.setAuthTokenHeader(i.client.accountAccessToken)

	resp, err := i.client.http.Get(urlStr, &result, nil)

	return result, resp, err
}

func (i *RQLService) Run(query string) (*RqlJobResult, error) {
	resCreate, _, err := i.CreateJob(query)
	if err != nil {
		return nil, err
	}
	if resCreate.Err != 0 {
		return nil, fmt.Errorf("CreateJob responded with error code %d", resCreate.Err)
	}

	wait := 5 * time.Second
	timeout := 320 * time.Second
	for x := 0; x < int(timeout.Seconds()/wait.Seconds()); x++ {
		resCheck, _, err := i.CheckJob(resCreate.Result.ID)
		if err != nil {
			return nil, err
		}
		if resCheck.Err != 0 {
			return nil, fmt.Errorf("GetJob responded with error code %d", resCheck.Err)
		}
		if resCheck.Result.Status == "success" {
			return &resCheck.Result.Result, nil
		}
		if resCheck.Result.Status == "failed" || resCheck.Result.Status == "cancelled" || resCheck.Result.Status == "timed_out" {
			return nil, fmt.Errorf("CheckJob responded with %s", resCheck.Result.Status)
		}
		time.Sleep(wait)
	}

	return nil, fmt.Errorf("Timed out waiting for job to complete")
}
