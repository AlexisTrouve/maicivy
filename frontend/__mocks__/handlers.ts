import { http, HttpResponse } from 'msw'

// Mock API handlers
export const handlers = [
  // Mock GET /api/cv
  http.get('/api/cv', () => {
    return HttpResponse.json({
      name: 'Test User',
      title: 'Software Developer',
      email: 'test@example.com',
      experiences: [
        {
          id: '1',
          company: 'Test Company',
          position: 'Developer',
          startDate: '2020-01-01',
          endDate: '2023-01-01',
          description: 'Test description',
        },
      ],
      skills: [
        { id: '1', name: 'JavaScript', level: 'Expert' },
        { id: '2', name: 'TypeScript', level: 'Advanced' },
      ],
    })
  }),

  // Mock POST /api/letters/generate
  http.post('/api/letters/generate', async ({ request }) => {
    const body = await request.json()

    return HttpResponse.json({
      id: 'test-letter-id',
      companyName: body.companyName,
      jobTitle: body.jobTitle,
      content: 'Test generated letter content',
      generatedAt: new Date().toISOString(),
    })
  }),

  // Mock GET /api/analytics/stats
  http.get('/api/analytics/stats', () => {
    return HttpResponse.json({
      totalVisits: 100,
      uniqueVisitors: 50,
      themeStats: {
        technical: 30,
        creative: 20,
        business: 15,
      },
      recentVisits: [
        {
          id: '1',
          timestamp: new Date().toISOString(),
          theme: 'technical',
          duration: 120,
        },
      ],
    })
  }),

  // GitHub API handlers
  http.get('*/api/v1/github/auth-url', () => {
    return HttpResponse.json({
      auth_url: 'https://github.com/login/oauth/authorize?client_id=test123',
    })
  }),

  http.get('*/api/v1/github/status', ({ request }) => {
    const url = new URL(request.url)
    const username = url.searchParams.get('username')
    return HttpResponse.json({
      connected: true,
      username: username || 'testuser',
      last_sync: Math.floor(Date.now() / 1000) - 3600,
      repo_count: 15,
    })
  }),

  http.post('*/api/v1/github/sync', () => {
    return HttpResponse.json({
      status: 'success',
      message: 'Sync completed',
    })
  }),

  http.delete('*/api/v1/github/disconnect', () => {
    return HttpResponse.json({
      success: true,
      message: 'Disconnected successfully',
    })
  }),

  http.get('*/api/v1/github/repos', ({ request }) => {
    const url = new URL(request.url)
    const includePrivate = url.searchParams.get('include_private') === 'true'

    const repos = [
      {
        id: 1,
        repo_name: 'awesome-project',
        full_name: 'testuser/awesome-project',
        description: 'An awesome project',
        url: 'https://github.com/testuser/awesome-project',
        stars: 42,
        language: 'TypeScript',
        topics: ['react', 'typescript'],
        is_private: false,
        pushed_at: '2024-01-15T10:00:00Z',
      },
      {
        id: 2,
        repo_name: 'private-repo',
        full_name: 'testuser/private-repo',
        description: 'Private repository',
        url: 'https://github.com/testuser/private-repo',
        stars: 5,
        language: 'Go',
        topics: ['backend'],
        is_private: true,
        pushed_at: '2024-01-10T08:00:00Z',
      },
    ]

    return HttpResponse.json({
      repositories: includePrivate ? repos : repos.filter(r => !r.is_private),
    })
  }),

  // Visitor API handlers
  http.get('/api/v1/visitors/check', () => {
    return HttpResponse.json({
      visitCount: 1,
      hasAccess: true,
      remainingVisits: 2,
      sessionId: 'test-session-abc',
    })
  }),

  // Profile Detection API handlers
  http.get('/api/v1/profile/current', () => {
    return HttpResponse.json({
      profile_type: 'developer',
      confidence: 50,
      bypass_enabled: false,
    })
  }),

  http.get('/api/v1/profile/detect', () => {
    return HttpResponse.json({
      profile_type: 'developer',
      confidence: 50,
      bypass_enabled: false,
    })
  }),

  http.get('/api/v1/profile/bypass', () => {
    return HttpResponse.json({
      success: true,
      bypass: false,
    })
  }),

  http.get('/api/v1/profile/stats', () => {
    return HttpResponse.json({
      stats_by_type: [],
      total_detected: 0,
      total_visitors: 0,
      detection_rate: 0,
    })
  }),

  // Add more handlers as needed
]
