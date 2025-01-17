package propel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/Khan/genqlient/graphql"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	pc "github.com/propeldata/terraform-provider-propel/propel_client"
)

func resourceDataSource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDataSourceCreate,
		ReadContext:   resourceDataSourceRead,
		UpdateContext: resourceDataSourceUpdate,
		DeleteContext: resourceDataSourceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		SchemaVersion: 1,
		Description:   "Provides a Propel Data Source resource. This can be used to create and manage Propel Data Sources.",
		Schema: map[string]*schema.Schema{
			"unique_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The Data Source's name.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The Data Source's description.",
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					"Snowflake",
					"S3",
					"Http",
				}, true),
				Description: "The Data Source's type. Depending on this, you will need to specify one of `http_connection_settings`, `s3_connection_settings`, or `snowflake_connection_settings`.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Data Source's status.",
			},
			"account": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Account that the Data Source belongs to.",
			},
			"environment": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The Environment that the Data Source belongs to",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of when the Data Source was created.",
			},
			"modified_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The date and time of when the Data Source was modified.",
			},
			"created_by": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The user who created the Data Source.",
			},
			"modified_by": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The user who modified the Data Source.",
			},
			"snowflake_connection_settings": {
				Type:          schema.TypeList,
				Optional:      true,
				ConflictsWith: []string{"http_connection_settings", "s3_connection_settings"},
				MaxItems:      1,
				Description:   "Snowflake connection settings. Specify these for Snowflake Data Sources.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"account": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The Snowflake account. Only include the part before the \"snowflakecomputing.com\" part of your Snowflake URL (make sure you are in classic console, not Snowsight). For AWS-based accounts, this looks like \"znXXXXX.us-east-2.aws\". For Google Cloud-based accounts, this looks like \"ffXXXXX.us-central1.gcp\".",
						},
						"database": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The Snowflake database name.",
						},
						"warehouse": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The Snowflake warehouse name. It should be \"PROPELLING\" if you used the default name in the setup script.",
						},
						"schema": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The Snowflake schema.",
						},
						"role": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The Snowflake role. It should be \"PROPELLER\" if you used the default name in the setup script.",
						},
						"username": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The Snowflake username. It should be \"PROPEL\" if you used the default name in the setup script.",
						},
						"password": {
							Type:        schema.TypeString,
							Required:    true,
							Sensitive:   true,
							Description: "The Snowflake password.",
						},
					},
				},
			},
			"http_connection_settings": {
				Type:          schema.TypeList,
				Optional:      true,
				ConflictsWith: []string{"snowflake_connection_settings", "s3_connection_settings"},
				MaxItems:      1,
				Elem: &schema.Resource{
					Description: "HTTP connection settings. Specify these for HTTP Data Sources.",
					Schema: map[string]*schema.Schema{
						"basic_auth": {
							Type:        schema.TypeList,
							Optional:    true,
							MaxItems:    1,
							Description: "The HTTP Basic authentication settings for uploading new data.\n\nIf this parameter is not provided, anyone with the URL to your tables will be able to upload data. While it's OK to test without HTTP Basic authentication, we recommend enabling it.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"username": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The username for HTTP Basic authentication that must be included in the Authorization header when uploading new data.",
									},
									"password": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The password for HTTP Basic authentication that must be included in the Authorization header when uploading new data.",
									},
								},
							},
						},
					},
				},
			},
			"s3_connection_settings": {
				Type:          schema.TypeList,
				Optional:      true,
				ConflictsWith: []string{"snowflake_connection_settings", "http_connection_settings"},
				MaxItems:      1,
				Elem: &schema.Resource{
					Description: "The connection settings for an S3 Data Source. These include the S3 bucket name, the AWS access key ID, the AWS secret access key, and the tables (along with their paths).",
					Schema: map[string]*schema.Schema{
						"bucket": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the S3 bucket.",
						},
						"aws_access_key_id": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The AWS access key ID for an IAM user with sufficient access to the S3 bucket.",
						},
						"aws_secret_access_key": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The AWS secret access key for an IAM user with sufficient access to the S3 bucket.",
						},
					},
				},
			},
			"table": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Description: "Specify an HTTP or S3 Data Source's tables with this. You do not need to use this for Snowflake Data Sources, since Snowflake Data Sources' tables are automatically introspected.",
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the table.",
						},
						"path": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "The path to the table's files in S3.",
						},
						"column": {
							Type:        schema.TypeList,
							Required:    true,
							ForceNew:    true,
							Description: "Specify a table's columns.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The column name.",
									},
									"type": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "The column type.",
										ValidateFunc: validation.StringInSlice([]string{
											"BOOLEAN",
											"DATE",
											"DOUBLE",
											"FLOAT",
											"INT8",
											"INT16",
											"INT32",
											"INT64",
											"STRING",
											"TIMESTAMP",
										}, false),
									},
									"nullable": {
										Type:        schema.TypeBool,
										Required:    true,
										Description: "Whether the column's type is nullable or not.",
									},
								},
							},
						},
					},
				},
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},
	}
}

func resourceDataSourceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// TODO(mroberts): The Propel GraphQL API should eventually return this uppercase.
	dataSourceType := d.Get("type").(string)
	switch strings.ToUpper(dataSourceType) {
	case "SNOWFLAKE":
		return resourceSnowflakeDataSourceCreate(ctx, d, meta)
	case "HTTP":
		return resourceHttpDataSourceCreate(ctx, d, meta)
	case "S3":
		return resourceS3DataSourceCreate(ctx, d, meta)
	default:
		return diag.Errorf("Unsupported Data Source type \"%v\"", dataSourceType)
	}
}

func resourceSnowflakeDataSourceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(graphql.Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	connectionSettings := d.Get("snowflake_connection_settings").([]interface{})[0].(map[string]interface{})

	uniqueName := d.Get("unique_name").(string)
	description := d.Get("description").(string)
	input := &pc.CreateSnowflakeDataSourceInput{
		UniqueName:  &uniqueName,
		Description: &description,
		ConnectionSettings: &pc.SnowflakeConnectionSettingsInput{
			Account:   connectionSettings["account"].(string),
			Database:  connectionSettings["database"].(string),
			Warehouse: connectionSettings["warehouse"].(string),
			Schema:    connectionSettings["schema"].(string),
			Role:      connectionSettings["role"].(string),
			Username:  connectionSettings["username"].(string),
			Password:  connectionSettings["password"].(string),
		},
	}

	response, err := pc.CreateSnowflakeDataSource(ctx, c, input)
	if err != nil {
		return diag.FromErr(err)
	}

	switch r := (*response.GetCreateSnowflakeDataSource()).(type) {
	case *pc.CreateSnowflakeDataSourceCreateSnowflakeDataSourceDataSourceResponse:
		d.SetId(r.DataSource.Id)

		timeout := d.Timeout(schema.TimeoutCreate)

		err = waitForDataSourceConnected(ctx, c, d.Id(), timeout)
		if err != nil {
			return diag.FromErr(err)
		}

		return resourceDataSourceRead(ctx, d, meta)
	case *pc.CreateSnowflakeDataSourceCreateSnowflakeDataSourceFailureResponse:
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Failed to create Data Source",
		})
	}

	return diags
}

func resourceHttpDataSourceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(graphql.Client)

	var basicAuth *pc.HttpBasicAuthInput
	if d.Get("http_connection_settings") != nil && len(d.Get("http_connection_settings").([]interface{})) > 0 {
		cs := d.Get("http_connection_settings").([]interface{})[0].(map[string]interface{})

		if def, ok := cs["basic_auth"]; ok {
			basicAuth = expandBasicAuth(def.([]interface{}))
		}
	}

	tables := make([]*pc.HttpDataSourceTableInput, 0)
	if def, ok := d.Get("table").([]interface{}); ok && len(def) > 0 {
		tables = expandHttpTables(def)
	}

	uniqueName := d.Get("unique_name").(string)
	description := d.Get("description").(string)
	input := &pc.CreateHttpDataSourceInput{
		UniqueName:  &uniqueName,
		Description: &description,
		ConnectionSettings: &pc.HttpConnectionSettingsInput{
			BasicAuth: basicAuth,
			Tables:    tables,
		},
	}

	response, err := pc.CreateHttpDataSource(ctx, c, input)
	if err != nil {
		return diag.FromErr(err)
	}

	r := response.CreateHttpDataSource
	d.SetId(r.DataSource.Id)

	timeout := d.Timeout(schema.TimeoutCreate)

	err = waitForDataSourceConnected(ctx, c, d.Id(), timeout)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceDataSourceRead(ctx, d, meta)
}

func resourceS3DataSourceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(graphql.Client)

	tables := make([]*pc.S3DataSourceTableInput, 0)
	if def, ok := d.Get("table").([]interface{}); ok && len(def) > 0 {
		tables = expandS3Tables(def)
	}

	connectionSettings := d.Get("s3_connection_settings").([]interface{})[0].(map[string]interface{})

	uniqueName := d.Get("unique_name").(string)
	description := d.Get("description").(string)
	input := &pc.CreateS3DataSourceInput{
		UniqueName:  &uniqueName,
		Description: &description,
		ConnectionSettings: &pc.S3ConnectionSettingsInput{
			Bucket:             connectionSettings["bucket"].(string),
			AwsAccessKeyId:     connectionSettings["aws_access_key_id"].(string),
			AwsSecretAccessKey: connectionSettings["aws_secret_access_key"].(string),
			Tables:             tables,
		},
	}

	response, err := pc.CreateS3DataSource(ctx, c, input)
	if err != nil {
		return diag.FromErr(err)
	}

	r := response.CreateS3DataSource
	d.SetId(r.DataSource.Id)

	timeout := d.Timeout(schema.TimeoutCreate)

	err = waitForDataSourceConnected(ctx, c, d.Id(), timeout)
	if err != nil {
		return diag.FromErr(err)
	}

	return resourceDataSourceRead(ctx, d, meta)
}

func resourceDataSourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(graphql.Client)

	response, err := pc.DataSource(ctx, c, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(response.DataSource.Id)
	if err := d.Set("unique_name", response.DataSource.GetUniqueName()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("description", response.DataSource.GetDescription()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("created_at", response.DataSource.GetCreatedAt().String()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("created_by", response.DataSource.GetCreatedBy()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("modified_at", response.DataSource.GetModifiedAt().String()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("modified_by", response.DataSource.GetModifiedBy()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("environment", response.DataSource.GetEnvironment().Id); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("account", response.DataSource.GetAccount().Id); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("type", response.DataSource.GetType()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("status", response.DataSource.GetStatus()); err != nil {
		return diag.FromErr(err)
	}

	// TODO(mroberts): The Propel GraphQL API should eventually return this uppercase.
	dataSourceType := string(response.DataSource.Type)
	switch strings.ToUpper(dataSourceType) {
	case "SNOWFLAKE":
		return handleSnowflakeConnectionSettings(response, d)
	case "HTTP":
		if diags := handleHttpTables(response, d); diags != nil {
			return diags
		}
		return handleHttpConnectionSettings(response, d)
	case "S3":
		if diags := handleS3Tables(response, d); diags != nil {
			return diags
		}
		return handleS3ConnectionSettings(response, d)
	default:
		return diag.Errorf("Unsupported Data Source type \"%v\"", dataSourceType)
	}
}

func handleSnowflakeConnectionSettings(response *pc.DataSourceResponse, d *schema.ResourceData) diag.Diagnostics {
	cs := d.Get("snowflake_connection_settings").([]interface{})[0].(map[string]interface{})

	settings := map[string]interface{}{
		"password": cs["password"],
	}

	switch s := response.DataSource.GetConnectionSettings().(type) {
	case *pc.DataSourceDataConnectionSettingsSnowflakeConnectionSettings:
		settings["account"] = s.GetAccount()
		settings["database"] = s.GetDatabase()
		settings["warehouse"] = s.GetWarehouse()
		settings["schema"] = s.GetSchema()
		settings["role"] = s.GetRole()
		settings["username"] = s.GetUsername()
	default:
		return diag.Errorf("Missing SnowflakeConnectionSettings")
	}

	if err := d.Set("snowflake_connection_settings", []map[string]interface{}{settings}); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func handleHttpTables(response *pc.DataSourceResponse, d *schema.ResourceData) diag.Diagnostics {
	// FIXME(mroberts): We need to handle the case where tables is not yet populated.
	if response.DataSource.Tables == nil {
		return nil
	}

	tables := make([]interface{}, 0, len(response.DataSource.Tables.Nodes))

	// FIXME(mroberts): This is only going to work for the first page of results.
	for _, table := range response.DataSource.Tables.Nodes {
		columns := make([]interface{}, 0, len(table.Columns.Nodes))

		// FIXME(mroberts): This is only going to work for the first page of results.
		for _, column := range table.Columns.Nodes {
			columns = append(columns, map[string]interface{}{
				"name":     column.Name,
				"type":     column.Type,
				"nullable": column.IsNullable,
			})
		}

		tables = append(tables, map[string]interface{}{
			"name":   table.Name,
			"column": columns,
		})
	}

	if err := d.Set("table", (interface{})(tables)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func handleS3Tables(response *pc.DataSourceResponse, d *schema.ResourceData) diag.Diagnostics {
	// FIXME(mroberts): We need to handle the case where tables is not yet populated.
	if response.DataSource.Tables == nil {
		return nil
	}

	tables := make([]interface{}, 0, len(response.DataSource.Tables.Nodes))

	// FIXME(mroberts): This is only going to work for the first page of results.
	for _, table := range response.DataSource.Tables.Nodes {
		columns := make([]interface{}, 0, len(table.Columns.Nodes))

		// FIXME(mroberts): This is only going to work for the first page of results.
		for _, column := range table.Columns.Nodes {
			columns = append(columns, map[string]interface{}{
				"name": column.Name,
				// FIXME(mroberts): What about `path`?
				"type":     column.Type,
				"nullable": column.IsNullable,
			})
		}

		tables = append(tables, map[string]interface{}{
			"name":   table.Name,
			"column": columns,
		})
	}

	if err := d.Set("table", (interface{})(tables)); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func handleHttpConnectionSettings(response *pc.DataSourceResponse, d *schema.ResourceData) diag.Diagnostics {
	if d.Get("http_connection_settings") == nil || len(d.Get("http_connection_settings").([]interface{})) == 0 {
		return nil
	}

	cs := d.Get("http_connection_settings").([]interface{})[0].(map[string]interface{})

	switch s := response.DataSource.GetConnectionSettings().(type) {
	case *pc.DataSourceDataConnectionSettingsHttpConnectionSettings:
		if s.BasicAuth == nil {
			cs["basic_auth"] = nil
		} else if cs["basic_auth"] != nil && len(cs["basic_auth"].([]interface{})) > 0 {
			basicAuth := cs["basic_auth"].([]interface{})[0].(map[string]interface{})
			basicAuth["username"] = s.BasicAuth.Username
		}
	default:
		return diag.Errorf("Missing HttpConnectionSettings")
	}

	return nil
}

func handleS3ConnectionSettings(response *pc.DataSourceResponse, d *schema.ResourceData) diag.Diagnostics {
	cs := d.Get("s3_connection_settings").([]interface{})[0].(map[string]interface{})

	settings := map[string]interface{}{
		"aws_secret_access_key": cs["awsSecretAccessKey"],
	}

	switch s := response.DataSource.GetConnectionSettings().(type) {
	case *pc.DataSourceDataConnectionSettingsS3ConnectionSettings:
		settings["bucket"] = s.GetBucket()
		settings["aws_access_key_id"] = s.GetAwsAccessKeyId()
	default:
		return diag.Errorf("Missing S3ConnectionSettings")
	}

	return nil
}

func resourceDataSourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(graphql.Client)

	if d.HasChanges("unique_name", "description") {
		id := d.Id()
		uniqueName := d.Get("unique_name").(string)
		description := d.Get("description").(string)
		modifyDataSource := &pc.ModifySnowflakeDataSourceInput{
			IdOrUniqueName: &pc.IdOrUniqueName{
				Id: &id,
			},
			UniqueName:  &uniqueName,
			Description: &description,
		}

		_, err := pc.ModifySnowflakeDataSource(ctx, c, modifyDataSource)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return resourceDataSourceRead(ctx, d, m)
}

func resourceDataSourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(graphql.Client)

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	_, err := pc.DeleteDataSource(ctx, c, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// d.SetId("") is automatically called assuming delete returns no errors, but
	// it is added here for explicitness.
	d.SetId("")

	return diags
}

func waitForDataSourceConnected(ctx context.Context, client graphql.Client, id string, timeout time.Duration) error {
	createStateConf := &resource.StateChangeConf{
		Pending: []string{
			string(pc.DataSourceStatusCreated),
			string(pc.DataSourceStatusConnecting),
		},
		Target: []string{
			string(pc.DataSourceStatusConnected),
		},
		Refresh: func() (interface{}, string, error) {
			resp, err := pc.DataSource(ctx, client, id)
			if err != nil {
				return nil, "", fmt.Errorf("error trying to read Data Source status: %s", err)
			}

			return resp, string(resp.DataSource.Status), nil
		},
		Timeout:                   timeout - time.Minute,
		Delay:                     10 * time.Second,
		MinTimeout:                5 * time.Second,
		ContinuousTargetOccurence: 3,
	}

	_, err := createStateConf.WaitForStateContext(ctx)
	if err != nil {
		return fmt.Errorf("error waiting for Data Source to be CONNECTED: %s", err)
	}

	return nil
}

func expandHttpTables(def []interface{}) []*pc.HttpDataSourceTableInput {
	tables := make([]*pc.HttpDataSourceTableInput, 0, len(def))

	for _, rawTable := range def {
		table := rawTable.(map[string]interface{})

		columns := expandHttpColumns(table["column"].([]interface{}))

		tables = append(tables, &pc.HttpDataSourceTableInput{
			Name:    table["name"].(string),
			Columns: columns,
		})
	}

	return tables
}

func expandHttpColumns(def []interface{}) []*pc.HttpDataSourceColumnInput {
	columns := make([]*pc.HttpDataSourceColumnInput, 0, len(def))

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

		columns = append(columns, &pc.HttpDataSourceColumnInput{
			Name:     column["name"].(string),
			Type:     columnType,
			Nullable: column["nullable"].(bool),
		})
	}

	return columns
}

func expandS3Tables(def []interface{}) []*pc.S3DataSourceTableInput {
	tables := make([]*pc.S3DataSourceTableInput, 0, len(def))

	for _, rawTable := range def {
		table := rawTable.(map[string]interface{})

		columns := expandS3Columns(table["column"].([]interface{}))

		path := table["path"].(string)
		tables = append(tables, &pc.S3DataSourceTableInput{
			Name:    table["name"].(string),
			Path:    &path,
			Columns: columns,
		})
	}

	return tables
}

func expandS3Columns(def []interface{}) []*pc.S3DataSourceColumnInput {
	columns := make([]*pc.S3DataSourceColumnInput, 0, len(def))

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

		columns = append(columns, &pc.S3DataSourceColumnInput{
			Name:     column["name"].(string),
			Type:     columnType,
			Nullable: column["nullable"].(bool),
		})
	}

	return columns
}

func expandBasicAuth(def []interface{}) *pc.HttpBasicAuthInput {
	basicAuth := def[0].(map[string]interface{})

	return &pc.HttpBasicAuthInput{
		Username: basicAuth["username"].(string),
		Password: basicAuth["password"].(string),
	}
}
