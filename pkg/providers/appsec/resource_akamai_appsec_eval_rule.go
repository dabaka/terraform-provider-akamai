package appsec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/v2/pkg/appsec"
	"github.com/akamai/terraform-provider-akamai/v2/pkg/akamai"
	"github.com/akamai/terraform-provider-akamai/v2/pkg/tools"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// appsec v1
//
// https://developer.akamai.com/api/cloud_security/application_security/v1.html
func resourceEvalRule() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceEvalRuleCreate,
		ReadContext:   resourceEvalRuleRead,
		UpdateContext: resourceEvalRuleUpdate,
		DeleteContext: resourceEvalRuleDelete,
		CustomizeDiff: customdiff.All(
			VerifyIDUnchanged,
		),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"config_id": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"security_policy_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"rule_id": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"rule_action": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: ValidateActions,
			},
			"condition_exception": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringIsJSON),
				DiffSuppressFunc: suppressEquivalentJSONDiffsConditionException,
			},
		},
	}
}

func resourceEvalRuleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := akamai.Meta(m)
	client := inst.Client(meta)
	logger := meta.Log("APPSEC", "resourceEvalRuleCreate")
	logger.Debugf("in resourceEvalRuleCreate")

	configID, err := tools.GetIntValue("config_id", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	version := getModifiableConfigVersion(ctx, configID, "evalRule", m)
	policyID, err := tools.GetStringValue("security_policy_id", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	ruleID, err := tools.GetIntValue("rule_id", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	action, err := tools.GetStringValue("rule_action", d)
	if err != nil {
		return diag.FromErr(err)
	}
	conditionexception, err := tools.GetStringValue("condition_exception", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}
	jsonPayloadRaw := []byte(conditionexception)
	rawJSON := (json.RawMessage)(jsonPayloadRaw)

	if err := validateActionAndConditionException(action, conditionexception); err != nil {
		return diag.FromErr(err)
	}

	createEvalRule := appsec.UpdateEvalRuleRequest{
		ConfigID:       configID,
		Version:        version,
		PolicyID:       policyID,
		RuleID:         ruleID,
		Action:         action,
		JsonPayloadRaw: rawJSON,
	}

	_, err = client.UpdateEvalRule(ctx, createEvalRule)
	if err != nil {
		logger.Warnf("calling 'createEvalRule': %s", err.Error())
	}

	d.SetId(fmt.Sprintf("%d:%s:%d", createEvalRule.ConfigID, createEvalRule.PolicyID, createEvalRule.RuleID))

	return resourceEvalRuleRead(ctx, d, m)
}

func resourceEvalRuleRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := akamai.Meta(m)
	client := inst.Client(meta)
	logger := meta.Log("APPSEC", "resourceEvalRuleRead")
	logger.Debugf("in resourceEvalRuleRead")

	iDParts, err := splitID(d.Id(), 3, "configID:securityPolicyID:ruleID")
	if err != nil {
		return diag.FromErr(err)
	}
	configID, err := strconv.Atoi(iDParts[0])
	if err != nil {
		return diag.FromErr(err)
	}
	version := getLatestConfigVersion(ctx, configID, m)
	policyID := iDParts[1]
	ruleID, err := strconv.Atoi(iDParts[2])
	if err != nil {
		return diag.FromErr(err)
	}

	getEvalRule := appsec.GetEvalRuleRequest{
		ConfigID: configID,
		Version:  version,
		PolicyID: policyID,
		RuleID:   ruleID,
	}

	evalrule, err := client.GetEvalRule(ctx, getEvalRule)
	if err != nil {
		logger.Warnf("calling 'getEvalRule': %s", err.Error())
	}

	if err := d.Set("config_id", getEvalRule.ConfigID); err != nil {
		return diag.Errorf("%s: %s", tools.ErrValueSet, err.Error())
	}
	if err := d.Set("security_policy_id", getEvalRule.PolicyID); err != nil {
		return diag.Errorf("%s: %s", tools.ErrValueSet, err.Error())
	}
	if err := d.Set("rule_id", getEvalRule.RuleID); err != nil {
		return diag.Errorf("%s: %s", tools.ErrValueSet, err.Error())
	}
	if err := d.Set("rule_action", string(evalrule.Action)); err != nil {
		return diag.Errorf("%s: %s", tools.ErrValueSet, err.Error())
	}

	if !evalrule.IsEmptyConditionException() {
		jsonBody, err := json.Marshal(evalrule.ConditionException)
		if err != nil {
			return diag.Errorf("%s", "Error Marshalling condition exception")
		}
		if err := d.Set("condition_exception", string(jsonBody)); err != nil {
			return diag.Errorf("%s: %s", tools.ErrValueSet, err.Error())
		}
	}

	return nil
}

func resourceEvalRuleUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := akamai.Meta(m)
	client := inst.Client(meta)
	logger := meta.Log("APPSEC", "resourceEvalRuleUpdate")
	logger.Debugf("in resourceEvalRuleUpdate")

	iDParts, err := splitID(d.Id(), 3, "configID:securityPolicyID:ruleID")
	if err != nil {
		return diag.FromErr(err)
	}
	configID, err := strconv.Atoi(iDParts[0])
	if err != nil {
		return diag.FromErr(err)
	}
	version := getModifiableConfigVersion(ctx, configID, "evalRule", m)
	policyID := iDParts[1]
	ruleID, err := strconv.Atoi(iDParts[2])
	if err != nil {
		return diag.FromErr(err)
	}
	action, err := tools.GetStringValue("rule_action", d)
	if err != nil {
		return diag.FromErr(err)
	}
	conditionexception, err := tools.GetStringValue("condition_exception", d)
	if err != nil && !errors.Is(err, tools.ErrNotFound) {
		return diag.FromErr(err)
	}

	if err := validateActionAndConditionException(action, conditionexception); err != nil {
		return diag.FromErr(err)
	}

	jsonPayloadRaw := []byte(conditionexception)
	rawJSON := (json.RawMessage)(jsonPayloadRaw)

	updateEvalRule := appsec.UpdateEvalRuleRequest{
		ConfigID:       configID,
		Version:        version,
		PolicyID:       policyID,
		RuleID:         ruleID,
		Action:         action,
		JsonPayloadRaw: rawJSON,
	}

	_, err = client.UpdateEvalRule(ctx, updateEvalRule)
	if err != nil {
		logger.Warnf("calling 'updateEvalRule': %s", err.Error())
	}

	return resourceEvalRuleRead(ctx, d, m)
}

func resourceEvalRuleDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	meta := akamai.Meta(m)
	client := inst.Client(meta)
	logger := meta.Log("APPSEC", "resourceEvalRuleDelete")
	logger.Debugf("in resourceEvalRuleDelete")

	iDParts, err := splitID(d.Id(), 3, "configID:securityPolicyID:ruleID")
	if err != nil {
		return diag.FromErr(err)
	}
	configID, err := strconv.Atoi(iDParts[0])
	if err != nil {
		return diag.FromErr(err)
	}
	version := getModifiableConfigVersion(ctx, configID, "evalRule", m)
	policyID := iDParts[1]
	ruleID, err := strconv.Atoi(iDParts[2])
	if err != nil {
		return diag.FromErr(err)
	}

	removeEvalRule := appsec.UpdateEvalRuleRequest{
		ConfigID: configID,
		Version:  version,
		PolicyID: policyID,
		RuleID:   ruleID,
		Action:   "none",
	}

	_, err = client.UpdateEvalRule(ctx, removeEvalRule)
	if err != nil {
		logger.Errorf("calling 'RemoveEvalRule': %s", err.Error())
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
