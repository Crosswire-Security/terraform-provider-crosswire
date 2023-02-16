package crosswire

import (
	"context"
	"fmt"
	"math/rand"
	"time"
	"unsafe"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// stringDefaultModifier is a plan modifier that sets a default value for a
// types.StringType attribute when it is not configured. The attribute must be
// marked as Optional and Computed. When setting the state during the resource
// Create, Read, or Update methods, this default value must also be included or
// the Terraform CLI will generate an error.
type stringDefaultModifier struct {
	Default string
}

// Description returns a plain text description of the validator's behavior, suitable for a practitioner to understand its impact.
func (m stringDefaultModifier) Description(ctx context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to %s", m.Default)
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior, suitable for a practitioner to understand its impact.
func (m stringDefaultModifier) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to `%s`", m.Default)
}

// PlanModifyString runs the logic of the plan modifier.
// Access to the configuration, plan, and state is available in `req`, while
// `resp` contains fields for updating the planned value, triggering resource
// replacement, and returning diagnostics.
func (m stringDefaultModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If the value is unknown or known, do not set default value.
	if !req.PlanValue.IsNull() {
		return
	}

	resp.PlanValue = types.StringValue(m.Default)
}

func StringDefault(defaultValue string) planmodifier.String {
	return stringDefaultModifier{
		Default: defaultValue,
	}
}

func ToPointer[T any](t T) *T {
	return &t
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandomStringGenerator(n int) string {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}
