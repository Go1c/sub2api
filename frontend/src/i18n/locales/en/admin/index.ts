import overview from './overview'
import accounts from './accounts'
import ops from './ops'
import settings from './settings'
import audit from './audit'

export default {
  ...overview,
  ...accounts,
  ...ops,
  ...settings,
  ...audit,
}
