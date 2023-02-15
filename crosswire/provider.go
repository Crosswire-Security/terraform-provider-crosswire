package crosswire

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &CrosswireProvider{}

// CrosswireProvider defines the provider implementation.
type CrosswireProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ScaffoldingProviderModel describes the provider data model.
type CrosswireProviderModel struct {
	Host   types.String `tfsdk:"host"`
	ApiKey types.String `tfsdk:"api_key"`
}

func (p *CrosswireProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "crosswire"
	resp.Version = p.version
}

func (p *CrosswireProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional: true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key for your Crosswire organization",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *CrosswireProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring Crosswire client")
	var config CrosswireProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.ApiKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown Crosswire API Key",
			"The provider cannot create the Crosswire API client as there is an unknown configuration value for the Crosswire API key. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the CROSSWIRE_API_KEY environment variable.")
	}

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown Crosswire API Host",
			"The provider cannot create the Crosswire API client as there is an unknown configuration value for the Crosswire API host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the CROSSWIRE_API_HOST environment variable.")
	}

	if resp.Diagnostics.HasError() {
		return
	}

	host := os.Getenv("CROSSWIRE_API_HOST")

	apiKey := os.Getenv("CROSSWIRE_API_KEY")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.ApiKey.IsNull() {
		apiKey = config.ApiKey.ValueString()
	}

	if host == "" {
		host = HostURL
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Crosswire API Secret Key",
			"The provider cannot create the Crosswire API client as there is a missing or empty value for the Crosswire API key. "+
				"Set the password value in the configuration or use the CROSSWIRE_API_KEY environment variable. "+
				"If one is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "crosswire_host", host)
	ctx = tflog.SetField(ctx, "crosswire_api_key", apiKey)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "crosswire_api_key")

	client, err := NewClient(&host, &apiKey)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Crosswire API Client",
			"An unexpected error occurred when creating the Crosswire API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Crosswire client error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
	tflog.Info(ctx, "Configured Crosswire client", map[string]any{"success": true})
}

func (p *CrosswireProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPolicyResource,
	}
}

func (p *CrosswireProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CrosswireProvider{
			version: version,
		}
	}
}
