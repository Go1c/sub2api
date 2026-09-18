export default {
    title: 'Import 2FA JSON',
    hint: 'Import email, password and a 2FA secret to authorize Codex. Prefer Team workspaces. After a 401, reauthorize the same Team and resume scheduling only after validation.',
    unavailable: 'Configure the private login worker and a stable encryption key first.',
    files: 'Account files (multiple JSON/TXT)', content: 'Or paste account data', groups: 'Account groups', proxy: 'Login and account egress',
    direct: 'No proxy', ipGroups: 'IP groups', proxies: 'Single proxy', refresh: 'Refresh jobs',
    queued: 'Queued', running: 'Signing in / recovering', succeeded: 'Account configured', failed: 'Failed; scheduling remains stopped',
    retry: 'Retry', submit: 'Import and authorize', loadFailed: 'Could not load settings or jobs',
    tooLarge: 'Total input must not exceed 1 MB', chooseFile: 'Select 1–20 files or paste account data',
    accepted: 'Queued {count} accounts.', submitFailed: 'Import failed. Check login service and input format.',
    retryFailed: 'The account may be busy or cooling down. Try again in five minutes.'
  }
