export default {
    retrySameGroup: '重试仍使用 IP 组 {id}，不切换组；组内代理可能重新选取。',
    retrySameProxy: '重试仍使用代理 #{id}。',
    retrySameExit: '重试保留原出口配置，不会自动换组。',
    retryAccountExit: '恢复仍使用账号当前出口配置；如需换组，请先修改账号出口。',
    changeGroup: '换 IP 组并重新导入',
    chooseNewGroup: '请手动选择重新导入使用的 IP 组。',
    reimportHint: '请重新提供 {email} 的材料并手动选择 IP 组；将更新同邮箱任务，不新增任务。',

    requireIPGroup: '请选择可用动态 IP、IP 组或单个 IP，禁止服务器直连',
    title: '2FA JSON 导入',
    hint: '上传邮箱、密码和 2FA 密钥，自动获取 Codex 账号配置。优先选择 Team；401 停调度后重新登录原 Team，验证通过才恢复调度。',
    unavailable: '请先配置内部登录 Worker 与固定加密密钥。',
    files: '账号文件（可多选 JSON／TXT）', content: '或粘贴账号内容', groups: '账号分组', proxy: '登录与账号出口', dynamicIpGroups: '动态粘性 IP', requireProxy: '请选择动态 IP、IP 组或单个 IP，禁止服务器直连',
    direct: '不使用代理', ipGroups: 'IP 组', proxies: '单代理', refresh: '刷新任务状态',
    queued: '排队中', running: '正在登录／恢复', succeeded: '已写入账号配置', failed: '处理失败，保持停调度',
    retry: '重试', submit: '导入并获取账号配置', loadFailed: '读取配置或任务状态失败，请重试',
    tooLarge: '文件与粘贴内容合计不能超过 1 MB', chooseFile: '请选择 1–20 个文件或粘贴账号内容',
    accepted: '已创建 {count} 个后台任务。', submitFailed: '导入失败，请检查登录服务与输入格式',
    retryFailed: '账号可能正在处理或冷却，请五分钟后重试'
  }
