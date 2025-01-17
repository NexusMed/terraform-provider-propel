package propel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/propeldata/terraform-provider-propel/propel/internal/utils"
	pc "github.com/propeldata/terraform-provider-propel/propel_client"
)

func resourceDataPool() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDataPoolCreate,
		ReadContext:   resourceDataPoolRead,
		UpdateContext: resourceDataPoolUpdate,
		DeleteContext: resourceDataPoolDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		Description:   "Provides a Propel Data Pool resource. This can be used to create and manage Propel Data Pools.",
		Schema: map[string]*schema.Schema{
			"unique_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The Data Pool's name.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The Data Pool's description.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Data Pool's status.",
			},
			"account": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Account that the Data Pool belongs to.",
			},
			"environment": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Environment that the Data Pool belongs to.",
			},
			"data_source": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The Data Source that the Data Pool belongs to.",
			},
			"table": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the Data Pool's table.",
			},
			"column": {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    false,
				Description: "The list of columns, their types and nullability.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The column name.",
						},
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The column type.",
							ValidateFunc: utils.IsValidColumnType,
						},
						"nullable": {
							Type:        schema.TypeBool,
							Required:    true,
							Description: "Whether the column's type is nullable or not.",
						},
					},
				},
			},
			"tenant_id": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The tenant ID for restricting access between customers.",
			},
			"timestamp": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The Data Pool's timestamp column.",
			},
		},
	}
}

func expandPoolColumns(def []interface{}) []*pc.DataPoolColumnInput {
	columns := make([]*pc.DataPoolColumnInput, 0, len(def))

	for _, rawColumn := range def {
		column := rawColumn.(map[string]interface{})

		var columnType pc.ColumnType
		switch column["type"].(string) {
		case "BOOLEAN":
			columnType = pc.ColumnTypeBoolean
		case "DATE":
			columnType = pc.ColumnTypeDate
		case "DOUBLE":
			columnType = pc.ColumnTypeDouble
		case "INT8":
			columnType = pc.ColumnTypeInt8
		case "INT16":
			columnType = pc.ColumnTypeInt16
		case "INT32":
			columnType = pc.ColumnTypeInt32
		case "INT64":
			columnType = pc.ColumnTypeInt64
		case "STRING":
			columnType = pc.ColumnTypeString
		case "TIMESTAMP":
			columnType = pc.ColumnTypeTimestamp
		}

		columns = append(columns, &pc.DataPoolColumnInput{
			ColumnName: column["name"].(string),
			Type:       columnType,
			IsNullable: column["nullable"].(bool),
		})
	}

	return columns
}

func resourceDataPoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(graphql.Client)

	var diags diag.Diagnostics

	id := d.Get("data_source").(string)
	uniqueName := d.Get("unique_name").(string)
	description := d.Get("description").(string)

	columns := make([]*pc.DataPoolColumnInput, 0)
	if def, ok := d.Get("column").([]interface{}); ok && len(def) > 0 {
		columns = expandPoolColumns(def)
	}

	input := &pc.CreateDataPoolInputV2{
		UniqueName:  &uniqueName,
		Description: &description,
		DataSource:  id,
		Table:       d.Get("table").(string),
		Timestamp: &pc.TimestampInput{
			ColumnName: d.Get("timestamp").(string),
		},
		Columns: columns,
	}
	if _, exists := d.GetOk("tenant_id"); exists {
		input.Tenant = &pc.TenantInput{
			ColumnName: d.Get("tenant_id").(string),
		}
	}

	response, err := pc.CreateDataPool(ctx, c, input)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(response.CreateDataPoolV2.DataPool.Id)

	timeout := d.Timeout(schema.TimeoutCreate)

	err = waitForDataPoolLive(ctx, c, d.Id(), timeout)
	if err != nil {
		return diag.FromErr(err)
	}

	resourceDataPoolRead(ctx, d, meta)

	return diags
}

func resourceDataPoolRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(graphql.Client)

	var diags diag.Diagnostics

	response, err := pc.DataPool(ctx, c, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(response.DataPool.Id)
	if err := d.Set("unique_name", response.DataPool.UniqueName); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("description", response.DataPool.Description); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("status", response.DataPool.Status); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("environment", response.DataPool.Environment.Id); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("account", response.DataPool.Account.Id); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("data_source", response.DataPool.DataSource.Id); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("table", response.DataPool.Table); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("timestamp", response.DataPool.Timestamp.ColumnName); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceDataPoolUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(graphql.Client)

	if d.HasChanges("unique_name", "description") {
		id := d.Id()
		uniqueName := d.Get("unique_name").(string)
		description := d.Get("descriptionunique_name").(string)
		input := &pc.ModifyDataPoolInput{
			IdOrUniqueName: &pc.IdOrUniqueName{
				Id: &id,
			},
			UniqueName:  &uniqueName,
			Description: &description,
		}

		_, err := pc.ModifyDataPool(ctx, c, input)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceDataPoolRead(ctx, d, m)
}

func resourceDataPoolDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(graphql.Client)

	var diags diag.Diagnostics

	_, err := pc.DeleteDataPool(ctx, c, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	timeout := d.Timeout(schema.TimeoutDelete)
	err = waitForDataPoolDeletion(ctx, c, d.Id(), timeout)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return diags
}

func waitForDataPoolLive(ctx context.Context, client graphql.Client, id string, timeout time.Duration) error {
	createStateConf := &resource.StateChangeConf{
		Pending: []string{
			string(pc.DataPoolStatusCreated),
			string(pc.DataPoolStatusPending),
		},
		Target: []string{
			string(pc.DataPoolStatusLive),
		},
		Refresh: func() (interface{}, string, error) {
			resp, err := pc.DataPool(ctx, client, id)
			if err != nil {
				return 0, "", fmt.Errorf("error trying to read Data Pool status: %s", err)
			}

			return resp, string(resp.DataPool.Status), nil
		},
		Timeout:                   timeout - time.Minute,
		Delay:                     10 * time.Second,
		MinTimeout:                5 * time.Second,
		ContinuousTargetOccurence: 3,
	}

	_, err := createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return fmt.Errorf("error waiting for Data Pool to be LIVE: %s", err)
	}

	return nil
}

func waitForDataPoolDeletion(ctx context.Context, client graphql.Client, id string, timeout time.Duration) error {
	ticketInterval := 10 // 10s
	timeoutSeconds := int(timeout.Seconds())
	n := 0

	ticker := time.NewTicker(time.Duration(ticketInterval) * time.Second)
	for range ticker.C {
		if n*ticketInterval > timeoutSeconds {
			ticker.Stop()
			break
		}

		_, err := pc.DataPool(ctx, client, id)
		if err != nil {
			ticker.Stop()

			if strings.Contains(err.Error(), "not found") {
				return nil
			}

			return fmt.Errorf("error trying to fetch Data Pool: %s", err)
		}

		n++
	}
	return nil
}
