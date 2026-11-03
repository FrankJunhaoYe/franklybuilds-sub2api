export default {
  suppliers: {
    title: '供应商管理', description: '管理上游接入、共享钱包与各分组 Key。', supplier: '供应商',
    create: '添加供应商', edit: '编辑供应商', search: '搜索名称或 URL', all: '全部供应商', selfOwned: '自有账号',
    url: '接入 URL', urlHint: '旗下所有 Key 共用此地址。修改后会清除旧地址的查询结果。',
    urlManaged: '接入地址由供应商统一维护：', keyNameHint: '用 Key 名称区分上游分组，例如「标准组」「高倍率组」。',
    notes: '备注', status: '状态', active: '启用', inactive: '停用', allStatus: '全部状态',
    keys: 'Key', keyCount: '{count} 个 Key', manageKeys: '管理 Key', addKey: '添加 Key', keyGroups: '本站分组',
    accountRate: '账号倍率', upstreamRate: '上游声明倍率', noKeys: '尚未添加 Key', noKeysHint: '添加上游 Key，并选择可使用它的本站分组。',
    sharedBalance: '共享钱包余额', sharedBalanceHint: '旗下 Key 共享一份钱包余额，不重复累加。',
    queryStatus: '查询状态', balanceSource: '余额查询方式', sourceAuto: '自动使用旗下 Key', sourceAccount: '指定已有 Key', sourceSecret: '专用查询 Key',
    sourceAutoHint: '添加 Key 后会自动查询共享余额，无需单独配置查询 Key。', advancedBalanceSource: '高级：指定余额查询方式',
    chooseKey: '选择查询 Key', queryKey: '专用余额查询 Key', queryKeyHint: '仅用于查询钱包，不参加模型调用。留空保留已保存的 Key。',
    secretSaved: '已保存专用查询 Key', secretRequired: '请输入专用余额查询 Key。', selectKeyRequired: '请选择用于查询余额的 Key。',
    refreshBalance: '刷新余额', refreshing: '正在查询', refreshed: '余额已更新', refreshUnavailable: '本次未能取得钱包余额，已保留最近成功结果。',
    balanceUnknown: '尚未查询', balanceSuccess: '查询成功', balanceError: '查询失败', balanceUnavailable: '暂无法获取',
    lastSuccess: '最近成功：{time}', noBalance: '未知', noSuppliers: '尚未添加供应商', noSuppliersHint: '先添加一个上游地址，再管理它的多个 Key。',
    noResults: '没有匹配的供应商', loadFailed: '加载供应商失败，请重试。', saveFailed: '保存供应商失败。', saved: '供应商已保存', deleted: '供应商已删除',
    deleteTitle: '删除供应商', deleteMessage: '删除「{name}」及其在本站保存的全部 {count} 个 Key？专用查询 Key 也会删除，历史用量记录保留。',
    deleteKeyTitle: '删除 Key', deleteKeyMessage: '删除「{name}」在本站保存的接入记录？历史用量记录保留。',
    disableTitle: '停用供应商', disableMessage: '停用「{name}」后，旗下 {count} 个 Key 将停止接收新请求。恢复供应商会保留每个 Key 自身的启停状态。',
    disabledHint: '供应商已停用，旗下 Key 不会接收新请求。', refreshKey: '刷新倍率与余额', keySavedProbeFailed: 'Key 已保存，部分查询未成功，可稍后手动刷新。',
    back: '返回供应商', viewAccounts: '在账号管理中查看', detailHint: '这里的 Key 与账号管理是同一条记录，保存后会直接出现在账号管理中。Key 可加入多个本站分组。',
    billingHint: '新 Key 默认同步上游倍率；本站分组倍率保持独立。',
    lastChecked: '最近查询：{time}',
    balanceErrors: {
      transport_unavailable: '查询服务暂不可用。', balance_account_unavailable: '原查询 Key 已不可用，请重新选择。',
      balance_accounts_unavailable: '暂时无法读取供应商 Key，请重试。',
      proxy_unavailable: '查询 Key 的代理暂不可用。', balance_key_missing: '请先为供应商添加一个 Key。',
      invalid_base_url: '接入 URL 无效或不允许访问。', request_build_failed: '无法准备余额查询。',
      request_failed: '无法连接上游或查询超时。', empty_response: '上游没有返回查询结果。', unsupported: '上游暂不支持钱包余额查询。',
      http_error: '上游拒绝了余额查询，请检查查询 Key。', invalid_response: '上游返回的余额格式无效。',
      wallet_not_provided: '上游未返回钱包余额，可稍后重试或在高级设置中指定查询 Key。'
    },
    operationFailed: '操作失败，请重试。'
  }
}
