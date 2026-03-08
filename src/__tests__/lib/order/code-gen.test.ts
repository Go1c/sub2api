import { describe, it, expect } from 'vitest';
import { generateRechargeCode } from '@/lib/order/code-gen';

describe('generateRechargeCode', () => {
  it('should generate code with s2p_ prefix', () => {
    const code = generateRechargeCode('cm1234567890');
    expect(code).toBe('s2p_cm1234567890');
  });

  it('should truncate long order IDs to fit 32 chars', () => {
    const longId = 'a'.repeat(50);
    const code = generateRechargeCode(longId);
    expect(code.length).toBeLessThanOrEqual(32);
    expect(code.startsWith('s2p_')).toBe(true);
  });

  it('should handle empty string', () => {
    const code = generateRechargeCode('');
    expect(code).toBe('s2p_');
  });
});
