import { renderHook, waitFor, act } from '@testing-library/react'
import { useTimelineData } from '../useTimelineData'
import { server } from '@/__mocks__/server'
import { rest } from 'msw'

const mockTimelineData = {
  success: true,
  data: {
    events: [
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
    ],
    total: 2,
    stats: {
      totalExperiences: 1,
      totalProjects: 1,
      categoriesBreakdown: { backend: 1, fullstack: 1 },
      yearsOfExperience: 4,
      topTechnologies: [
        { name: 'React', count: 5 },
        { name: 'Node.js', count: 4 },
      ],
    },
  },
}

const mockCategories = {
  success: true,
  categories: ['backend', 'frontend', 'fullstack'],
  total: 3,
}

const mockMilestones = {
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
}

describe('useTimelineData', () => {
  beforeAll(() => server.listen())
  afterEach(() => server.resetHandlers())
  afterAll(() => server.close())

  it('should fetch timeline data on mount when autoFetch is true', async () => {
    server.use(
      rest.get('/api/v1/timeline', (req, res, ctx) => res(ctx.json(mockTimelineData))),
      rest.get('/api/v1/timeline/categories', (req, res, ctx) => res(ctx.json(mockCategories))),
      rest.get('/api/v1/timeline/milestones', (req, res, ctx) => res(ctx.json(mockMilestones)))
    )

    const { result } = renderHook(() => useTimelineData({ autoFetch: true }))

    expect(result.current.isLoading).toBe(true)

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.events).toHaveLength(2)
    expect(result.current.categories).toEqual(['backend', 'frontend', 'fullstack'])
    expect(result.current.milestones).toHaveLength(1)
    expect(result.current.stats?.totalExperiences).toBe(1)
    expect(result.current.error).toBeNull()
  })

  it('should not fetch on mount when autoFetch is false', async () => {
    const fetchSpy = jest.spyOn(global, 'fetch')

    renderHook(() => useTimelineData({ autoFetch: false }))

    expect(fetchSpy).not.toHaveBeenCalled()

    fetchSpy.mockRestore()
  })

  it('should filter events by type locally', async () => {
    server.use(
      rest.get('/api/v1/timeline', (req, res, ctx) => res(ctx.json(mockTimelineData))),
      rest.get('/api/v1/timeline/categories', (req, res, ctx) => res(ctx.json(mockCategories))),
      rest.get('/api/v1/timeline/milestones', (req, res, ctx) => res(ctx.json(mockMilestones)))
    )

    const { result } = renderHook(() =>
      useTimelineData({ autoFetch: true, type: 'experience' })
    )

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    // Should only have experience type
    expect(result.current.events).toHaveLength(1)
    expect(result.current.events[0].type).toBe('experience')
  })

  it('should filter events by category', async () => {
    const { result } = renderHook(() =>
      useTimelineData({ autoFetch: true, category: 'backend' })
    )

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.events).toHaveLength(1)
    expect(result.current.events[0].category).toBe('backend')
  })

  it('should apply filters dynamically using filter function', async () => {
    server.use(
      rest.get('/api/v1/timeline', (req, res, ctx) => res(ctx.json(mockTimelineData))),
      rest.get('/api/v1/timeline/categories', (req, res, ctx) => res(ctx.json(mockCategories))),
      rest.get('/api/v1/timeline/milestones', (req, res, ctx) => res(ctx.json(mockMilestones)))
    )

    const { result } = renderHook(() => useTimelineData({ autoFetch: true }))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    expect(result.current.events).toHaveLength(2)

    // Apply filter
    act(() => {
      result.current.filter({ type: 'project' })
    })

    expect(result.current.events).toHaveLength(1)
    expect(result.current.events[0].type).toBe('project')
  })

  it('should reset filters using reset function', async () => {
    server.use(
      rest.get('/api/v1/timeline', (req, res, ctx) => res(ctx.json(mockTimelineData))),
      rest.get('/api/v1/timeline/categories', (req, res, ctx) => res(ctx.json(mockCategories))),
      rest.get('/api/v1/timeline/milestones', (req, res, ctx) => res(ctx.json(mockMilestones)))
    )

    const { result } = renderHook(() => useTimelineData({ autoFetch: true }))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    // Apply filter
    act(() => {
      result.current.filter({ type: 'project' })
    })

    expect(result.current.events).toHaveLength(1)

    // Reset filters
    act(() => {
      result.current.reset()
    })

    expect(result.current.events).toHaveLength(2)
  })

  it('should refetch data when refetch is called', async () => {
    const callCounts = { timeline: 0 }

    server.use(
      rest.get('*/api/v1/timeline', (req, res, ctx) => {
        callCounts.timeline++
        return res(ctx.json(mockTimelineData))
      }),
      rest.get('*/api/v1/timeline/categories', (req, res, ctx) => res(ctx.json(mockCategories))),
      rest.get('*/api/v1/timeline/milestones', (req, res, ctx) => res(ctx.json(mockMilestones)))
    )

    const { result } = renderHook(() => useTimelineData({ autoFetch: true }))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    })

    const initialCount = callCounts.timeline
    expect(initialCount).toBeGreaterThan(0)

    await act(async () => {
      await result.current.refetch()
    })

    await waitFor(() => {
      expect(callCounts.timeline).toBeGreaterThan(initialCount)
    })
  })

  it('should handle API errors gracefully', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation()

    server.use(
      rest.get('*/api/v1/timeline', (req, res, ctx) => {
        return res(ctx.status(404), ctx.json({ message: 'Not found' }))
      })
    )

    const { result } = renderHook(() => useTimelineData({ autoFetch: true }))

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false)
    }, { timeout: 5000 })

    expect(result.current.error).toBeTruthy()
    expect(result.current.events).toHaveLength(0)

    consoleErrorSpy.mockRestore()
  })
})
