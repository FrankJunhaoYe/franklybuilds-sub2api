import lottery from './lottery'
import adminAccessKeys from './adminAccessKeys'
import mcp from './mcp'
import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import supportTickets from './supportTickets'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'

export default {
  lottery,
  adminAccessKeys,
  mcp,
  ...landing,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...supportTickets,
  ...batchImage,
  admin,
  ...misc,
}
