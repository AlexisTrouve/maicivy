/**
 * Artillery custom functions
 * Helper functions for load testing scenarios
 */

const crypto = require('crypto')

/**
 * Generate a unique session ID for each virtual user
 */
function setSessionID(context, events, done) {
  // Generate unique session ID
  context.vars.sessionID = `session-${crypto.randomUUID()}`

  // Set cookie header
  context.vars.sessionCookie = `session_id=${context.vars.sessionID}`

  return done()
}

/**
 * Generate random visitor data
 */
function generateVisitorData(context, events, done) {
  const userAgents = [
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36',
    'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36',
  ]

  const profiles = ['visitor', 'recruiter', 'tech_lead', 'developer']

  context.vars.userAgent = userAgents[Math.floor(Math.random() * userAgents.length)]
  context.vars.profileType = profiles[Math.floor(Math.random() * profiles.length)]

  return done()
}

/**
 * Log custom metrics
 */
function logMetrics(context, events, done) {
  events.emit('counter', 'custom.requests', 1)

  if (context.vars.statusCode === 200) {
    events.emit('counter', 'custom.success', 1)
  } else {
    events.emit('counter', 'custom.errors', 1)
  }

  return done()
}

/**
 * Simulate realistic think time
 */
function thinkTime(context, events, done) {
  // Random think time between 1-5 seconds
  const thinkTimeMs = Math.floor(Math.random() * 4000) + 1000

  setTimeout(() => {
    return done()
  }, thinkTimeMs)
}

/**
 * Validate CV response
 */
function validateCVResponse(context, events, done) {
  if (context.vars.cvResponse) {
    const cv = JSON.parse(context.vars.cvResponse)

    if (cv.experiences && cv.experiences.length > 0) {
      events.emit('counter', 'custom.valid_cv', 1)
    } else {
      events.emit('counter', 'custom.invalid_cv', 1)
    }
  }

  return done()
}

/**
 * Simulate cache hit/miss
 */
function simulateCacheCheck(context, events, done) {
  // 80% cache hit rate simulation
  const isCacheHit = Math.random() < 0.8

  if (isCacheHit) {
    context.vars.cacheHit = true
    events.emit('counter', 'custom.cache_hits', 1)
  } else {
    context.vars.cacheHit = false
    events.emit('counter', 'custom.cache_misses', 1)
  }

  return done()
}

/**
 * Handle rate limiting
 */
function handleRateLimit(context, events, done) {
  if (context.vars.statusCode === 429) {
    events.emit('counter', 'custom.rate_limited', 1)
    // Add exponential backoff
    context.vars.backoffMs = (context.vars.backoffMs || 1000) * 2
  } else {
    context.vars.backoffMs = 1000
  }

  return done()
}

/**
 * Generate random company name
 */
function randomCompany(context, events, done) {
  const companies = [
    'Google', 'Microsoft', 'Apple', 'Amazon', 'Meta',
    'Netflix', 'Uber', 'Airbnb', 'Spotify', 'Stripe',
    'Tesla', 'SpaceX', 'Salesforce', 'Adobe', 'Oracle',
  ]

  context.vars.companyName = companies[Math.floor(Math.random() * companies.length)]

  return done()
}

/**
 * Generate random CV theme
 */
function randomTheme(context, events, done) {
  const themes = [
    'backend', 'frontend', 'fullstack', 'devops',
    'cpp', 'artistic', 'data-science', 'mobile',
  ]

  context.vars.theme = themes[Math.floor(Math.random() * themes.length)]

  return done()
}

/**
 * Check response time and emit custom metrics
 */
function checkResponseTime(context, events, done) {
  const responseTime = context.vars.responseTime || 0

  if (responseTime < 100) {
    events.emit('counter', 'custom.fast_responses', 1)
  } else if (responseTime < 500) {
    events.emit('counter', 'custom.medium_responses', 1)
  } else {
    events.emit('counter', 'custom.slow_responses', 1)
  }

  return done()
}

module.exports = {
  setSessionID,
  generateVisitorData,
  logMetrics,
  thinkTime,
  validateCVResponse,
  simulateCacheCheck,
  handleRateLimit,
  randomCompany,
  randomTheme,
  checkResponseTime,
}
