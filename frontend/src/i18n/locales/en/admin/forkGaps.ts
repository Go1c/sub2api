// Fork i18n gap pack: restores admin keys missing from monothithic after upstream merges.
// Loaded last among admin modular overlays (deep-merge fill).
export default {
  accounts: {
      errorHistory: {
            columns: {
                    count: 'Count',
                    message: 'Error Message',
                    model: 'Model',
                    source: 'Source',
                    statusCode: 'Status',
                    time: 'Time',
                    userEmail: 'User Email'
                  },
            dupHint: 'This error repeated {count} times in a short window',
            empty: 'No error records',
            subtitle: 'Most recent 20 error records',
            title: 'Error History'
          },
      fromModel: 'From model',
      messages: {
            accountCreated: 'Account created'
          },
      toModel: 'To model',
      upstreamBalance: {
            failed: 'Request failed',
            label: 'Upstream Balance',
            success: 'Request succeeded'
          }
    },
  affiliates: {
      records: {
            fingerprint: 'Fingerprint',
            ip: 'IP',
            result: 'Result',
            signupBonusAmount: 'Signup Bonus',
            signupBonusAt: 'Registered At',
            signupBonusGranted: 'Granted'
          }
    },
  channels: {
      emptyModelsInPricing: 'Pricing entry has no models',
      form: {
            imageInputPrice: 'Image input price'
          },
      noGroupsSelected: 'No groups selected'
    },
  ops: {
      runtime: {
            metricThresholds: 'Metric thresholds',
            metricThresholdsHint: 'Thresholds used for runtime health evaluation',
            requestErrorRateMaxPercent: 'Max request error rate (%)',
            requestErrorRateMaxPercentHint: 'Treat as unhealthy when error rate exceeds this value',
            slaMinPercent: 'Min SLA availability (%)',
            slaMinPercentHint: 'Alert when availability falls below this value',
            ttftP99MaxMs: 'TTFT P99 max (ms)',
            ttftP99MaxMsHint: 'P99 first-token latency threshold',
            upstreamErrorRateMaxPercent: 'Max upstream error rate (%)',
            upstreamErrorRateMaxPercentHint: 'Alert when upstream error rate exceeds this value'
          },
      settings: {
            accountAlertCooldownMinutes: 'Cooldown (minutes)',
            accountAlertIntervalMinutes: 'Scan interval (minutes)',
            accountAlertMaxRows: 'Max accounts',
            accountAlertMaxUsers: 'Top user emails',
            accountAlertMinCount: 'Trigger count',
            accountAlertWindowMinutes: 'Window (minutes)',
            accountErrorAlert: 'Account Error Telegram Alert',
            enableAccountErrorAlert: 'Enable account error alert',
            telegramBotToken: 'Telegram Bot Token',
            telegramChatId: 'Telegram Chat ID',
            validation: {
                    accountAlertCooldownRange: 'Account error alert cooldown must be between 0 and 10080 minutes',
                    accountAlertIntervalRange: 'Account error alert scan interval must be between 1 and 1440 minutes',
                    accountAlertMaxRowsRange: 'Account error alert max accounts must be between 1 and 50',
                    accountAlertMaxUsersRange: 'Account error alert top user emails must be between 0 and 10',
                    accountAlertMinCountRange: 'Account error alert trigger count must be between 1 and 100000',
                    accountAlertWindowRange: 'Account error alert window must be between 1 and 1440 minutes',
                    telegramBotTokenRequired: 'Telegram Bot Token is required when account error alert is enabled',
                    telegramChatIdRequired: 'Telegram Chat ID is required when account error alert is enabled'
                  }
          },
      userRequestMonitor: {
            adminOnly: 'Admin only',
            capture: {
                    actions: 'Actions',
                    bytes: 'Bytes',
                    contentType: 'Content-Type',
                    endpoint: 'Endpoint',
                    expiresAt: 'Expires at',
                    model: 'Model',
                    requestId: 'Request ID',
                    time: 'Time'
                  },
            capturesTitle: 'Request captures',
            close: 'Close',
            copied: 'Copied',
            copy: 'Copy body',
            copyFailed: 'Copy failed',
            create: 'Start monitoring',
            createFailed: 'Failed to create monitor',
            createTitle: 'Create monitor',
            created: 'Monitor created',
            creating: 'Creating...',
            deleteCapture: 'Delete capture',
            deleteConfirm: 'Delete the request monitor for {email} and all of its captures? This cannot be undone.',
            deleteFailed: 'Failed to delete capture',
            deleteMonitor: 'Delete',
            deleteMonitorFailed: 'Failed to delete monitor',
            deleted: 'Capture deleted',
            description: 'Start a time-boxed monitor for one user and capture future original request bodies.',
            detail: 'Detail',
            detailTitle: 'Request Body Detail',
            download: 'Download',
            downloadFailed: 'Failed to download captures',
            downloaded: 'Capture export started',
            durationMinutes: 'Duration (minutes)',
            empty: 'No user request monitors.',
            invalidForm: 'Enter a valid user ID and monitor settings',
            loadCaptureFailed: 'Failed to load capture detail',
            loadCapturesFailed: 'Failed to load captures',
            loadFailed: 'Failed to load request monitors',
            monitorDeleted: 'Monitor deleted',
            noCaptures: 'No captures yet.',
            rateLimit: 'Max captures/min',
            rawWarning: 'Warning: request bodies are stored raw without redaction. Capture failures are logged only and never block user requests.',
            refresh: 'Refresh',
            retentionDays: 'Retention days',
            sampleRate: 'Sample rate %',
            searchPlaceholder: 'Search by email',
            status: {
                    active: 'Active',
                    all: 'All',
                    expired: 'Expired',
                    stopped: 'Stopped'
                  },
            stop: 'Stop',
            stopFailed: 'Failed to stop monitor',
            stopped: 'Monitor stopped',
            table: {
                    actions: 'Actions',
                    captures: 'Captures',
                    endsAt: 'Ends at',
                    limits: 'Rate / Sample',
                    status: 'Status',
                    user: 'User'
                  },
            title: 'User Request Monitoring',
            truncated: 'truncated',
            userId: 'User ID',
            userIdPlaceholder: 'e.g. 123',
            viewCaptures: 'View captures'
          }
    },
  settings: {
      features: {
            affiliate: {
                    balanceGate: {
                              description: 'Prevents gifted balances from being used directly by abuse accounts. Applies only to balance billing, not subscription billing.',
                              minBalance: 'Minimum Current Balance',
                              minBalanceHint: 'Account balance must be greater than this value to use balance-based services.',
                              minRecharge: 'Minimum Historical Recharge',
                              minRechargeHint: 'Historical recharge must be greater than this value. Invite bonuses do not count as recharge.',
                              title: 'Balance Usage Gate'
                            },
                    tiers: {
                              description: 'Levels are fixed to L1-L4. When invited users and invitee recharge volume both meet a threshold, the highest matched tier is used.',
                              hint: 'Leave rebate rate empty to disable that level. After saving, both the user page and rebate accrual use this backend configuration.',
                              level: 'Level',
                              minInvitees: 'Invitees >=',
                              minRecharge: 'Recharge >=',
                              rebateRate: 'Rebate Rate',
                              title: 'Tiered Rebate Settings',
                              unconfigured: 'Not configured'
                            }
                  }
          },
      payment: {
            providerMapay: 'Mapay'
          },
      registration: {
            invitationRegistrationMode: 'Invitation verification mode',
            invitationRegistrationModeAffiliateLink: 'User invite link',
            invitationRegistrationModeBoth: 'Both allowed',
            invitationRegistrationModeHint: 'Choose which invitation credentials users can register with.',
            invitationRegistrationModeRedeemCode: 'Admin invitation code'
          },
      site: {
            frontendLocales: 'Available Frontend Languages',
            frontendLocalesHint: 'Controls which languages users can choose from in the frontend language switcher.',
            sitePages: {
                    add: 'Add Page',
                    content: 'Markdown Content',
                    contentHint: 'Markdown is supported. After saving, the page is available under its path.',
                    contentPlaceholder: '# Docs\n\nWrite Markdown content here.',
                    description: 'Configure the docs, terms, and privacy items shown in the top navigation as Markdown pages or embedded link pages.',
                    invalidLink: 'Link mode requires a complete http(s) URL',
                    invalidSlug: 'Public page path must start with doc/ and cannot contain ?, #, backslash, double slash, or ..',
                    itemLabel: 'Page #{n}',
                    key: 'Page Key',
                    keyPlaceholder: 'e.g., docs',
                    linkHint: 'In link mode, navigation opens this http(s) URL. The path can still embed it at /doc/...',
                    linkPlaceholder: 'https://docs.example.com',
                    linkUrl: 'Link URL',
                    mode: 'Page Mode',
                    modeLink: 'Link',
                    modeMarkdown: 'Markdown',
                    required: 'Public pages require a key, title, and path',
                    slug: 'Path',
                    slugPlaceholder: 'doc/docs',
                    title: 'Public Pages',
                    titleLabel: 'Page Title',
                    titlePlaceholder: 'e.g., Docs'
                  }
          },
      openaiFastPolicy: {
            addUserId: 'Add user ID',
            removeUserId: 'Remove user ID',
            userIdPlaceholder: 'e.g. 123'
          }
    },
  subscriptions: {
      columns: {
            exhaustedAt: 'Exhausted At'
          },
      createdEnd: 'Created to',
      createdStart: 'Created from',
      emailFilter: 'Email',
      failedToResetWeeklyLimit: 'Failed to reset weekly usage limit',
      limit: 'Limit',
      planId: 'Plan ID',
      quotaPool: 'Credit Pool',
      recent30dWaste: '30d Waste',
      remaining: 'remaining',
      resetWeeklyLimit: 'Reset Weekly Limit',
      resetWeeklyLimitConfirm: 'Reset only the current weekly usage window for \'{user}\'? This clears weekly usage only; total quota, cumulative quota used, daily usage, subscription status, and expiration remain unchanged.',
      resetWeeklyLimitTitle: 'Reset Weekly Usage Limit',
      status: {
            suspended: 'Suspended'
          },
      used: 'Used',
      wasteStats: {
            byPlan: 'Waste by Plan',
            dailyBreakdown: 'Daily Breakdown',
            description: 'Track unused subscription credit by period and plan.',
            end: 'End date',
            failedToLoad: 'Failed to load waste stats',
            period: 'Period',
            plan: 'Plan',
            start: 'Start date',
            subscriptions: 'Subscriptions',
            title: 'Subscription Waste Stats',
            totalQuota: 'Total Quota',
            totalUsed: 'Total Used',
            totalWaste: 'Wasted Credit',
            trend: 'Trend',
            unknownPlan: 'Unknown plan',
            userId: 'User ID',
            wasteRate: 'Waste Rate',
            weeklyBreakdown: 'Weekly Breakdown'
          },
      weeklyLimitResetSuccess: 'Weekly usage limit reset successfully'
    },
  usage: {
      billingModeVideo: 'Video',
      tokenRanking: {
            subtitle: 'Per-user token usage for the current filters and time range',
            rowHint: "Click to view this user's usage details",
            userCount: '{count} users',
            columns: {
                    user: 'User',
                    requests: 'Requests',
                    inputTokens: 'Input Tokens',
                    outputTokens: 'Output Tokens',
                    cacheTokens: 'Cache Tokens',
                    totalTokens: 'Total Tokens',
                    cost: 'Cost'
                  }
          }
    },
  users: {
      typeBalancePayment: 'Subscription via Balance',
      typePromoBalance: 'Balance (Promo)',
      typeSubscriptionPayment: 'Subscription Payment',
      passwordCopied: 'Password copied'
    }
}
