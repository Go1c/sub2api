export function generateRechargeCode(orderId: string): string {
  const prefix = 's2p_';
  const maxIdLength = 32 - prefix.length; // 28
  const truncatedId = orderId.slice(0, maxIdLength);
  return `${prefix}${truncatedId}`;
}
