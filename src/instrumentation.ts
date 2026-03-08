export async function register() {
  if (process.env.NEXT_RUNTIME === 'nodejs') {
    const { startTimeoutScheduler } = await import('@/lib/order/timeout');
    startTimeoutScheduler();
  }
}
