// Mock for until-async to fix Jest/MSW compatibility issue
// until-async is a dependency of MSW that uses ESM exports

/**
 * Simplified mock of the 'until' function from until-async
 * Used by MSW to wait for async operations
 */
async function until(callback) {
  const maxAttempts = 100
  const delay = 10 // ms

  for (let i = 0; i < maxAttempts; i++) {
    try {
      const result = await callback()
      if (result) {
        return result
      }
    } catch (error) {
      // Ignore errors and continue trying
    }

    // Wait a bit before next attempt
    await new Promise(resolve => setTimeout(resolve, delay))
  }

  throw new Error('until() timeout: condition was not met')
}

module.exports = { until }
exports.until = until
