package codersdk_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/coder/coder/v2/codersdk"
)

func TestNormalizeTemplateBuilderRegistryURL(t *testing.T) {
	t.Parallel()

	t.Run("Accepts", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			in   string
			want string
		}{
			{"Empty", "", codersdk.DefaultTemplateBuilderRegistryURL},
			{"Whitespaceonly", "   ", codersdk.DefaultTemplateBuilderRegistryURL},
			{"BareHost", "registry.coder.com", "registry.coder.com"},
			{"BareHostSurroundingSpace", "  mirror.internal.example  ", "mirror.internal.example"},
			{"HostPort", "mirror.example.com:8443", "mirror.example.com:8443"},
			{"HTTPSStripped", "https://mirror.example.com", "mirror.example.com"},
			{"HTTPStripped", "http://mirror.example.com", "mirror.example.com"},
			{"UppercaseSchemeStripped", "HTTPS://mirror.example.com", "mirror.example.com"},
			{"TrailingSlashStripped", "https://mirror.example.com/", "mirror.example.com"},
			{"TrailingSlashesStripped", "mirror.example.com///", "mirror.example.com"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				got, err := codersdk.NormalizeTemplateBuilderRegistryURL(tc.in)
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			})
		}
	})

	t.Run("Rejects", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			in   string
		}{
			{"DoubledScheme", "https://https://mirror.example.com"},
			{"Path", "mirror.example.com/coder"},
			{"Interpolation", "mirror.example.com/${var.evil}"},
			{"Directive", "mirror.example.com/%{if true}"},
			{"Backslash", `mirror.example.com\x`},
			{"Quote", `mirror.example.com/"`},
			{"InteriorSpace", "mirror .example.com"},
			{"NonBreakingSpace", "mirror\u00a0.example.com"},
			{"VerticalTab", "mirror\v.example.com"},
			{"SchemeOnly", "https://"},
			{"LeadingDot", ".mirror.example.com"},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				_, err := codersdk.NormalizeTemplateBuilderRegistryURL(tc.in)
				require.Error(t, err)
				require.ErrorContains(t, err, "bare host")
			})
		}
	})

	t.Run("RejectsCredentialsWithoutEchoingThem", func(t *testing.T) {
		t.Parallel()
		// Userinfo is rejected (not stripped), and the error must not disclose the
		// credential; this is the P1 the config validation exists to close.
		_, err := codersdk.NormalizeTemplateBuilderRegistryURL("https://user:s3cr3t-token@mirror.example.com")
		require.Error(t, err)
		require.NotContains(t, err.Error(), "s3cr3t-token")
	})
}

// TestTemplateBuilderRegistryURLConfigValidation covers the config boundary: a
// malformed CODER_TEMPLATE_BUILDER_REGISTRY_URL must fail at server start naming
// the option, instead of surfacing as a per-request compose failure.
func TestTemplateBuilderRegistryURLConfigValidation(t *testing.T) {
	t.Parallel()

	registryOption := func() (set func(string) error) {
		opts := (&codersdk.DeploymentValues{}).Options()
		for i := range opts {
			if opts[i].Flag == "template-builder-registry-url" {
				return opts[i].Value.Set
			}
		}
		return nil
	}

	set := registryOption()
	require.NotNil(t, set, "template-builder-registry-url option must exist")

	require.NoError(t, set("mirror.example.com"))
	require.NoError(t, set(""))

	err := set("https://user:s3cr3t-token@mirror.example.com")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "s3cr3t-token")
}
