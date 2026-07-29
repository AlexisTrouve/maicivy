import { rest } from 'msw'

// Mock API handlers for MSW v1
export const handlers = [
  // Mock GET /api/cv
  rest.get('/api/cv', (req, res, ctx) => {
    return res(
      ctx.json({
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
    )
  }),

  // Mock GET /api/cv/themes and /api/v1/cv/themes
  rest.get('*/api/cv/themes', (req, res, ctx) => {
    return res(
      ctx.json([
        {
          id: 'technical',
          name: 'Technique',
          description: 'Focus backend et DevOps',
          icon: '⚙️',
        },
        {
          id: 'creative',
          name: 'Créatif',
          description: 'Frontend et design',
          icon: '🎨',
        },
        {
          id: 'business',
          name: 'Business',
          description: 'Gestion de projet',
          icon: '💼',
        },
      ])
    )
  }),

  rest.get('*/api/v1/cv/themes', (req, res, ctx) => {
    return res(
      ctx.json([
        {
          id: 'technical',
          name: 'Technique',
          description: 'Focus backend et DevOps',
          icon: '⚙️',
        },
        {
          id: 'creative',
          name: 'Créatif',
          description: 'Frontend et design',
          icon: '🎨',
        },
        {
          id: 'business',
          name: 'Business',
          description: 'Gestion de projet',
          icon: '💼',
        },
      ])
    )
  }),

  // Mock POST /api/letters/generate
  rest.post('/api/letters/generate', (req, res, ctx) => {
    const body = req.body as any

    return res(
      ctx.json({
        id: 'test-letter-id',
        companyName: body.companyName,
        jobTitle: body.jobTitle,
        content: 'Test generated letter content',
        generatedAt: new Date().toISOString(),
      })
    )
  }),

  // Mock GET /api/analytics/stats
  rest.get('/api/analytics/stats', (req, res, ctx) => {
    return res(
      ctx.json({
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
    )
  }),

  // GitHub API handlers
  rest.get('*/api/v1/github/auth-url', (req, res, ctx) => {
    return res(
      ctx.json({
        auth_url: 'https://github.com/login/oauth/authorize?client_id=test123',
      })
    )
  }),

  rest.get('*/api/v1/github/status', (req, res, ctx) => {
    const username = req.url.searchParams.get('username')
    return res(
      ctx.json({
        connected: true,
        username: username || 'testuser',
        last_sync: Math.floor(Date.now() / 1000) - 3600,
        repo_count: 15,
      })
    )
  }),

  rest.post('*/api/v1/github/sync', (req, res, ctx) => {
    return res(
      ctx.json({
        status: 'success',
        message: 'Sync completed',
      })
    )
  }),

  rest.delete('*/api/v1/github/disconnect', (req, res, ctx) => {
    return res(
      ctx.json({
        success: true,
        message: 'Disconnected successfully',
      })
    )
  }),

  rest.get('*/api/v1/github/repos', (req, res, ctx) => {
    const includePrivate = req.url.searchParams.get('include_private') === 'true'

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

    return res(
      ctx.json({
        repositories: includePrivate ? repos : repos.filter(r => !r.is_private),
      })
    )
  }),

  // Visitor API handlers
  rest.get('*/api/v1/visitors/check', (req, res, ctx) => {
    return res(
      ctx.json({
        visitCount: 1,
        hasAccess: true,
        remainingVisits: 2,
        sessionId: 'test-session-abc',
      })
    )
  }),

  // Profile Detection API handlers
  rest.get('*/api/v1/profile/current', (req, res, ctx) => {
    return res(
      ctx.json({
        profile_type: 'developer',
        confidence: 50,
        bypass_enabled: false,
      })
    )
  }),

  rest.get('*/api/v1/profile/detect', (req, res, ctx) => {
    return res(
      ctx.json({
        profile_type: 'developer',
        confidence: 50,
        bypass_enabled: false,
      })
    )
  }),

  rest.get('*/api/v1/profile/bypass', (req, res, ctx) => {
    return res(
      ctx.json({
        success: true,
        bypass: false,
      })
    )
  }),

  rest.get('*/api/v1/profile/stats', (req, res, ctx) => {
    return res(
      ctx.json({
        stats_by_type: [],
        total_detected: 0,
        total_visitors: 0,
        detection_rate: 0,
      })
    )
  }),

  // Timeline API handlers
  rest.get('*/api/v1/timeline', (req, res, ctx) => {
    const category = req.url.searchParams.get('category')

    const allEvents = [
      {
        id: '1',
        type: 'experience',
        title: 'Senior Developer',
        subtitle: 'Tech Corp',
        content: 'Led development team',
        startDate: '2020-01-01',
        endDate: '2023-12-31',
        tags: ['React', 'Node.js'],
        category: 'backend',
        isCurrent: false,
      },
      {
        id: '2',
        type: 'project',
        title: 'E-commerce Platform',
        subtitle: 'Personal Project',
        content: 'Built scalable platform',
        startDate: '2022-06-01',
        tags: ['Go', 'PostgreSQL'],
        category: 'fullstack',
        isCurrent: true,
      },
    ]

    let filteredEvents = allEvents
    if (category && category !== 'all') {
      filteredEvents = allEvents.filter(event => event.category === category)
    }

    return res(
      ctx.json({
        success: true,
        data: {
          events: filteredEvents,
          total: filteredEvents.length,
          stats: {
            totalExperiences: filteredEvents.filter(e => e.type === 'experience').length,
            totalProjects: filteredEvents.filter(e => e.type === 'project').length,
            categoriesBreakdown: { backend: 1, fullstack: 1 },
            yearsOfExperience: 4,
            topTechnologies: [
              { name: 'React', count: 5 },
              { name: 'Node.js', count: 4 },
            ],
          },
        },
      })
    )
  }),

  rest.get('*/api/v1/timeline/categories', (req, res, ctx) => {
    return res(
      ctx.json({
        success: true,
        categories: ['backend', 'frontend', 'fullstack'],
        total: 3,
      })
    )
  }),

  rest.get('*/api/v1/timeline/milestones', (req, res, ctx) => {
    return res(
      ctx.json({
        success: true,
        milestones: [
          {
            id: '1',
            title: 'Started Career',
            description: 'First job as developer',
            date: '2019-01-01',
            icon: 'briefcase',
            type: 'career',
          },
        ],
        total: 1,
      })
    )
  }),

  // Add more handlers as needed
]
