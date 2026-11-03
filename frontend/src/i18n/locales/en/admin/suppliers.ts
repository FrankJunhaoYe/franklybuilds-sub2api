export default {
  suppliers: {
    title: 'Suppliers', description: 'Manage upstream connections, shared wallets, and group keys.', supplier: 'Supplier',
    create: 'Add supplier', edit: 'Edit supplier', search: 'Search name or URL', all: 'All suppliers', selfOwned: 'Self-owned accounts',
    url: 'Base URL', urlHint: 'All keys share this address. Changing it clears previous lookup results.',
    urlManaged: 'The supplier manages this address:', keyNameHint: 'Name each key after its upstream group, such as “Standard” or “Premium”.',
    notes: 'Notes', status: 'Status', active: 'Active', inactive: 'Inactive', allStatus: 'All statuses',
    keys: 'Keys', keyCount: '{count} keys', manageKeys: 'Manage keys', addKey: 'Add key', keyGroups: 'Local groups',
    accountRate: 'Account multiplier', upstreamRate: 'Upstream multiplier', noKeys: 'No keys yet', noKeysHint: 'Add an upstream key and choose the local groups that can use it.',
    sharedBalance: 'Shared wallet balance', sharedBalanceHint: 'All keys share one wallet balance, counted only once.',
    queryStatus: 'Balance query', balanceSource: 'Balance lookup', sourceAuto: 'Automatically use a supplier key', sourceAccount: 'Select an existing key', sourceSecret: 'Dedicated lookup key',
    sourceAutoHint: 'Adding a key automatically queries the shared balance. No separate lookup key is needed.', advancedBalanceSource: 'Advanced: choose a balance source',
    chooseKey: 'Choose a lookup key', queryKey: 'Dedicated balance lookup key', queryKeyHint: 'Used only for wallet queries, never model requests. Leave blank to keep the saved key.',
    secretSaved: 'A dedicated lookup key is saved', secretRequired: 'Enter a dedicated balance lookup key.', selectKeyRequired: 'Choose a key for balance queries.',
    refreshBalance: 'Refresh balance', refreshing: 'Querying', refreshed: 'Balance updated', refreshUnavailable: 'No wallet balance was returned. The last successful result has been kept.',
    balanceUnknown: 'Not queried', balanceSuccess: 'Query succeeded', balanceError: 'Query failed', balanceUnavailable: 'Unavailable',
    lastSuccess: 'Last success: {time}', noBalance: 'Unknown', noSuppliers: 'No suppliers yet', noSuppliersHint: 'Add an upstream address, then manage its keys.',
    noResults: 'No matching suppliers', loadFailed: 'Could not load suppliers. Please retry.', saveFailed: 'Could not save supplier.', saved: 'Supplier saved', deleted: 'Supplier deleted',
    deleteTitle: 'Delete supplier', deleteMessage: 'Delete “{name}” and all {count} locally saved keys? The dedicated lookup key will also be removed. Historical usage is retained.',
    deleteKeyTitle: 'Delete key', deleteKeyMessage: 'Delete the locally saved connection for “{name}”? Historical usage is retained.',
    disableTitle: 'Disable supplier', disableMessage: 'Disabling “{name}” stops new requests to its {count} keys. Re-enabling the supplier preserves each key’s own status.',
    disabledHint: 'This supplier is inactive. Its keys will not receive new requests.', refreshKey: 'Refresh multiplier and balance', keySavedProbeFailed: 'Key saved. Some queries failed; you can refresh them later.',
    back: 'Back to suppliers', viewAccounts: 'View in account management', detailHint: 'These keys are the same records shown in account management and appear there as soon as they are saved. A key can serve multiple local groups.',
    billingHint: 'New keys sync the upstream multiplier by default. Local group multipliers remain independent.',
    lastChecked: 'Last query: {time}',
    balanceErrors: {
      transport_unavailable: 'The query service is unavailable.', balance_account_unavailable: 'The selected key is no longer available. Choose another key.',
      balance_accounts_unavailable: 'Could not load the supplier keys. Please retry.',
      proxy_unavailable: 'The query key’s proxy is unavailable.', balance_key_missing: 'Add a key to this supplier first.',
      invalid_base_url: 'The base URL is invalid or inaccessible.', request_build_failed: 'Could not prepare the balance query.',
      request_failed: 'The upstream connection failed or timed out.', empty_response: 'The upstream returned no result.', unsupported: 'This upstream does not support wallet balance queries.',
      http_error: 'The upstream rejected the balance query. Check the query key.', invalid_response: 'The upstream returned an invalid balance.',
      wallet_not_provided: 'The upstream returned no wallet balance. Retry later or choose a query key in advanced settings.'
    },
    operationFailed: 'The operation failed. Please retry.'
  }
}
