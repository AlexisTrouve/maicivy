import http from 'k6/http'
import { check, sleep } from 'k6'
import { Rate, Trend, Counter } from 'k6/metrics'

// Custom metrics
const errorRate = new Rate('errors')
const letterGenerationTime = new Trend('letter_generation_time')
const successfulGenerations = new Counter('successful_generations')
const rateLimitHits = new Counter('rate_limit_hits')

// Test configuration
export const options = {
  // Stages for letter generation (lower concurrency due to AI costs)
  stages: [
    { duration: '1m', target: 5 },    // Ramp up to 5 users
    { duration: '2m', target: 20 },   // Ramp up to 20 users
    { duration: '3m', target: 20 },   // Stay at 20 users
    { duration: '1m', target: 0 },    // Ramp down
  ],

  // More relaxed thresholds for AI-dependent endpoint
  thresholds: {
    'http_req_duration': [
      'p(95)<15000',  // 95% < 15s (AI API can be slow)
      'p(99)<30000',  // 99% < 30s
    ],
    'http_req_failed': ['rate<0.05'], // Error rate < 5%
    'errors': ['rate<0.05'],
  },
}

const BASE_URL = __ENV.API_URL || 'http://localhost:3000'

// Test companies
const companies = [
  'Google',
  'Microsoft',
  'Apple',
  'Amazon',
  'Meta',
  'Netflix',
  'Uber',
  'Airbnb',
  'Spotify',
  'Stripe',
]

export default function () {
  // Simulate visitor session
  const visitorID = `visitor-${__VU}-${__ITER}`

  // Test 1: Check if letters can be generated (access gate)
  testAccessGate(visitorID)

  // Test 2: Generate letter (if access granted)
  const company = companies[Math.floor(Math.random() * companies.length)]
  testGenerateLetter(visitorID, company)

  // Test 3: Get letter history
  testGetLetterHistory(visitorID)

  // Longer think time for letter generation
  sleep(Math.random() * 3 + 2) // 2-5 seconds
}

function testAccessGate(visitorID) {
  const url = `${BASE_URL}/api/letters/access-gate`

  const res = http.get(url, {
    headers: {
      'Cookie': `session_id=${visitorID}`,
    },
    tags: { name: 'AccessGate' },
  })

  check(res, {
    'Access gate status is 200': (r) => r.status === 200,
    'Access gate response time < 100ms': (r) => r.timings.duration < 100,
  })
}

function testGenerateLetter(visitorID, companyName) {
  const url = `${BASE_URL}/api/letters/generate`

  const payload = JSON.stringify({
    company_name: companyName,
    letter_type: Math.random() > 0.5 ? 'motivation' : 'anti_motivation',
  })

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Cookie': `session_id=${visitorID}`,
    },
    tags: { name: 'GenerateLetter' },
    timeout: '30s', // AI can take time
  }

  const res = http.post(url, payload, params)

  const success = check(res, {
    'Letter status is 200 or 429': (r) => r.status === 200 || r.status === 429,
    'Letter generation time < 20s': (r) => r.timings.duration < 20000,
    'Letter has content': (r) => {
      if (r.status === 200) {
        try {
          const body = JSON.parse(r.body)
          return body.motivation_letter || body.anti_motivation_letter
        } catch {
          return false
        }
      }
      return true // 429 is acceptable (rate limited)
    },
  })

  // Track rate limiting
  if (res.status === 429) {
    rateLimitHits.add(1)
  }

  // Record metrics
  errorRate.add(!success)
  if (res.status === 200) {
    letterGenerationTime.add(res.timings.duration)
    successfulGenerations.add(1)
  }
}

function testGetLetterHistory(visitorID) {
  const url = `${BASE_URL}/api/letters/history`

  const res = http.get(url, {
    headers: {
      'Cookie': `session_id=${visitorID}`,
    },
    tags: { name: 'LetterHistory' },
  })

  check(res, {
    'History status is 200': (r) => r.status === 200,
    'History response time < 500ms': (r) => r.timings.duration < 500,
  })
}

// Spike test scenario (stress test)
export const spikeOptions = {
  stages: [
    { duration: '10s', target: 5 },    // Warm up
    { duration: '30s', target: 50 },   // Spike to 50 users
    { duration: '1m', target: 50 },    // Stay at spike
    { duration: '30s', target: 0 },    // Recover
  ],
}

// Soak test scenario (long duration, moderate load)
export const soakOptions = {
  stages: [
    { duration: '2m', target: 10 },    // Ramp up
    { duration: '30m', target: 10 },   // Stay for 30 minutes
    { duration: '2m', target: 0 },     // Ramp down
  ],
}

export function setup() {
  console.log('Starting Letters API load test')
  console.log(`Base URL: ${BASE_URL}`)
  console.log('Note: Letter generation includes AI API calls (slower)')
}

export function teardown(data) {
  console.log('Load test completed')
  console.log(`Rate limit hits: ${rateLimitHits.value}`)
}

// Run with:
// k6 run letters-load-test.js
//
// Run spike test:
// k6 run --config spikeOptions letters-load-test.js
//
// Expected results:
// ✓ http_req_duration............: avg=8s     min=50ms  med=5s    max=25s   p(95)=12s
// ✓ letter_generation_time.......: avg=7s     p(95)=11s
// ✓ successful_generations.......: 150
// ✓ rate_limit_hits..............: 25
//
// Notes:
// - AI API latency: 5-15 seconds typical
// - Rate limiting expected (5/day per visitor)
// - Some 429 responses are normal and acceptable
