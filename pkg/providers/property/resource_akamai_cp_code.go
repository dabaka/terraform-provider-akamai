package property

import (
	"fmt"
	"github.com/akamai/terraform-provider-akamai/v2/pkg/akamai"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/client-v1"
	"github.com/akamai/terraform-provider-akamai/v2/pkg/tools"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// PAPI CP Code
//
// https://developer.akamai.com/api/luna/papi/data.html#cpcode
// https://developer.akamai.com/api/luna/papi/resources.html#cpcodesapi
func resourceCPCode() *schema.Resource {
	return &schema.Resource{
		Create: resourceCPCodeCreate,
		Read:   resourceCPCodeRead,
		Delete: resourceCPCodeDelete,
		Update: resourceCPCodeUpdate,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"contract": {
				Type:     schema.TypeString,
				Required: true,
			},
			"group": {
				Type:     schema.TypeString,
				Required: true,
			},
			"product": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceCPCodeCreate(d *schema.ResourceData, meta interface{}) error {
	akactx := akamai.ContextGet(inst.Name())
	logger := akactx.Log("PAPI", "resourceCPCodeCreate")
	CorrelationID := "[PAPI][resourceCPCodeCreate-" + akactx.OperationID() + "]"

	logger.Debug("Creating CP Code")
	// Because CPCodes can't be deleted, we re-use an existing CPCode if it's there
	name, ok := d.Get("name").(string)
	if !ok {
		return fmt.Errorf("%w: %s, %q", tools.ErrInvalidType, "name", "string")
	}
	product, ok := d.Get("product").(string)
	if !ok {
		return fmt.Errorf("%w: %s, %q", tools.ErrInvalidType, "product", "string")
	}
	group, ok := d.Get("group").(string)
	if !ok {
		return fmt.Errorf("%w: %s, %q", tools.ErrInvalidType, "group", "string")
	}
	contract, ok := d.Get("contract").(string)
	if !ok {
		return fmt.Errorf("%w: %s, %q", tools.ErrInvalidType, "contract", "string")
	}
	cpCodes := resourceCPCodePAPINewCPCodes(contract, group)
	cpCode, err := cpCodes.FindCpCode(name, CorrelationID)
	// TODO: err can indicate that either error was returned while fetching CP Codes from PAPI or no CP Codes were found for provided group and contract
	// this should be modified in client library as currently we do not know whether there was an actual error (in which case err should be returned immediately without proceeding to create the resource)
	if cpCode == nil || err != nil {
		cpCode = cpCodes.NewCpCode()
		cpCode.ProductID = product
		cpCode.CpcodeName = name

		logger.Debug("CPCode: %+v")
		err := cpCode.Save(CorrelationID)
		if err != nil {
			logger.Debug("Error saving")
			logger.Debug("%s", err.(client.APIError).RawBody)
			return err
		}
	}
	var found bool
	for _, id := range cpCode.ProductIDs {
		if id == product {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("attempting to modify product ID: %w", ErrPAPICPCodeModify)
	}
	logger.Debug("Resulting CP Code: %#v", cpCode)
	d.SetId(cpCode.CpcodeID)
	return resourceCPCodeRead(d, meta)
}

func resourceCPCodeDelete(d *schema.ResourceData, meta interface{}) error {
	akactx := akamai.ContextGet(inst.Name())
	logger := akactx.Log("PAPI", "resourceCPCodeDelete")
	logger.Debug("Deleting CP Code")
	// No PAPI CP Code delete operation exists.
	// https://developer.akamai.com/api/luna/papi/resources.html#cpcodesapi
	return schema.Noop(d, meta)
}

func resourceCPCodeRead(d *schema.ResourceData, _ interface{}) error {
	akactx := akamai.ContextGet(inst.Name())
	logger := akactx.Log("PAPI", "resourceCPCodeRead")
	CorrelationID := "[PAPI][resourceCPCodeRead-" + akactx.OperationID() + "]"

	logger.Debug("Read CP Code")
	name, ok := d.Get("name").(string)
	if !ok {
		return fmt.Errorf("%w: %s, %q", tools.ErrInvalidType, "name", "string")
	}
	group, ok := d.Get("group").(string)
	if !ok {
		return fmt.Errorf("%w: %s, %q", tools.ErrInvalidType, "group", "string")
	}
	contract, ok := d.Get("contract").(string)
	if !ok {
		return fmt.Errorf("%w: %s, %q", tools.ErrInvalidType, "contract", "string")
	}
	cpCodes := resourceCPCodePAPINewCPCodes(contract, group)
	cpCode, err := cpCodes.FindCpCode(d.Id(), CorrelationID)
	if cpCode == nil || err != nil {
		cpCode, err = cpCodes.FindCpCode(name, CorrelationID)
		if err != nil {
			return err
		}
	}

	if cpCode == nil {
		return nil
	}

	d.SetId(cpCode.CpcodeID)
	logger.Debug("Read CP Code: %+v", cpCode)
	return nil
}

func resourceCPCodeUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("contract") {
		return fmt.Errorf("attempting to modify contract ID: %w", ErrPAPICPCodeModify)
	}
	if d.HasChange("product") {
		return fmt.Errorf("attempting to modify product ID: %w", ErrPAPICPCodeModify)
	}
	if d.HasChange("group") {
		return fmt.Errorf("attempting to modify group ID: %w", ErrPAPICPCodeModify)
	}
	return resourceCPCodeRead(d, meta)
}

func resourceCPCodePAPINewCPCodes(contractID, groupID string) *papi.CpCodes {
	contract := &papi.Contract{
		ContractID: contractID,
	}
	group := &papi.Group{
		GroupID: groupID,
	}
	return papi.NewCpCodes(contract, group)
}
