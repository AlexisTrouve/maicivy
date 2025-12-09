import { z } from 'zod';

/**
 * Validation schemas for maicivy frontend
 * Using Zod for type-safe validation
 */

// ============================================================================
// Letter Generation
// ============================================================================

export const letterGenerationSchema = z.object({
  companyName: z
    .string()
    .min(2, 'Le nom de l\'entreprise doit contenir au moins 2 caractères')
    .max(100, 'Le nom de l\'entreprise ne peut pas dépasser 100 caractères')
    .regex(
      /^[a-zA-ZÀ-ÿ0-9\s\-'\.&,()]+$/,
      'Seuls les lettres, chiffres, espaces et caractères courants sont autorisés'
    )
    .refine(
      (val) => !val.includes('<') && !val.includes('>'),
      'Les caractères < et > ne sont pas autorisés'
    ),
});

export type LetterGenerationInput = z.infer<typeof letterGenerationSchema>;

// ============================================================================
// CV Theme Selection
// ============================================================================

const validThemes = [
  'backend',
  'frontend',
  'fullstack',
  'cpp',
  'devops',
  'artistic',
  'data-science',
  'mobile',
] as const;

export const cvThemeSchema = z.object({
  theme: z.enum(validThemes, {
    errorMap: () => ({ message: 'Thème invalide' }),
  }),
});

export type CVThemeInput = z.infer<typeof cvThemeSchema>;

// ============================================================================
// Analytics Events
// ============================================================================

const validEventTypes = [
  'page_view',
  'click',
  'cv_theme_change',
  'letter_generated',
  'pdf_downloaded',
  'analytics_viewed',
] as const;

export const analyticsEventSchema = z.object({
  eventType: z.enum(validEventTypes, {
    errorMap: () => ({ message: 'Type d\'événement invalide' }),
  }),
  eventData: z.record(z.unknown()).optional(),
  timestamp: z.number().optional(),
});

export type AnalyticsEventInput = z.infer<typeof analyticsEventSchema>;

// ============================================================================
// Contact Form (if implemented)
// ============================================================================

export const contactFormSchema = z.object({
  name: z
    .string()
    .min(2, 'Le nom doit contenir au moins 2 caractères')
    .max(50, 'Le nom ne peut pas dépasser 50 caractères')
    .regex(/^[a-zA-ZÀ-ÿ\s\-']+$/, 'Seuls les lettres et espaces sont autorisés'),
  email: z
    .string()
    .email('Adresse email invalide')
    .max(100, 'L\'email ne peut pas dépasser 100 caractères'),
  subject: z
    .string()
    .min(3, 'Le sujet doit contenir au moins 3 caractères')
    .max(100, 'Le sujet ne peut pas dépasser 100 caractères'),
  message: z
    .string()
    .min(10, 'Le message doit contenir au moins 10 caractères')
    .max(1000, 'Le message ne peut pas dépasser 1000 caractères'),
});

export type ContactFormInput = z.infer<typeof contactFormSchema>;

// ============================================================================
// Search Query
// ============================================================================

export const searchSchema = z.object({
  query: z
    .string()
    .min(1, 'La recherche ne peut pas être vide')
    .max(100, 'La recherche ne peut pas dépasser 100 caractères')
    .refine(
      (val) => !val.includes('<script'),
      'Caractères non autorisés dans la recherche'
    ),
});

export type SearchInput = z.infer<typeof searchSchema>;

// ============================================================================
// Filter Options
// ============================================================================

export const filterSchema = z.object({
  dateFrom: z.string().datetime().optional(),
  dateTo: z.string().datetime().optional(),
  theme: z.enum(validThemes).optional(),
  limit: z.number().min(1).max(100).default(10),
  offset: z.number().min(0).default(0),
});

export type FilterInput = z.infer<typeof filterSchema>;

// ============================================================================
// URL Validation
// ============================================================================

export const urlSchema = z
  .string()
  .url('URL invalide')
  .refine(
    (val) => val.startsWith('http://') || val.startsWith('https://'),
    'L\'URL doit commencer par http:// ou https://'
  )
  .refine(
    (val) => !val.includes('localhost') && !val.includes('127.0.0.1'),
    'Les URLs locales ne sont pas autorisées'
  );

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Validates an email address
 */
export function isValidEmail(email: string): boolean {
  try {
    z.string().email().parse(email);
    return true;
  } catch {
    return false;
  }
}

/**
 * Validates a URL
 */
export function isValidURL(url: string): boolean {
  try {
    urlSchema.parse(url);
    return true;
  } catch {
    return false;
  }
}

/**
 * Sanitizes a string by removing HTML tags
 */
export function sanitizeString(input: string): string {
  // Remove HTML tags
  let sanitized = input.replace(/<[^>]*>/g, '');

  // Remove script tags and content
  sanitized = sanitized.replace(/<script[\s\S]*?<\/script>/gi, '');

  // Trim whitespace
  sanitized = sanitized.trim();

  // Normalize whitespace
  sanitized = sanitized.replace(/\s+/g, ' ');

  return sanitized;
}

/**
 * Checks if a string contains XSS patterns
 */
export function containsXSS(input: string): boolean {
  const xssPatterns = [
    /<script/i,
    /javascript:/i,
    /onerror=/i,
    /onload=/i,
    /onclick=/i,
    /<iframe/i,
    /<embed/i,
    /<object/i,
    /eval\(/i,
    /expression\(/i,
  ];

  return xssPatterns.some((pattern) => pattern.test(input));
}

/**
 * Checks if a string contains SQL injection patterns
 */
export function containsSQLInjection(input: string): boolean {
  const sqlPatterns = [
    /' or '1'='1/i,
    /' or 1=1/i,
    /'; drop table/i,
    /'; delete from/i,
    /union select/i,
    /exec\(/i,
    /execute\(/i,
  ];

  return sqlPatterns.some((pattern) => pattern.test(input));
}

/**
 * Validates that a theme is valid
 */
export function isValidTheme(theme: string): boolean {
  return validThemes.includes(theme as any);
}

/**
 * Validates that an event type is valid
 */
export function isValidEventType(eventType: string): boolean {
  return validEventTypes.includes(eventType as any);
}

/**
 * Safely parses JSON with error handling
 */
export function safeJSONParse<T>(json: string): T | null {
  try {
    return JSON.parse(json) as T;
  } catch {
    return null;
  }
}

/**
 * Validates form data against a schema
 * Returns { success: true, data } or { success: false, errors }
 */
export function validateFormData<T>(
  schema: z.ZodSchema<T>,
  data: unknown
): { success: true; data: T } | { success: false; errors: z.ZodError } {
  const result = schema.safeParse(data);

  if (result.success) {
    return { success: true, data: result.data };
  } else {
    return { success: false, errors: result.error };
  }
}

/**
 * Formats Zod errors for display
 */
export function formatZodErrors(error: z.ZodError): Record<string, string> {
  const formatted: Record<string, string> = {};

  error.errors.forEach((err) => {
    const path = err.path.join('.');
    formatted[path] = err.message;
  });

  return formatted;
}

/**
 * Creates a human-readable error message from Zod error
 */
export function getErrorMessage(error: z.ZodError): string {
  return error.errors.map((err) => err.message).join(', ');
}

// ============================================================================
// Constants
// ============================================================================

export const VALIDATION_CONSTANTS = {
  MAX_STRING_LENGTH: 1000,
  MAX_COMPANY_NAME_LENGTH: 100,
  MAX_EMAIL_LENGTH: 100,
  MAX_MESSAGE_LENGTH: 1000,
  MIN_NAME_LENGTH: 2,
  MIN_MESSAGE_LENGTH: 10,
  MAX_UPLOAD_SIZE_MB: 10,
  ALLOWED_FILE_TYPES: ['.pdf', '.doc', '.docx'],
} as const;

// ============================================================================
// Custom Validation Rules
// ============================================================================

/**
 * Creates a custom Zod validator that checks for dangerous patterns
 */
export function createSafeStringValidator(
  minLength = 1,
  maxLength = 1000,
  fieldName = 'field'
) {
  return z
    .string()
    .min(minLength, `${fieldName} must be at least ${minLength} characters`)
    .max(maxLength, `${fieldName} cannot exceed ${maxLength} characters`)
    .refine((val) => !containsXSS(val), `${fieldName} contains invalid characters`)
    .refine((val) => !containsSQLInjection(val), `${fieldName} contains invalid patterns`);
}

/**
 * Validator for safe text content
 */
export const safeTextValidator = createSafeStringValidator(1, 1000, 'Text');

/**
 * Validator for safe names
 */
export const safeNameValidator = z
  .string()
  .min(2, 'Name must be at least 2 characters')
  .max(50, 'Name cannot exceed 50 characters')
  .regex(/^[a-zA-ZÀ-ÿ\s\-']+$/, 'Name can only contain letters, spaces, hyphens, and apostrophes');

/**
 * Validator for safe company names
 */
export const safeCompanyNameValidator = z
  .string()
  .min(2, 'Company name must be at least 2 characters')
  .max(100, 'Company name cannot exceed 100 characters')
  .regex(
    /^[a-zA-ZÀ-ÿ0-9\s\-'\.&,()]+$/,
    'Company name contains invalid characters'
  )
  .refine((val) => !containsXSS(val), 'Invalid characters detected');
