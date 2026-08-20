export default {
  nav: { checkin: 'Daily Check-in', checkinRecords: 'Check-in Records' },
  checkin: {
    title: 'Daily Check-in',
    description: 'Build your streak and receive a daily balance reward.',
    loadFailed: 'Unable to load check-in status.', retry: 'Retry',
    stats: { balance: 'Current balance', totalCheckins: 'Total check-ins', totalReward: 'Total rewards', currentStreak: 'Current streak', days: '{count} days' },
    nextMilestone: { title: 'Next milestone', detail: 'Day {day} · +${bonus}', remaining: '{count} days to go', none: 'No milestone configured' },
    action: { ready: 'Check in now', loading: 'Checking in...', checked: 'Checked in today', exhausted: "Today's reward pool is exhausted", disabled: 'Check-in unavailable' },
    today: { awarded: 'Today you received ${amount}.', exhausted: 'Your streak is recorded. The daily reward pool was already exhausted.', replayed: 'This check-in was already recorded; no duplicate reward was issued.' },
    history: { title: 'Recent check-ins', empty: 'No check-in records yet.', time: 'Time', base: 'Base reward', milestone: 'Milestone bonus', actual: 'Actual reward', balance: 'Balance after', streak: 'Streak / cycle', status: 'Status', awarded: 'Awarded', exhausted: 'Pool exhausted' },
    sidebar: { ready: 'Reward available today', checked: 'Checked in today', exhausted: 'Streak recorded today' },
    settings: {
      title: 'Daily check-in', description: 'Configure rewards, the operating timezone, and the global daily budget independently.',
      enabled: 'Enable daily check-in', enabledHint: 'When disabled, user entry points are hidden while historical records remain available to administrators.',
      minReward: 'Minimum random reward', maxReward: 'Maximum random reward', timezone: 'Operating timezone', dailyCap: 'Global daily payout cap', dailyCapHint: 'Use 0 for no daily limit. Rewards are never partially paid.',
      milestones: 'Streak milestones', milestoneDay: 'Cycle day', milestoneBonus: 'Fixed bonus', addMilestone: 'Add milestone', removeMilestone: 'Remove milestone',
      maximumReward: 'Maximum single reward', budgetWarning: 'The daily cap is below the theoretical maximum single reward. A valid check-in may receive no reward.',
      save: 'Save check-in settings', saving: 'Saving...', saved: 'Check-in settings saved.', loadFailed: 'Unable to load check-in settings.', saveFailed: 'Unable to save check-in settings.'
    },
    admin: {
      title: 'Check-in Records', description: 'Audit daily check-ins, streaks, payouts, and zero-award budget events.', search: 'Email or username', userId: 'User ID', date: 'Business date', allStatuses: 'All statuses',
      columns: { user: 'User', businessDate: 'Business date', checkedAt: 'Checked at', streak: 'Streak / cycle', base: 'Base reward', milestone: 'Milestone bonus', actual: 'Actual reward', balance: 'Balance after', status: 'Status' },
      loadFailed: 'Unable to load check-in records.',
      stats: {
        day: 'Today', week: 'This week', month: 'This month', all: 'All time', range: '{from} ~ {to}', timezone: '{timezone}',
        uniqueUsers: 'Users', checkins: 'Check-ins', total: 'Total paid', avg: 'Average', p50: 'P50', p90: 'P90', max: 'Max',
        loadFailed: 'Unable to load check-in stats.'
      }
    }
  }
}
