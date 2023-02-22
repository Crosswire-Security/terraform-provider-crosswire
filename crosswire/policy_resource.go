package crosswire

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &PolicyResource{}
var _ resource.ResourceWithConfigure = &PolicyResource{}

func NewPolicyResource() resource.Resource {
	return &PolicyResource{}
}

// ExampleResource defines the resource implementation.
type PolicyResource struct {
	client *Client
}

// ExampleResourceModel describes the resource data model.
type PolicyResourceModel struct {
	Owner                UserModel          `tfsdk:"owner"`
	Name                 types.String       `tfsdk:"name"`
	Entitlements         []EntitlementModel `tfsdk:"entitlements"`
	Condition            ConditionModel     `tfsdk:"condition"`
	SpecialApprover      types.String       `tfsdk:"special_approver"`
	ApprovalBehavior     types.String       `tfsdk:"approval_behavior"`
	UserApprovers        []UserModel        `tfsdk:"user_approvers"`
	TTL                  types.Int64        `tfsdk:"ttl"`
	EntitlementApprovers []EntitlementModel `tfsdk:"entitlement_approvers"`

	Id          types.String `tfsdk:"id"`
	State       types.String `tfsdk:"state"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

type EntitlementModel struct {
	Provider types.String `tfsdk:"provider"`
	Subject  types.String `tfsdk:"subject"`
	Object   types.String `tfsdk:"object"`
}

type UserModel struct {
	EmailAddress types.String `tfsdk:"email_address"`
}

type ConditionModel struct {
	Quantifier    types.String       `tfsdk:"quantifier"`
	Entitlements  []EntitlementModel `tfsdk:"entitlements"`
	Subconditions []ConditionModel   `tfsdk:"subconditions"`
}

func (p *PolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func attributeEntitlementSchemaV0() schema.NestedAttributeObject {
	return schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"provider": schema.StringAttribute{Required: true},
			"subject":  schema.StringAttribute{Required: true},
			"object":   schema.StringAttribute{Required: true},
		},
	}
}

func userAttributesV0() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"email_address": schema.StringAttribute{
			Required: true,
			Validators: []validator.String{
				stringvalidator.RegexMatches(
					regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$"),
					"must be a valid email address"),
			},
		},
	}
}

func attributeConditionSchemaV0(level int) schema.NestedAttributeObject {
	attributes := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"quantifier": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("ANY", "ALL"),
				},
				Description: "ANY only requires one of the entitlements or subconditions to be `true` in order for this condition block to be true while ALL requires all of them to be true.",
			},
			"entitlements": schema.SetNestedAttribute{
				Optional:     true,
				NestedObject: attributeEntitlementSchemaV0(),
				Description:  "Set of provider-subject-object tuples governing the truth value of this condition block.",
			},
		},
	}
	if level < 2 {
		attributes.Attributes["subconditions"] = schema.SetNestedAttribute{
			Optional:     true,
			NestedObject: attributeConditionSchemaV0(level + 1),
			Description:  "Set of subconditions governing the truth value of this condition block. If you need more than 3 levels of subconditions, please contact someone at Crosswire for assistance.",
		}
	}

	return attributes
}

func (p *PolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Policy resource",
		Attributes: map[string]schema.Attribute{
			"owner": schema.SingleNestedAttribute{
				Required:    true,
				Attributes:  userAttributesV0(),
				Description: "Email address of user creating the policy. This email address should exist within Crosswire.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: `Name of the policy. This is what users will see when requesting access.`,
			},
			"entitlements": schema.SetNestedAttribute{
				Required:     true,
				NestedObject: attributeEntitlementSchemaV0(),
				Description:  "Set of Provider-Subject-Object tuples corresponding to what access users will receive upon getting access to the policy.",
			},
			"condition": schema.SingleNestedAttribute{
				Required:    true,
				Attributes:  attributeConditionSchemaV0(0).Attributes,
				Description: "Conditions necessary to become eligible for this policy.",
			},
			"special_approver": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					StringDefault("NONE"),
				},
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("NONE", "AUTO", "SELF", "MANAGER"),
				},
				Description: `AUTO will automatically grant the policy if eligible.
Self will grant the policy once requested.
Manager requires the subject's manager to approve access.
If this is set to anything besides "NONE", don't set user_approvers or entitlement_approvers.`,
			},
			"approval_behavior": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					StringDefault("ANY"),
				},
				Validators: []validator.String{
					stringvalidator.OneOfCaseInsensitive("ANY", "ALL"),
				},
				Description: `ANY requires only one approval from the set of approvers specified
ALL requires approvals from every approver in order to gain access. When selecting this, make sure to have a small number of approvers to reduce in-flight time to gain access.`,
			},
			"user_approvers": schema.SetNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: userAttributesV0(),
				},
				Description: "Set of users (email addresses) who will be approving requests to this policy.",
			},
			"entitlement_approvers": schema.SetNestedAttribute{
				Optional:     true,
				NestedObject: attributeEntitlementSchemaV0(),
				Description: `Set of provider-subject-object tuples whose users will be approving requests to this policy.
Typically these would be group memberships rather than application access.`,
			},
			"ttl": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of seconds a user can hold the policy any given time",
			},
			"id": schema.StringAttribute{
				Computed: true,
				// Required:    true,
				Description: "Crosswire policy id",
			},
			"state": schema.StringAttribute{
				Computed:    true,
				Description: "Current state of the policy",
			},
			"last_updated": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp Terraform received the policy's latest update",
			},
		},
	}
}

func (p PolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data PolicyResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.TTL.ValueInt64() > 0 && data.SpecialApprover.ValueString() == "AUTO" {
		resp.Diagnostics.AddError(
			"Auto policies cannot have a TTL",
			"Because users are granted access automatically, TTL is redundant as it would continue to be regranted until the user no longer qualifies, at which time, the user would lose access immediately regardless of time remaining.",
		)
	}

	if data.SpecialApprover.ValueString() == "NONE" && len(data.EntitlementApprovers) == 0 && len(data.UserApprovers) == 0 {
		resp.Diagnostics.AddError(
			"No approvers selected",
			"At least one approver needs to be set to approve policy requests",
		)
	}

	if len(data.Entitlements) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("entitlements"),
			"No entitlements selected",
			"At least one entitlement needs to be set for the policy to function",
		)
	}
}

func (p *PolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Client)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	p.client = client
}

func (p *PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicyResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entitlementsFromModelConverter := func(modelEntitlements []EntitlementModel) []Entitlement {
		var entitlements []Entitlement
		for _, entitlement := range modelEntitlements {
			entitlements = append(entitlements, Entitlement{
				Provider: entitlement.Provider.ValueString(),
				Subject:  entitlement.Subject.ValueString(),
				Object:   entitlement.Object.ValueString(),
			})
		}
		return entitlements
	}

	var conditionFromModelConverter func(ConditionModel) Condition
	conditionFromModelConverter = func(conditionModel ConditionModel) Condition {
		condition := Condition{
			Quantifier:   conditionModel.Quantifier.ValueString(),
			Entitlements: entitlementsFromModelConverter(conditionModel.Entitlements),
		}
		if len(conditionModel.Subconditions) > 0 {
			for _, subcondition := range conditionModel.Subconditions {
				condition.Subconditions = append(condition.Subconditions, conditionFromModelConverter(subcondition))
			}
		}

		return condition
	}

	var userApprovers []string
	for _, user := range data.UserApprovers {
		userApprovers = append(userApprovers, user.EmailAddress.ValueString())
	}

	// Generate API request body from plan
	policy := Policy{
		Owner:                data.Owner.EmailAddress.ValueString(),
		Name:                 data.Name.ValueString(),
		Entitlements:         entitlementsFromModelConverter(data.Entitlements),
		Condition:            conditionFromModelConverter(data.Condition),
		SpecialApprover:      ToPointer(data.SpecialApprover.ValueString()),
		ApprovalBehavior:     ToPointer(data.ApprovalBehavior.ValueString()),
		UserApprovers:        userApprovers,
		EntitlementApprovers: entitlementsFromModelConverter(data.EntitlementApprovers),
	}

	createdPolicy, err := p.client.createPolicy(policy)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating policy",
			"Could not create policy, unexpected error: "+err.Error(),
		)
		return
	}

	if !data.TTL.IsUnknown() {
		policy.Ttl = ToPointer(data.TTL.ValueInt64())
	}

	entitlementsToModelConverter := func(entitlements []Entitlement) []EntitlementModel {
		var entitlementsModel []EntitlementModel
		for _, entitlement := range entitlements {
			entitlementsModel = append(entitlementsModel, EntitlementModel{
				Provider: types.StringValue(entitlement.Provider),
				Subject:  types.StringValue(entitlement.Subject),
				Object:   types.StringValue(entitlement.Object),
			})
		}
		return entitlementsModel
	}

	var userApproversModel []UserModel
	for _, user := range createdPolicy.UserApprovers {
		userApproversModel = append(userApproversModel, UserModel{EmailAddress: types.StringValue(user)})
	}

	data.Owner = UserModel{EmailAddress: types.StringValue(createdPolicy.Owner)}
	data.Name = types.StringValue(createdPolicy.Name)
	data.Entitlements = entitlementsToModelConverter(createdPolicy.Entitlements)
	data.Condition = conditionToModelConverter(createdPolicy.Condition)
	if createdPolicy.SpecialApprover != nil {
		data.SpecialApprover = types.StringValue(*createdPolicy.SpecialApprover)
	}
	if createdPolicy.ApprovalBehavior != nil {
		data.ApprovalBehavior = types.StringValue(*createdPolicy.ApprovalBehavior)
	}
	data.UserApprovers = userApproversModel
	data.EntitlementApprovers = entitlementsToModelConverter(createdPolicy.EntitlementApprovers)
	data.Id = types.StringValue(createdPolicy.Id)
	data.State = types.StringValue(createdPolicy.State)
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (p *PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PolicyResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Look up policy from Crosswire
	policy, err := p.client.getPolicy(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Policies",
			"Could not read policies, unexpected error: "+err.Error(),
		)
		return
	}

	if policy == nil {
		return
	}

	// Overwrite items with refreshed state
	var userApproversModel []UserModel
	for _, user := range policy.UserApprovers {
		userApproversModel = append(userApproversModel, UserModel{EmailAddress: types.StringValue(user)})
	}

	state.Owner = UserModel{EmailAddress: types.StringValue(policy.Owner)}
	state.Name = types.StringValue(policy.Name)
	state.Entitlements = entitlementsToModelConverter(policy.Entitlements)
	state.Condition = conditionToModelConverter(policy.Condition) //ConditionModel{Quantifier: types.StringValue(policy.Condition.Quantifier), Entitlements: conditions.Entitlements, Subconditions: []ConditionModel{{Quantifier: types.StringValue("ahh")}}} //conditionToModelConverter(policy.Condition)
	state.SpecialApprover = types.StringValue(*policy.SpecialApprover)
	state.ApprovalBehavior = types.StringValue(*policy.ApprovalBehavior)
	state.UserApprovers = userApproversModel
	state.EntitlementApprovers = entitlementsToModelConverter(policy.EntitlementApprovers)
	state.Id = types.StringValue(policy.Id)
	state.State = types.StringValue(policy.State)

	// Set refreshed state
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func entitlementsToModelConverter(entitlements []Entitlement) []EntitlementModel {
	var entitlementsModel []EntitlementModel
	for _, entitlement := range entitlements {
		entitlementsModel = append(entitlementsModel, EntitlementModel{
			Provider: types.StringValue(entitlement.Provider),
			Subject:  types.StringValue(entitlement.Subject),
			Object:   types.StringValue(entitlement.Object),
		})
	}
	return entitlementsModel
}

func conditionToModelConverter(condition Condition) ConditionModel {
	conditionModel := ConditionModel{
		Quantifier:   types.StringValue(condition.Quantifier),
		Entitlements: entitlementsToModelConverter(condition.Entitlements),
	}
	if len(condition.Subconditions) > 0 {
		for _, subcondition := range condition.Subconditions {
			conditionModel.Subconditions = append(conditionModel.Subconditions, conditionToModelConverter(subcondition))
		}
	}
	return conditionModel
}

func (p *PolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update is not yet supported", "Please contact customer support for additional information.")
}

func (p *PolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Delete is not yet supported", "Please contact customer support for additional information.")
}

func (p *PolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
