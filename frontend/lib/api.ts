import {
  ApiError,
  GeneratedLetters,
  GenerateLetterRequest,
  VisitorStatus
} from './types';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export class ApiClient {
  private baseUrl: string;
  private defaultHeaders: HeadersInit;

  constructor() {
    this.baseUrl = API_BASE_URL;
    this.defaultHeaders = {
      'Content-Type': 'application/json',
    };
  }

  /**
   * Fetch wrapper avec retry logic
   */
  private async fetchWithRetry<T>(
    url: string,
    options: RequestInit = {},
    retries = 3
  ): Promise<T> {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 30000); // 30s timeout

    try {
      const response = await fetch(url, {
        ...options,
        headers: {
          ...this.defaultHeaders,
          ...options.headers,
        },
        credentials: 'include', // Pour les cookies de session
        signal: controller.signal,
      });

      clearTimeout(timeout);

      // Gestion des erreurs HTTP
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({
          message: 'Une erreur est survenue',
        }));

        throw {
          success: false,
          message: errorData.message || `Erreur ${response.status}`,
          code: errorData.code,
          statusCode: response.status,
        } as ApiError;
      }

      return await response.json();
    } catch (error) {
      clearTimeout(timeout);

      // Retry logic pour erreurs réseau
      if (retries > 0 && this.isRetryable(error)) {
        await this.delay(1000 * (4 - retries)); // Exponential backoff
        return this.fetchWithRetry<T>(url, options, retries - 1);
      }

      throw error;
    }
  }

  /**
   * Détermine si l'erreur est retryable
   */
  private isRetryable(error: any): boolean {
    // Ne pas retry les erreurs client (4xx)
    if (error.statusCode && error.statusCode >= 400 && error.statusCode < 500) {
      return false;
    }

    // Retry erreurs réseau et serveur (5xx)
    return true;
  }

  /**
   * Delay helper pour retry
   */
  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  /**
   * GET request
   */
  async get<T>(endpoint: string, params?: Record<string, string>): Promise<T> {
    const url = new URL(`${this.baseUrl}${endpoint}`);

    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        url.searchParams.append(key, value);
      });
    }

    return this.fetchWithRetry<T>(url.toString(), {
      method: 'GET',
    });
  }

  /**
   * POST request
   */
  async post<T>(endpoint: string, data?: any): Promise<T> {
    return this.fetchWithRetry<T>(`${this.baseUrl}${endpoint}`, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  /**
   * PUT request
   */
  async put<T>(endpoint: string, data?: any): Promise<T> {
    return this.fetchWithRetry<T>(`${this.baseUrl}${endpoint}`, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  /**
   * DELETE request
   */
  async delete<T>(endpoint: string): Promise<T> {
    return this.fetchWithRetry<T>(`${this.baseUrl}${endpoint}`, {
      method: 'DELETE',
    });
  }
}

// Export singleton
export const api = new ApiClient();

// Export helpers typés (seront enrichis dans les phases suivantes)
export const cvApi = {
  getCV: (theme?: string) =>
    api.get<any>('/api/cv', theme ? { theme } : undefined),
  getThemes: () =>
    api.get<any>('/api/cv/themes'),
};

export const healthApi = {
  check: () => api.get<{ status: string; timestamp: string }>('/health'),
};

// Letters API (Phase 3)
export const lettersApi = {
  generate: (data: GenerateLetterRequest) =>
    api.post<GeneratedLetters>('/api/v1/letters/generate', data),

  getById: (id: string) =>
    api.get<GeneratedLetters>(`/api/v1/letters/${id}`),

  downloadPDF: async (id: string, type: 'motivation' | 'anti' | 'both') => {
    const response = await fetch(
      `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}/api/v1/letters/${id}/pdf?type=${type}`,
      {
        method: 'GET',
        credentials: 'include',
      }
    );

    if (!response.ok) {
      throw new Error('Failed to download PDF');
    }

    return response.blob();
  },
};

export const visitorsApi = {
  checkStatus: () =>
    api.get<VisitorStatus>('/api/v1/visitors/check'),
};

// Analytics API (Phase 4)
export const analyticsApi = {
  getStats: (period?: 'day' | 'week' | 'month') =>
    api.get<any>('/api/v1/analytics/stats', period ? { period } : undefined),

  getThemes: () =>
    api.get<any>('/api/v1/analytics/themes'),

  getLetters: (period?: 'day' | 'week' | 'month') =>
    api.get<any>('/api/v1/analytics/letters', period ? { period } : undefined),

  getHeatmap: (page?: string) =>
    api.get<any>('/api/v1/analytics/heatmap', page ? { page } : undefined),
};

// Profile Detection API (Phase 5 - Feature 3)
export const profileApi = {
  // Détection manuelle du profil (debug)
  detect: () =>
    api.get<import('./types').ProfileDetection>('/api/v1/profile/detect'),

  // Récupérer le profil actuel depuis le middleware/cache
  getCurrent: () =>
    api.get<import('./types').ProfileDetection>('/api/v1/profile/current'),

  // Vérifier le statut de bypass
  getBypassStatus: () =>
    api.get<{ success: boolean; bypass: boolean }>('/api/v1/profile/bypass'),

  // Activer le bypass manuellement (admin only)
  enableBypass: () =>
    api.post<{ success: boolean; message: string }>('/api/v1/profile/bypass/enable'),

  // Récupérer les stats de profils détectés
  getStats: () =>
    api.get<import('./types').ProfileStats>('/api/v1/profile/stats'),
};

// GitHub API (Phase 5 - Feature 1)
export const githubApi = {
  // Obtenir l'URL d'authentification OAuth GitHub
  getAuthURL: () =>
    api.get<import('./types').GitHubAuthURLResponse>('/api/v1/github/auth-url'),

  // Déclencher une synchronisation manuelle
  sync: (username: string) =>
    api.post<import('./types').GitHubSyncResponse>('/api/v1/github/sync', { username }),

  // Récupérer le statut de connexion GitHub
  getStatus: (username: string) =>
    api.get<import('./types').GitHubSyncStatus>('/api/v1/github/status', { username }),

  // Récupérer la liste des repos GitHub
  getRepos: (username: string, includePrivate = false) =>
    api.get<import('./types').GitHubReposResponse>('/api/v1/github/repos', {
      username,
      include_private: includePrivate.toString(),
    }),

  // Déconnecter GitHub
  disconnect: (username: string) =>
    api.delete<import('./types').GitHubDisconnectResponse>(
      `/api/v1/github/disconnect?username=${username}`
    ),
};

// Timeline API (Phase 5 - Feature 2)
export const timelineApi = {
  getTimeline: async (category?: string, from?: string, to?: string) => {
    const params: Record<string, string> = {};
    if (category) params.category = category;
    if (from) params.from = from;
    if (to) params.to = to;

    const response = await api.get<import('./types').TimelineResponse>(
      '/api/v1/timeline',
      Object.keys(params).length > 0 ? params : undefined
    );
    return response.data;
  },

  getCategories: async () => {
    const response = await api.get<import('./types').TimelineCategoriesResponse>(
      '/api/v1/timeline/categories'
    );
    return response.categories;
  },

  getMilestones: async () => {
    const response = await api.get<import('./types').TimelineMilestonesResponse>(
      '/api/v1/timeline/milestones'
    );
    return response.milestones;
  },
};
