package selectel

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/terraform-providers/terraform-provider-selectel/selectel/internal/api/scheduledbackup"
)

func resourceCloudBackupPlanV2() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceCloudBackupPlanV2Create,
		UpdateContext: resourceCloudBackupPlanV2Update,
		DeleteContext: resourceCloudBackupPlanV2Delete,
		Timeouts:      &schema.ResourceTimeout{
			// todo
		},
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Project identifier in UUID format",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Human-readable name of the plan",
			},
			"backup_mode": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "full",
				ValidateFunc: validation.StringInSlice([]string{"full", "frequency"}, true),
				Description:  `Backup mode used for this plan. Allowed values: "full", "frequency"`,
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Detailed plan description",
			},
			"max_backups": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Maximum number of backups to save in a full plan or full backups in a frequency plan",
			},
			"schedule_type": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "crontab",
				ValidateFunc: validation.StringInSlice([]string{"crontab", "full"}, true),
				Description:  `Backup scheduling type. Allowed values: "calendar", "crontab"`,
			},
			"schedule_pattern": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "0 0 * * *",
				Description: "Backup scheduling pattern",
			},
			"resources": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "List of resources included in the plan",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
		CustomizeDiff: func(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
			_ = d.Clear("resources")
			return nil
		},
	}
}

func resourceCloudBackupPlanV2Create(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diagErr := getScheduledBackupClient(d, meta)
	if diagErr != nil {
		return diagErr
	}

	var (
		projectID          = d.Get("project_id").(string)
		name               = d.Get("name").(string)
		backupMode, _      = d.Get("backup_mode").(string)
		description, _     = d.Get("description").(string)
		maxBackups         = d.Get("max_backups").(int)
		scheduleType, _    = d.Get("schedule_type").(string)
		schedulePattern, _ = d.Get("schedule_pattern").(string)
	)

	plan := scheduledbackup.Plan{
		BackupMode:      backupMode,
		Description:     description,
		MaxBackups:      maxBackups,
		Name:            name,
		Resources:       nil, // todo resources
		SchedulePattern: schedulePattern,
		ScheduleType:    scheduleType,
	}

	createdPlan, _, err := client.PlanCreate(ctx, projectID, &plan)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(createdPlan.ID)

	stateConf := &resource.StateChangeConf{
		Pending: []string{
			scheduledbackup.PlanStatusSuspended,
		},
		Target: []string{
			scheduledbackup.PlanStatusStarted,
		},
		Timeout: d.Timeout(schema.TimeoutCreate),
		Refresh: func() (result interface{}, state string, err error) {
			p, _, err := client.Plan(ctx, projectID, createdPlan.ID)
			if err != nil {
				return nil, "", err
			}

			if p == nil {
				return nil, "", fmt.Errorf("can't find created plan %q", createdPlan.ID)
			}

			return p, p.Status, nil
		},
		MinTimeout: 10 * time.Second,
	}

	_, err = stateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf(
			"error waiting for the server %s to become '%s': %v",
			createdPlan.ID, scheduledbackup.PlanStatusStarted, err,
		)
	}

	return nil
}

func resourceCloudBackupPlanV2Update(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diagErr := getScheduledBackupClient(d, meta)
	if diagErr != nil {
		return diagErr
	}

	var (
		projectID          = d.Get("project_id").(string)
		name               = d.Get("name").(string)
		backupMode, _      = d.Get("backup_mode").(string)
		description, _     = d.Get("description").(string)
		maxBackups         = d.Get("max_backups").(int)
		scheduleType, _    = d.Get("schedule_type").(string)
		schedulePattern, _ = d.Get("schedule_pattern").(string)
	)

	plan := scheduledbackup.Plan{
		BackupMode:      backupMode,
		Description:     description,
		MaxBackups:      maxBackups,
		Name:            name,
		Resources:       nil, // todo resources
		SchedulePattern: schedulePattern,
		ScheduleType:    scheduleType,
	}

	_, _, err := client.PlanUpdate(ctx, projectID, d.Id(), &plan)
	if err != nil {
		return diag.FromErr(err)
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{
			scheduledbackup.PlanStatusSuspended,
		},
		Target: []string{
			scheduledbackup.PlanStatusStarted,
		},
		Timeout: d.Timeout(schema.TimeoutUpdate),
		Refresh: func() (result interface{}, state string, err error) {
			p, _, err := client.Plan(ctx, projectID, d.Id())
			if err != nil {
				return nil, "", err
			}

			if p == nil {
				return nil, "", fmt.Errorf("can't find created plan %q", d.Id())
			}

			return p, p.Status, nil
		},
		MinTimeout: 10 * time.Second,
	}

	_, err = stateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf(
			"error waiting for the server %s to become '%s': %v",
			d.Id(), scheduledbackup.PlanStatusStarted, err,
		)
	}

	return nil
}

func resourceCloudBackupPlanV2Delete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, diagErr := getScheduledBackupClient(d, meta)
	if diagErr != nil {
		return diagErr
	}

	var (
		projectID = d.Get("project_id").(string)
		planID    = d.Id()
	)

	_, err := client.PlanDelete(ctx, projectID, planID)
	if err != nil {
		return diag.FromErr(err)
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{
			scheduledbackup.PlanStatusStarted,
			scheduledbackup.PlanStatusSuspended,
		},
		Target: []string{
			// todo test
		},
		Timeout: d.Timeout(schema.TimeoutDelete),
		Refresh: func() (result interface{}, state string, err error) {
			p, resp, err := client.Plan(ctx, projectID, planID)
			if err != nil {
				return nil, "", err
			}

			if resp.StatusCode == http.StatusNotFound {
				return nil, "", nil
			}

			return p, p.Status, nil
		},
	}

	_, err = stateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.Errorf("error waiting for the server %s to be deleted: %v", d.Id(), err)
	}

	d.SetId("")

	return nil
}
