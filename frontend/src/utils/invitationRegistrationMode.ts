export type InvitationRegistrationMode = 'redeem_code' | 'affiliate_link' | 'both'

export const DEFAULT_INVITATION_REGISTRATION_MODE: InvitationRegistrationMode = 'redeem_code'

export function normalizeInvitationRegistrationMode(value?: unknown): InvitationRegistrationMode {
  const mode = typeof value === 'string' ? value.trim() : ''
  switch (mode) {
    case 'affiliate_link':
    case 'both':
      return mode
    default:
      return DEFAULT_INVITATION_REGISTRATION_MODE
  }
}

export function invitationRegistrationModeSupportsAffiliateLink(value?: unknown): boolean {
  const mode = normalizeInvitationRegistrationMode(value)
  return mode === 'affiliate_link' || mode === 'both'
}
