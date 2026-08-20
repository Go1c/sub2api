export default {
  nav: { checkin: '每日签到', checkinRecords: '签到流水' },
  checkin: {
    title: '每日签到', description: '坚持签到，领取每日余额奖励。', loadFailed: '签到状态加载失败。', retry: '重试',
    stats: { balance: '当前余额', totalCheckins: '累计签到', totalReward: '累计奖励', currentStreak: '当前连续', days: '{count} 天' },
    nextMilestone: { title: '下一里程碑', detail: '第 {day} 天 · +${bonus}', remaining: '还差 {count} 天', none: '暂未配置里程碑' },
    action: { ready: '立即签到', loading: '签到中...', checked: '今日已签到', exhausted: '今日奖池已发完', disabled: '签到未开放' },
    today: { awarded: '今日获得 ${amount}。', exhausted: '连续签到已记录，但今日全站奖池已发完。', replayed: '本次签到已记录，没有重复发放奖励。' },
    history: { title: '最近签到', empty: '暂无签到记录。', time: '签到时间', base: '基础奖励', milestone: '里程碑奖金', actual: '实际奖励', balance: '签到后余额', streak: '连续 / 循环', status: '状态', awarded: '已发放', exhausted: '奖池耗尽' },
    sidebar: { ready: '今日奖励待领取', checked: '今日已签到', exhausted: '今日连续签到已记录' },
    settings: {
      title: '每日签到', description: '独立配置签到奖励、运营时区和全站每日预算。', enabled: '开启每日签到', enabledHint: '关闭后隐藏用户入口，管理员仍可查看历史签到流水。',
      minReward: '最小随机奖励', maxReward: '最大随机奖励', timezone: '运营时区', dailyCap: '全站每日发放上限', dailyCapHint: '填 0 表示不限额；余额不足时不会部分发放。',
      milestones: '连续签到里程碑', milestoneDay: '循环天数', milestoneBonus: '固定奖金', addMilestone: '添加里程碑', removeMilestone: '删除里程碑', maximumReward: '理论最高单次奖励',
      budgetWarning: '每日上限低于理论最高单次奖励，部分有效签到可能无法获得奖励。', save: '保存签到设置', saving: '保存中...', saved: '签到设置已保存。', loadFailed: '签到设置加载失败。', saveFailed: '签到设置保存失败。'
    },
    admin: {
      title: '签到流水', description: '审计每日签到、连续天数、奖励发放和预算耗尽记录。', search: '邮箱或用户名', userId: '用户 ID', date: '业务日期', allStatuses: '全部状态',
      columns: { user: '用户', businessDate: '业务日期', checkedAt: '签到时间', streak: '连续 / 循环', base: '基础奖励', milestone: '里程碑奖金', actual: '实际奖励', balance: '余额快照', status: '状态' }, loadFailed: '签到流水加载失败。',
      stats: {
        day: '今日', week: '本周', month: '本月', all: '累计', range: '{from} ~ {to}', timezone: '{timezone}',
        uniqueUsers: '参与用户', checkins: '签到次数', total: '发放总额', avg: '平均额度', p50: 'P50', p90: 'P90', max: '最大额度',
        loadFailed: '签到统计加载失败。'
      }
    }
  }
}
