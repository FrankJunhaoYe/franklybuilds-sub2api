package service

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSupplierGatePreservesAccountState(t *testing.T) {
	account := &Account{Status: StatusActive, Schedulable: true, SupplierDisabled: true}
	require.False(t, account.IsActive())
	require.False(t, account.IsSchedulable())
	require.False(t, account.IsCredentialUsableForShadow())
	require.Equal(t, StatusActive, account.Status)
	require.True(t, account.Schedulable)
	account.SupplierDisabled = false
	require.True(t, account.IsSchedulable())
	account.Schedulable = false
	require.False(t, account.IsSchedulable())
}

func TestSupplierURLOverridesPerProtocolDestinations(t *testing.T) {
	id := int64(12)
	for _, protocol := range []string{APIProtocolAdaptive, APIProtocolAnthropic, APIProtocolChatCompletions} {
		t.Run(protocol, func(t *testing.T) {
			account := &Account{SupplierID: &id, Platform: PlatformKimi, Type: AccountTypeAPIKey, Credentials: map[string]any{
				"base_url": "https://supplier.example", "api_protocol": protocol,
				"api_base_urls": map[string]any{APIProtocolChatCompletions: "https://old.example", APIProtocolAnthropic: "https://other.example"},
			}}
			require.Equal(t, "https://supplier.example", account.GetOpenAIBaseURL())
			require.Equal(t, "https://supplier.example", account.GetOpenAIFormatBaseURL())
			require.Equal(t, "https://supplier.example", account.GetCNProtocolBaseURL(APIProtocolResponses))
			account.Credentials["base_url"] = ""
			require.Empty(t, account.GetOpenAIBaseURL(), "missing supplier URL must never send credentials to a default vendor")
		})
	}
}
