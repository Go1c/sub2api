import overview from './overview'
import channels from './channels'
import accounts from './accounts'
import channelIq from './channelIq'
import channelTurnState from './channelTurnState'
import requestHealth from './requestHealth'
import resources from './resources'
import ops from './ops'
import settings from './settings'
import audit from './audit'
import promptAudit from './promptAudit'
import plugins from './plugins'

export default {
  ...overview,
  ...channels,
  ...accounts,
  ...channelIq,
  ...channelTurnState,
  ...requestHealth,
  ...resources,
  ...ops,
  ...settings,
  ...audit,
  ...promptAudit,
  ...plugins,
}
