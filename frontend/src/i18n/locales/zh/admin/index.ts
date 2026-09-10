import overview from './overview'
import accounts from './accounts'
import ops from './ops'
import settings from './settings'
import audit from './audit'
import promptAudit from './promptAudit'
import channelIq from './channelIq'

export default {
  promptAudit,
  channelIq,
  ...overview,
  ...accounts,
  ...ops,
  ...settings,
  ...audit,
}
