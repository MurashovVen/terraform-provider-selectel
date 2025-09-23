package scheduledbackup

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type (
	Checkpoint struct {
		ID              string           `json:"id"`
		PlanID          string           `json:"plan_id"`
		CreatedAt       string           `json:"created_at"`
		Status          string           `json:"status"`
		CheckpointItems []CheckpointItem `json:"checkpoint_items"`
	}

	CheckpointItem struct {
		ID              string             `json:"id"`
		BackupID        string             `json:"backup_id"`
		ChainID         string             `json:"chain_id"`
		CheckpointID    string             `json:"checkpoint_id"`
		CreatedAt       string             `json:"created_at"`
		BackupCreatedAt string             `json:"backup_created_at"`
		IsIncremental   bool               `json:"is_incremental"`
		Status          string             `json:"status"`
		Resource        CheckpointResource `json:"resource"`
	}

	CheckpointResource struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	}

	CheckpointsQuery struct {
		PlanName   string
		VolumeName string
	}
)

func (q *CheckpointsQuery) queryParamsRaw() string {
	params := url.Values{}
	if q.PlanName != "" {
		params.Add("plan_name", q.PlanName)
	}

	if q.VolumeName != "" {
		params.Add("volume_name", q.VolumeName)
	}

	return params.Encode()
}

func (client *ServiceClient) Checkpoints(ctx context.Context, projectID string, q *CheckpointsQuery) ([]*Checkpoint, *ResponseResult, error) {
	queryParams := ""
	if q != nil { // todo test all variants
		queryParams = "?" + q.queryParamsRaw()
	}

	u := fmt.Sprintf("%s/%s/checkpoints%s", client.Endpoint, projectID, queryParams)

	responseResult, err := client.DoRequest(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, nil, err
	}
	if responseResult.Err != nil {
		return nil, responseResult, responseResult.Err
	}

	var result struct {
		Result []*Checkpoint `json:"result"`
	}
	err = responseResult.ExtractResult(&result)
	if err != nil {
		return nil, responseResult, err
	}

	return result.Result, responseResult, nil
}
