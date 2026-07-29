/**
 * Test fixtures and helpers for frontend tests
 */

// Mock CV Data
export const mockCVData = {
  name: 'John Doe',
  title: 'Full Stack Developer',
  email: 'john.doe@example.com',
  phone: '+33 6 12 34 56 78',
  location: 'Paris, France',
  summary: 'Experienced full-stack developer with 5+ years of experience',

  experiences: [
    {
      id: '1',
      company: 'Tech Corp',
      position: 'Senior Developer',
      startDate: '2020-01-01',
      endDate: '2023-12-31',
      description: 'Led development of microservices architecture',
      skills: ['Go', 'React', 'PostgreSQL'],
    },
    {
      id: '2',
      company: 'Startup Inc',
      position: 'Full Stack Developer',
      startDate: '2018-06-01',
      endDate: '2019-12-31',
      description: 'Built RESTful APIs and responsive web applications',
      skills: ['Node.js', 'TypeScript', 'MongoDB'],
    },
  ],

  education: [
    {
      id: '1',
      institution: 'University of Technology',
      degree: 'Master in Computer Science',
      startDate: '2015-09-01',
      endDate: '2017-06-30',
      description: 'Specialized in distributed systems and cloud computing',
    },
  ],

  skills: [
    { id: '1', name: 'Go', level: 'Expert', category: 'Backend' },
    { id: '2', name: 'React', level: 'Advanced', category: 'Frontend' },
    { id: '3', name: 'TypeScript', level: 'Advanced', category: 'Language' },
    { id: '4', name: 'PostgreSQL', level: 'Intermediate', category: 'Database' },
    { id: '5', name: 'Docker', level: 'Advanced', category: 'DevOps' },
  ],

  projects: [
    {
      id: '1',
      name: 'E-commerce Platform',
      description: 'Built scalable e-commerce platform handling 10k+ daily users',
      technologies: ['Go', 'React', 'PostgreSQL', 'Redis'],
      url: 'https://github.com/johndoe/ecommerce',
      startDate: '2022-01-01',
      endDate: '2023-06-30',
    },
  ],
}

// Mock Letter Generation Request
export const mockLetterRequest = {
  companyName: 'Tech Innovations Inc',
  jobTitle: 'Senior Full Stack Developer',
  jobDescription: 'Looking for an experienced developer to lead our team',
  aiModel: 'claude',
}

// Mock Letter Generation Response
export const mockLetterResponse = {
  id: 'letter-123',
  companyName: 'Tech Innovations Inc',
  motivationLetter: 'Dear Hiring Manager,\n\nI am writing to express my strong interest in joining Tech Innovations Inc...',
  antiMotivationLetter: 'Dear Hiring Manager,\n\nI am writing to explain why I absolutely should NOT work at Tech Innovations Inc...',
  companyInfo: {
    name: 'Tech Innovations Inc',
    industry: 'Technology',
    description: 'A leading tech company',
  },
  createdAt: '2024-01-15T10:00:00Z',
  updatedAt: '2024-01-15T10:00:00Z',
}

// Mock Analytics Data
export const mockAnalyticsData = {
  totalVisits: 250,
  uniqueVisitors: 120,
  avgSessionDuration: 180, // seconds

  themeStats: {
    technical: 80,
    creative: 60,
    business: 40,
    academic: 30,
    general: 40,
  },

  recentVisits: [
    {
      id: '1',
      timestamp: '2024-01-15T10:00:00Z',
      theme: 'technical',
      duration: 240,
      interactions: 5,
    },
    {
      id: '2',
      timestamp: '2024-01-15T09:30:00Z',
      theme: 'creative',
      duration: 180,
      interactions: 3,
    },
  ],

  topInteractions: [
    { type: 'cv_view', count: 150 },
    { type: 'letter_generate', count: 45 },
    { type: 'project_click', count: 80 },
  ],
}

// Mock User Session
export const mockUserSession = {
  sessionId: 'session-abc-123',
  fingerprint: 'fp-xyz-789',
  visitCount: 1,
  theme: 'technical',
  startedAt: '2024-01-15T10:00:00Z',
}

// Helper function to create mock fetch response
export function createMockFetchResponse<T>(data: T, status = 200) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    json: async () => data,
    text: async () => JSON.stringify(data),
    headers: new Headers(),
  } as Response)
}

// Helper function to wait for async updates
export const waitFor = (ms: number) => new Promise(resolve => setTimeout(resolve, ms))

// Helper function to render with providers
import { render, RenderOptions } from '@testing-library/react'
import { ReactElement } from 'react'

export function renderWithProviders(
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) {
  // Add any providers here (ThemeProvider, QueryClientProvider, etc.)
  const AllTheProviders = ({ children }: { children: React.ReactNode }) => {
    return <>{children}</>
  }

  return render(ui, { wrapper: AllTheProviders, ...options })
}
