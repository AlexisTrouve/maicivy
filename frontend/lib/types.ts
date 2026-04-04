// Types communs API
export interface ApiResponse<T> {
  data: T;
  success: boolean;
  message?: string;
}

export interface ApiError {
  success: false;
  message: string;
  code?: string;
  statusCode: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  meta: {
    page: number;
    limit: number;
    total: number;
    totalPages: number;
  };
}

// Link type for detail modals
export interface DetailLink {
  type: 'github' | 'demo' | 'website' | 'linkedin' | 'other';
  url: string;
  label?: string;
}

// CV Types
export interface Experience {
  id: string;
  title: string;
  company: string;
  description: string;
  startDate: string;
  endDate?: string;
  technologies: string[];
  tags: string[];
  score?: number;
  category?: string;
  featured?: boolean;
  // Extended fields for detail modal
  catchphrase?: string;
  functionalDescription?: string;
  technicalDescription?: string;
  images?: string[];
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  links?: any[]; // Backend format: {name, url, icon}, mapped to DetailLink in components
}

// Skill levels matching backend SkillLevel type
export type SkillLevel = 'beginner' | 'intermediate' | 'advanced' | 'expert';

export interface Skill {
  id: string;
  name: string;
  level: SkillLevel;
  category: string;
  yearsExperience: number;
  score?: number;
}

export interface Project {
  id: string;
  title: string;
  description: string;
  githubUrl?: string;
  demoUrl?: string;
  technologies: string[];
  stars?: number;
  language?: string;
  featured: boolean;
  score?: number;
  // Extended fields for detail modal
  category?: string;
  catchphrase?: string;
  inProgress?: boolean;
  functionalDescription?: string;
  technicalDescription?: string;
  images?: string[];
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  links?: any[]; // Backend format: {name, url, icon}, mapped to DetailLink in components
}

export interface CVData {
  theme: string;
  experiences: Experience[];
  skills: Skill[];
  projects: Project[];
}

export interface Theme {
  id: string;
  name: string;
  description: string;
  icon: string;
}

// Letters Types (Phase 3)
export interface CompanyInfo {
  industry?: string;
  description?: string;
  website?: string;
  size?: string;
  location?: string;
}

export interface GeneratedLetters {
  id: string;
  companyName: string;
  motivationLetter: string;
  antiMotivationLetter: string;
  companyInfo?: CompanyInfo;
  createdAt: string;
  updatedAt?: string;
}

export interface LetterHistoryItem {
  id: string;
  companyName: string;
  createdAt: string;
}

export interface GenerateLetterRequest {
  company_name: string;
  lang?: string; // 'fr' or 'en'
}

export interface GenerateLetterResponse extends GeneratedLetters {}

// Platform message (Malt, LinkedIn, Upwork...)
export interface GeneratePlatformMessageRequest {
  mission: string;   // description de la mission copiée-collée
  platform?: 'malt' | 'linkedin' | 'upwork';
  tjm?: number;      // tarif journalier en euros
  lang?: string;
}

export interface PlatformMessageResponse {
  content: string;
  platform: string;
  tokens_used: number;
  estimated_cost: number;
  model: string;
}

export interface VisitorStatus {
  visitCount: number;
  hasAccess: boolean;
  profileDetected?: string;
  remainingVisits: number;
  sessionId: string;
}

// Analytics Types (Phase 4)
export interface AnalyticsStats {
  totalVisitors: number;
  totalPageViews: number;
  totalLetters: number;
  conversionRate: number;
  activeVisitors?: number;
  period?: 'day' | 'week' | 'month';
  avgSessionDuration?: number;
}

export interface ThemeStat {
  theme: string;
  count: number;
  percentage: number;
}

export interface ThemeStatsResponse {
  themes: ThemeStat[];
  total: number;
}

export interface LettersStat {
  date: string;
  count: number;
}

export interface LettersStatsResponse {
  total: number;
  history: LettersStat[];
  period?: 'day' | 'week' | 'month';
}

export interface HeatmapPoint {
  x: number;
  y: number;
  intensity: number;
  element?: string;
}

export interface HeatmapData {
  points: HeatmapPoint[];
  maxIntensity: number;
}

export interface RealtimeData {
  currentVisitors: number;
  timestamp: number;
  recentEvents?: AnalyticsEvent[];
}

export interface AnalyticsEvent {
  id: string;
  type: 'page_view' | 'letter_generated' | 'pdf_exported' | 'interaction';
  timestamp: string;
  metadata?: Record<string, any>;
}

// Profile Detection Types (Phase 5 - Feature 3)
export type ProfileType = 'recruiter' | 'cto' | 'tech_lead' | 'ceo' | 'developer' | 'other';

export interface DeviceInfo {
  browser: string;
  os: string;
  deviceType: string; // 'mobile', 'tablet', 'desktop', 'tool'
  isBot: boolean;
}

export interface EnrichmentData {
  company_name?: string;
  company_domain?: string;
  company_type?: string;
  industry?: string;
  company_size?: string;
  city?: string;
  country?: string;
  job_role?: string;
  job_title?: string;
}

export interface ProfileDetection {
  profile_type: ProfileType;
  confidence: number; // 0-100
  enrichment_data?: EnrichmentData;
  device_info?: DeviceInfo;
  detection_sources?: string[];
  bypass_enabled?: boolean;
}

export interface ProfileStats {
  stats_by_type: Array<{
    profile_type: string;
    count: number;
    avg_confidence: number;
  }>;
  total_detected: number;
  total_visitors: number;
  detection_rate: number;
}

// GitHub Types (Phase 5 - Feature 1)
export interface GitHubToken {
  access_token: string;
  token_type: string;
  expires_at: number;
}

export interface GitHubProfile {
  id: number;
  username: string;
  connected_at: number;
  synced_at: number;
  created_at: string;
  updated_at: string;
}

export interface GitHubRepository {
  id: number;
  username: string;
  repo_name: string;
  full_name: string;
  description: string;
  url: string;
  stars: number;
  language: string;
  topics: string[];
  is_private: boolean;
  pushed_at: string;
  created_at: string;
  updated_at: string;
}

export interface GitHubSyncStatus {
  connected: boolean;
  username?: string;
  last_sync: number;
  repo_count: number;
}

export interface GitHubAuthURLResponse {
  auth_url: string;
}

export interface GitHubCallbackResponse {
  success: boolean;
  username: string;
  connected_at: number;
}

export interface GitHubSyncRequest {
  username: string;
}

export interface GitHubSyncResponse {
  status: string;
  username: string;
}

export interface GitHubReposResponse {
  repositories: GitHubRepository[];
}

export interface GitHubDisconnectResponse {
  success: boolean;
  message: string;
}

// Timeline Types (Phase 5 - Feature 2)
export interface TimelineEvent {
  id: string;
  type: 'experience' | 'project';
  title: string;
  subtitle: string;
  content: string;
  startDate: string;
  endDate?: string;
  tags: string[];
  category: string;
  image?: string;
  isCurrent: boolean;
  duration?: string;
  githubUrl?: string;
  demoUrl?: string;
  stats?: {
    stars?: number;
    forks?: number;
    language?: string;
  };
}

export interface TimelineMilestone {
  id: string;
  title: string;
  description: string;
  date: string;
  icon: string;
  type: 'achievement' | 'career' | 'education' | 'project';
}

export interface TimelineStats {
  totalExperiences: number;
  totalProjects: number;
  categoriesBreakdown: Record<string, number>;
  yearsOfExperience: number;
  topTechnologies: TechnologyCount[];
}

export interface TechnologyCount {
  name: string;
  count: number;
}

export interface TimelineFilters {
  category?: string;
  type?: 'experience' | 'project' | 'all';
  from?: string;
  to?: string;
}

export interface TimelineResponse {
  success: boolean;
  data: {
    events: TimelineEvent[];
    total: number;
    stats: TimelineStats;
  };
}

export interface TimelineCategoriesResponse {
  success: boolean;
  categories: string[];
  total: number;
}

export interface TimelineMilestonesResponse {
  success: boolean;
  milestones: TimelineMilestone[];
  total: number;
}

// Activity Feed Types (Auto-sync from ProjectTracker)
export interface ActivityCommit {
  sha: string;
  message: string;
  date: string;
  author: string;
}

export interface ActivityProject {
  id: number;
  name: string;
  description: string;
  repo_url: string;
  category: string;
  showcase: boolean;
  languages: string[];
  commits_7d: number;
  commits_30d: number;
  recent_commits: ActivityCommit[];
  last_activity: string;
  created_at: string;
  updated_at: string;
}

export interface ActivityStats {
  id: number;
  total_commits_30d: number;
  active_projects: number;
  top_languages: string[];
  last_updated: string;
  created_at: string;
  updated_at: string;
}

export interface ActivityFeedResponse {
  last_updated: string;
  projects: ActivityProject[];
  stats: ActivityStats;
}

export interface ActivityProjectsResponse {
  projects: ActivityProject[];
}

// Blog Types
export interface BlogCommitRef {
  sha: string;
  message: string;
  date: string;
  project: string;
}

export interface BlogPost {
  id: number;
  slug: string;
  title: string;
  summary: string;
  content: string;
  content_html: string;
  project_name: string;
  tags: string[];
  generated_from_commits: BlogCommitRef[];
  cover_image_url?: string;
  reading_time_minutes: number;
  published: boolean;
  published_at?: string;
  created_at: string;
  updated_at: string;
}

export interface BlogPostListResponse {
  posts: BlogPost[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface BlogGenerateRequest {
  project_name: string;
  commit_shas?: string[];
  auto_select: boolean;
}

// Git Stats Types (Gitea)
export interface GitDayStat {
  date: string;      // "2026-03-15"
  commits: number;
  additions: number;
  deletions: number;
}

export interface GitRepoStat {
  name: string;
  description: string;
  language: string;
  stars: number;
  updatedAt: string;
}

export interface GitStatsResponse {
  daily: GitDayStat[];
  repos: GitRepoStat[];
  totalCommits: number;
  totalAdded: number;
  totalDeleted: number;
  activeRepos: number;
  period: string;
}
