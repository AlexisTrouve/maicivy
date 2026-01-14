import http from 'k6/http'
import { check, sleep } from 'k6'
import { Rate, Trend, Counter } from 'k6/metrics'

// Custom metrics
const errorRate = new Rate('errors')
const responseTime = new Trend('response_time')
const successfulRequests = new Counter('successful_requests')

// Test configuration
export const options = {
  // Stages: ramp up, steady state, ramp down
  stages: [
    { duration: '30s', target: 20 },   // Ramp up to 20 users
    { duration: '2m', target: 100 },   // Ramp up to 100 users
    { duration: '3m', target: 100 },   // Stay at 100 users for 3 minutes
    { duration: '30s', target: 0 },    // Ramp down to 0
  ],

  // Thresholds: define success criteria
  thresholds: {
    'http_req_duration': [
      'p(50)<100',   // 50% of requests < 100ms
      'p(95)<500',   // 95% of requests < 500ms
      'p(99)<1000',  // 99% of requests < 1s
    ],
    'http_req_failed': ['rate<0.01'], // Error rate < 1%
    'errors': ['rate<0.01'],           // Custom error rate < 1%
  },
}

// Base URL
const BASE_URL = __ENV.API_URL || 'http://localhost:3000'

// Test scenarios
const themes = ['backend', 'frontend', 'fullstack', 'devops', 'cpp', 'artistic']

export default function () {
  // Randomly select a theme
  const theme = themes[Math.floor(Math.random() * themes.length)]

  // Test 1: Get CV by theme
  testGetCV(theme)

  // Test 2: Get experiences
  testGetExperiences()

  // Test 3: Get skills
  testGetSkills()

  // Test 4: Get projects
  testGetProjects()

  // Think time (simulate real user behavior)
  sleep(Math.random() * 2 + 1) // 1-3 seconds
}

function testGetCV(theme) {
  const url = `${BASE_URL}/api/cv?theme=${theme}`

  const res = http.get(url, {
    tags: { name: 'GetCV' },
  })

  const success = check(res, {
    'CV status is 200': (r) => r.status === 200,
    'CV response time < 500ms': (r) => r.timings.duration < 500,
    'CV has experiences': (r) => {
      try {
        const body = JSON.parse(r.body)
        return body.experiences && body.experiences.length > 0
      } catch {
        return false
      }
    },
  })

  // Record metrics
  errorRate.add(!success)
  responseTime.add(res.timings.duration)
  if (success) {
    successfulRequests.add(1)
  }
}

function testGetExperiences() {
  const url = `${BASE_URL}/api/experiences?page=1&limit=20`

  const res = http.get(url, {
    tags: { name: 'GetExperiences' },
  })

  const success = check(res, {
    'Experiences status is 200': (r) => r.status === 200,
    'Experiences response time < 300ms': (r) => r.timings.duration < 300,
    'Experiences has pagination': (r) => {
      try {
        const body = JSON.parse(r.body)
        return body.meta && body.meta.page && body.meta.total
      } catch {
        return false
      }
    },
  })

  errorRate.add(!success)
  responseTime.add(res.timings.duration)
  if (success) {
    successfulRequests.add(1)
  }
}

function testGetSkills() {
  const url = `${BASE_URL}/api/skills?limit=50`

  const res = http.get(url, {
    tags: { name: 'GetSkills' },
  })

  const success = check(res, {
    'Skills status is 200': (r) => r.status === 200,
    'Skills response time < 200ms': (r) => r.timings.duration < 200,
  })

  errorRate.add(!success)
  responseTime.add(res.timings.duration)
  if (success) {
    successfulRequests.add(1)
  }
}

function testGetProjects() {
  const url = `${BASE_URL}/api/projects?page=1&limit=20`

  const res = http.get(url, {
    tags: { name: 'GetProjects' },
  })

  const success = check(res, {
    'Projects status is 200': (r) => r.status === 200,
    'Projects response time < 300ms': (r) => r.timings.duration < 300,
  })

  errorRate.add(!success)
  responseTime.add(res.timings.duration)
  if (success) {
    successfulRequests.add(1)
  }
}

// Setup function (runs once before test)
export function setup() {
  console.log('Starting CV API load test')
  console.log(`Base URL: ${BASE_URL}`)
  console.log('Targets: p95 < 500ms, error rate < 1%')
}

// Teardown function (runs once after test)
export function teardown(data) {
  console.log('Load test completed')
}

// Run with:
// k6 run cv-load-test.js
//
// Or with custom API URL:
// k6 run --env API_URL=http://your-server.com cv-load-test.js
//
// With output to InfluxDB:
// k6 run --out influxdb=http://localhost:8086/k6 cv-load-test.js
//
// Expected results:
// ✓ http_req_duration............: avg=150ms  min=20ms  med=100ms  max=800ms  p(90)=300ms p(95)=450ms
// ✓ http_req_failed..............: 0.12%   ✓ 12  ✗ 9988
// ✓ iterations..................: 10000
// ✓ successful_requests.........: 9988
