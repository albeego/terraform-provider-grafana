package machinelearning

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/machine-learning-go-client/mlapi"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceOutlierDetector() *schema.Resource {
	return &schema.Resource{

		Description: `
An outlier detector monitors the results of a query and reports when its values are outside normal bands.

The normal band is configured by choice of algorithm, its sensitivity and other configuration.

Visit https://grafana.com/docs/grafana-cloud/machine-learning/outlier-detection/ for more details.
`,

		CreateContext: ResourceOutlierCreate,
		ReadContext:   ResourceOutlierRead,
		UpdateContext: ResourceOutlierUpdate,
		DeleteContext: ResourceOutlierDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"id": {
				Description: "The ID of the outlier detector.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"name": {
				Description: "The name of the outlier detector.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"metric": {
				Description: "The metric used to query the outlier detector results.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"description": {
				Description: "A description of the outlier detector.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"datasource_id": {
				Description:  "The id of the datasource to query.",
				Type:         schema.TypeInt,
				Optional:     true,
				ExactlyOneOf: []string{"datasource_uid"},
			},
			"datasource_uid": {
				Description: "The uid of the datasource to query.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			"datasource_type": {
				Description:  "The type of datasource being queried. Currently allowed values are prometheus, graphite, loki, postgres, and datadog.",
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"prometheus", "graphite", "loki", "postgres", "datadog"}, false),
			},
			"query_params": {
				Description: "An object representing the query params to query Grafana with.",
				Type:        schema.TypeMap,
				Required:    true,
			},
			"interval": {
				Description: "The data interval in seconds to monitor.",
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     300,
			},
			"algorithm": {
				Description: "The algorithm to use and its configuration. See https://grafana.com/docs/grafana-cloud/machine-learning/outlier-detection/ for details.",
				Type:        schema.TypeSet,
				Required:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Description:  "The name of the algorithm to use ('mad' or 'dbscan').",
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringInSlice([]string{"mad", "dbscan"}, false),
						},
						"sensitivity": {
							Description:  "Specify the sensitivity of the detector (in range [0,1]).",
							Type:         schema.TypeFloat,
							Required:     true,
							ValidateFunc: validation.FloatBetween(0, 1.0),
						},
						"config": {
							Description: "For DBSCAN only, specify the configuration map",
							Type:        schema.TypeSet,
							Optional:    true,
							MaxItems:    1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"epsilon": {
										Description:  "Specify the epsilon parameter (positive float)",
										Type:         schema.TypeFloat,
										Required:     true,
										ValidateFunc: validation.FloatAtLeast(0),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func ResourceOutlierCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	outlier, err := makeMLOutlier(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	outlier, err = c.NewOutlierDetector(ctx, outlier)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(outlier.ID)
	return ResourceOutlierRead(ctx, d, meta)
}

func ResourceOutlierRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	outlier, err := c.OutlierDetector(ctx, d.Id())
	if err != nil {
		var diags diag.Diagnostics
		if strings.HasPrefix(err.Error(), "status: 404") {
			name := d.Get("name").(string)
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Outlier Detector %q is in Terraform state, but no longer exists in Grafana ML", name),
				Detail:   fmt.Sprintf("%q will be recreated when you apply", name),
			})
			d.SetId("")
			return diags
		}
		return diag.FromErr(err)
	}

	d.Set("name", outlier.Name)
	d.Set("metric", outlier.Metric)
	d.Set("description", outlier.Description)
	if outlier.DatasourceID != 0 {
		d.Set("datasource_id", outlier.DatasourceID)
	} else {
		d.Set("datasource_id", nil)
	}
	if outlier.DatasourceUID != "" {
		d.Set("datasource_uid", outlier.DatasourceUID)
	} else {
		d.Set("datasource_uid", nil)
	}
	d.Set("datasource_type", outlier.DatasourceType)
	d.Set("query_params", outlier.QueryParams)
	d.Set("interval", outlier.Interval)
	d.Set("algorithm", convertToSetStructure(outlier.Algorithm))

	return nil
}

func ResourceOutlierUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	outlier, err := makeMLOutlier(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = c.UpdateOutlierDetector(ctx, outlier)
	if err != nil {
		return diag.FromErr(err)
	}
	return ResourceOutlierRead(ctx, d, meta)
}

func ResourceOutlierDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).MLAPI
	err := c.DeleteOutlierDetector(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}

func convertToSetStructure(al mlapi.OutlierAlgorithm) []interface{} {
	algorithmSet := make([]interface{}, 0, 1)
	algorithmConfigSet := make([]interface{}, 0, 1)

	if al.Config != nil {
		config := map[string]interface{}{
			"epsilon": al.Config.Epsilon,
		}
		algorithmConfigSet = append(algorithmConfigSet, config)
	}

	algorithm := map[string]interface{}{
		"name":        al.Name,
		"sensitivity": al.Sensitivity,
		"config":      algorithmConfigSet,
	}
	algorithmSet = append(algorithmSet, algorithm)
	return algorithmSet
}

func makeMLOutlier(d *schema.ResourceData, meta interface{}) (mlapi.OutlierDetector, error) {
	alSet := d.Get("algorithm").(*schema.Set)
	al := alSet.List()[0].(map[string]interface{})

	var algorithm mlapi.OutlierAlgorithm
	algorithm.Name = strings.ToLower(al["name"].(string))
	algorithm.Sensitivity = al["sensitivity"].(float64)

	if algorithm.Name == "dbscan" {
		config := new(mlapi.OutlierAlgorithmConfig)
		if configSet, ok := al["config"]; ok && configSet.(*schema.Set).Len() == 1 {
			cfg := configSet.(*schema.Set).List()[0].(map[string]interface{})
			config.Epsilon = cfg["epsilon"].(float64)
		} else {
			return mlapi.OutlierDetector{}, fmt.Errorf("DBSCAN algorithm requires a single \"config\" block")
		}
		algorithm.Config = config
	}

	return mlapi.OutlierDetector{
		ID:             d.Id(),
		Name:           d.Get("name").(string),
		Metric:         d.Get("metric").(string),
		Description:    d.Get("description").(string),
		GrafanaURL:     meta.(*common.Client).GrafanaAPIURL,
		DatasourceID:   uint(d.Get("datasource_id").(int)),
		DatasourceUID:  d.Get("datasource_uid").(string),
		DatasourceType: d.Get("datasource_type").(string),
		QueryParams:    d.Get("query_params").(map[string]interface{}),
		Interval:       uint(d.Get("interval").(int)),
		Algorithm:      algorithm,
	}, nil
}
