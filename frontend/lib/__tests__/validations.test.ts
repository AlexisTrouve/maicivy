import {
  letterGenerationSchema,
  cvThemeSchema,
  analyticsEventSchema,
  contactFormSchema,
  searchSchema,
  filterSchema,
  urlSchema,
  isValidEmail,
  isValidURL,
  sanitizeString,
  containsXSS,
  containsSQLInjection,
  isValidTheme,
  isValidEventType,
  safeJSONParse,
  validateFormData,
  formatZodErrors,
  getErrorMessage,
  createSafeStringValidator,
  safeTextValidator,
  safeNameValidator,
  safeCompanyNameValidator,
} from '../validations';
import { z } from 'zod';

describe('Validation Schemas', () => {
  describe('letterGenerationSchema', () => {
    it('should validate correct company name', () => {
      const result = letterGenerationSchema.safeParse({
        companyName: 'Google Inc.'
      });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.companyName).toBe('Google Inc.');
      }
    });

    it('should reject company name that is too short', () => {
      const result = letterGenerationSchema.safeParse({
        companyName: 'A'
      });

      expect(result.success).toBe(false);
    });

    it('should reject company name that is too long', () => {
      const result = letterGenerationSchema.safeParse({
        companyName: 'A'.repeat(101)
      });

      expect(result.success).toBe(false);
    });

    it('should reject company name with invalid characters', () => {
      const result = letterGenerationSchema.safeParse({
        companyName: 'Company <script>alert("xss")</script>'
      });

      expect(result.success).toBe(false);
    });

    it('should allow company names with special chars (&, -, .)', () => {
      const result = letterGenerationSchema.safeParse({
        companyName: 'Johnson & Johnson Inc.'
      });

      expect(result.success).toBe(true);
    });

    it('should allow company names with accents', () => {
      const result = letterGenerationSchema.safeParse({
        companyName: 'Société Générale'
      });

      expect(result.success).toBe(true);
    });
  });

  describe('cvThemeSchema', () => {
    it('should validate valid themes', () => {
      const validThemes = ['backend', 'frontend', 'fullstack', 'cpp', 'devops', 'artistic', 'data-science', 'mobile'];

      validThemes.forEach(theme => {
        const result = cvThemeSchema.safeParse({ theme });
        expect(result.success).toBe(true);
      });
    });

    it('should reject invalid theme', () => {
      const result = cvThemeSchema.safeParse({
        theme: 'invalid-theme'
      });

      expect(result.success).toBe(false);
    });

    it('should provide custom error message', () => {
      const result = cvThemeSchema.safeParse({
        theme: 'wrong'
      });

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.error.errors[0].message).toBe('Thème invalide');
      }
    });
  });

  describe('analyticsEventSchema', () => {
    it('should validate correct event', () => {
      const result = analyticsEventSchema.safeParse({
        eventType: 'page_view',
        eventData: { page: '/cv' },
        timestamp: Date.now()
      });

      expect(result.success).toBe(true);
    });

    it('should allow optional fields', () => {
      const result = analyticsEventSchema.safeParse({
        eventType: 'click'
      });

      expect(result.success).toBe(true);
    });

    it('should reject invalid event type', () => {
      const result = analyticsEventSchema.safeParse({
        eventType: 'invalid_event'
      });

      expect(result.success).toBe(false);
    });

    it('should validate all event types', () => {
      const validEvents = ['page_view', 'click', 'cv_theme_change', 'letter_generated', 'pdf_downloaded', 'analytics_viewed'];

      validEvents.forEach(eventType => {
        const result = analyticsEventSchema.safeParse({ eventType });
        expect(result.success).toBe(true);
      });
    });
  });

  describe('contactFormSchema', () => {
    it('should validate correct contact form', () => {
      const result = contactFormSchema.safeParse({
        name: 'John Doe',
        email: 'john@example.com',
        subject: 'Question about CV',
        message: 'I would like to know more about your experience.'
      });

      expect(result.success).toBe(true);
    });

    it('should reject invalid email', () => {
      const result = contactFormSchema.safeParse({
        name: 'John Doe',
        email: 'invalid-email',
        subject: 'Test',
        message: 'Test message here.'
      });

      expect(result.success).toBe(false);
    });

    it('should reject name with invalid characters', () => {
      const result = contactFormSchema.safeParse({
        name: 'John123',
        email: 'john@example.com',
        subject: 'Test',
        message: 'Test message'
      });

      expect(result.success).toBe(false);
    });

    it('should reject message that is too short', () => {
      const result = contactFormSchema.safeParse({
        name: 'John Doe',
        email: 'john@example.com',
        subject: 'Test',
        message: 'Short'
      });

      expect(result.success).toBe(false);
    });

    it('should allow names with hyphens and apostrophes', () => {
      const result = contactFormSchema.safeParse({
        name: "Jean-Pierre O'Connor",
        email: 'jp@example.com',
        subject: 'Question',
        message: 'This is a longer message for testing.'
      });

      expect(result.success).toBe(true);
    });
  });

  describe('searchSchema', () => {
    it('should validate correct search query', () => {
      const result = searchSchema.safeParse({
        query: 'JavaScript developer'
      });

      expect(result.success).toBe(true);
    });

    it('should reject empty query', () => {
      const result = searchSchema.safeParse({
        query: ''
      });

      expect(result.success).toBe(false);
    });

    it('should reject query with XSS attempt', () => {
      const result = searchSchema.safeParse({
        query: '<script>alert("xss")</script>'
      });

      expect(result.success).toBe(false);
    });
  });

  describe('filterSchema', () => {
    it('should validate correct filter', () => {
      const result = filterSchema.safeParse({
        dateFrom: '2024-01-01T00:00:00.000Z',
        dateTo: '2024-12-31T23:59:59.999Z',
        theme: 'backend',
        limit: 20,
        offset: 0
      });

      expect(result.success).toBe(true);
    });

    it('should use default values', () => {
      const result = filterSchema.safeParse({});

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.limit).toBe(10);
        expect(result.data.offset).toBe(0);
      }
    });

    it('should reject limit > 100', () => {
      const result = filterSchema.safeParse({
        limit: 101
      });

      expect(result.success).toBe(false);
    });

    it('should reject negative offset', () => {
      const result = filterSchema.safeParse({
        offset: -1
      });

      expect(result.success).toBe(false);
    });
  });

  describe('urlSchema', () => {
    it('should validate correct HTTPS URL', () => {
      const result = urlSchema.safeParse('https://example.com');
      expect(result.success).toBe(true);
    });

    it('should validate correct HTTP URL', () => {
      const result = urlSchema.safeParse('http://example.com');
      expect(result.success).toBe(true);
    });

    it('should reject localhost URLs', () => {
      const result = urlSchema.safeParse('http://localhost:3000');
      expect(result.success).toBe(false);
    });

    it('should reject 127.0.0.1 URLs', () => {
      const result = urlSchema.safeParse('http://127.0.0.1:8080');
      expect(result.success).toBe(false);
    });

    it('should reject invalid URL format', () => {
      const result = urlSchema.safeParse('not a url');
      expect(result.success).toBe(false);
    });
  });
});

describe('Helper Functions', () => {
  describe('isValidEmail', () => {
    it('should return true for valid email', () => {
      expect(isValidEmail('test@example.com')).toBe(true);
      expect(isValidEmail('user.name+tag@example.co.uk')).toBe(true);
    });

    it('should return false for invalid email', () => {
      expect(isValidEmail('invalid')).toBe(false);
      expect(isValidEmail('test@')).toBe(false);
      expect(isValidEmail('@example.com')).toBe(false);
    });
  });

  describe('isValidURL', () => {
    it('should return true for valid URLs', () => {
      expect(isValidURL('https://example.com')).toBe(true);
      expect(isValidURL('http://test.org/path')).toBe(true);
    });

    it('should return false for invalid URLs', () => {
      expect(isValidURL('http://localhost')).toBe(false);
      expect(isValidURL('not a url')).toBe(false);
      expect(isValidURL('ftp://example.com')).toBe(false);
    });
  });

  describe('sanitizeString', () => {
    it('should remove HTML tags', () => {
      const input = '<p>Hello <b>World</b></p>';
      const output = sanitizeString(input);
      expect(output).toBe('Hello World');
    });

    it('should remove script tags and content', () => {
      const input = 'Hello <script>alert("xss")</script> World';
      const output = sanitizeString(input);
      expect(output).toBe('Hello World');
    });

    it('should trim whitespace', () => {
      const input = '  Hello World  ';
      const output = sanitizeString(input);
      expect(output).toBe('Hello World');
    });

    it('should normalize whitespace', () => {
      const input = 'Hello    World';
      const output = sanitizeString(input);
      expect(output).toBe('Hello World');
    });
  });

  describe('containsXSS', () => {
    it('should detect script tags', () => {
      expect(containsXSS('<script>alert("xss")</script>')).toBe(true);
      expect(containsXSS('<SCRIPT>alert("xss")</SCRIPT>')).toBe(true);
    });

    it('should detect javascript: protocol', () => {
      expect(containsXSS('javascript:alert("xss")')).toBe(true);
    });

    it('should detect event handlers', () => {
      expect(containsXSS('<img onerror="alert()">')).toBe(true);
      expect(containsXSS('<body onload="alert()">')).toBe(true);
      expect(containsXSS('<div onclick="evil()">')).toBe(true);
    });

    it('should detect dangerous tags', () => {
      expect(containsXSS('<iframe src="evil.com"></iframe>')).toBe(true);
      expect(containsXSS('<embed src="evil.swf">')).toBe(true);
      expect(containsXSS('<object data="evil">')).toBe(true);
    });

    it('should detect eval and expression', () => {
      expect(containsXSS('eval(malicious)')).toBe(true);
      expect(containsXSS('expression(malicious)')).toBe(true);
    });

    it('should return false for safe strings', () => {
      expect(containsXSS('Hello World')).toBe(false);
      expect(containsXSS('This is a normal string')).toBe(false);
    });
  });

  describe('containsSQLInjection', () => {
    it('should detect common SQL injection patterns', () => {
      expect(containsSQLInjection("' or '1'='1")).toBe(true);
      expect(containsSQLInjection("' or 1=1--")).toBe(true);
    });

    it('should detect DROP TABLE attempts', () => {
      expect(containsSQLInjection("'; drop table users;")).toBe(true);
      expect(containsSQLInjection("'; DROP TABLE users;")).toBe(true);
    });

    it('should detect DELETE FROM attempts', () => {
      expect(containsSQLInjection("'; delete from users;")).toBe(true);
    });

    it('should detect UNION SELECT', () => {
      expect(containsSQLInjection('union select * from users')).toBe(true);
      expect(containsSQLInjection('UNION SELECT password FROM users')).toBe(true);
    });

    it('should detect exec/execute', () => {
      expect(containsSQLInjection('exec(malicious)')).toBe(true);
      expect(containsSQLInjection('execute sp_help')).toBe(true);
    });

    it('should return false for safe strings', () => {
      expect(containsSQLInjection('Normal text')).toBe(false);
      expect(containsSQLInjection('Company name')).toBe(false);
    });
  });

  describe('isValidTheme', () => {
    it('should return true for valid themes', () => {
      expect(isValidTheme('backend')).toBe(true);
      expect(isValidTheme('frontend')).toBe(true);
      expect(isValidTheme('fullstack')).toBe(true);
    });

    it('should return false for invalid themes', () => {
      expect(isValidTheme('invalid')).toBe(false);
      expect(isValidTheme('')).toBe(false);
    });
  });

  describe('isValidEventType', () => {
    it('should return true for valid event types', () => {
      expect(isValidEventType('page_view')).toBe(true);
      expect(isValidEventType('click')).toBe(true);
      expect(isValidEventType('letter_generated')).toBe(true);
    });

    it('should return false for invalid event types', () => {
      expect(isValidEventType('invalid_event')).toBe(false);
      expect(isValidEventType('')).toBe(false);
    });
  });

  describe('safeJSONParse', () => {
    it('should parse valid JSON', () => {
      const json = '{"name": "John", "age": 30}';
      const result = safeJSONParse<any>(json);

      expect(result).toEqual({ name: 'John', age: 30 });
    });

    it('should return null for invalid JSON', () => {
      const json = '{invalid json}';
      const result = safeJSONParse(json);

      expect(result).toBeNull();
    });

    it('should handle empty string', () => {
      const result = safeJSONParse('');
      expect(result).toBeNull();
    });
  });

  describe('validateFormData', () => {
    const schema = z.object({
      name: z.string().min(2),
      age: z.number().min(0)
    });

    it('should return success for valid data', () => {
      const result = validateFormData(schema, {
        name: 'John',
        age: 30
      });

      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.name).toBe('John');
      }
    });

    it('should return errors for invalid data', () => {
      const result = validateFormData(schema, {
        name: 'J',
        age: -1
      });

      expect(result.success).toBe(false);
      if (!result.success) {
        expect(result.errors).toBeDefined();
      }
    });
  });

  describe('formatZodErrors', () => {
    it('should format Zod errors to object', () => {
      const schema = z.object({
        name: z.string().min(2, 'Name too short'),
        email: z.string().email('Invalid email')
      });

      const result = schema.safeParse({
        name: 'A',
        email: 'invalid'
      });

      if (!result.success) {
        const formatted = formatZodErrors(result.error);

        expect(formatted.name).toBe('Name too short');
        expect(formatted.email).toBe('Invalid email');
      }
    });
  });

  describe('getErrorMessage', () => {
    it('should create readable error message', () => {
      const schema = z.object({
        name: z.string().min(2, 'Name too short'),
        email: z.string().email('Invalid email')
      });

      const result = schema.safeParse({
        name: 'A',
        email: 'invalid'
      });

      if (!result.success) {
        const message = getErrorMessage(result.error);
        expect(message).toContain('Name too short');
        expect(message).toContain('Invalid email');
      }
    });
  });
});

describe('Custom Validators', () => {
  describe('createSafeStringValidator', () => {
    it('should create validator with custom lengths', () => {
      const validator = createSafeStringValidator(5, 20, 'username');

      const valid = validator.safeParse('validuser');
      expect(valid.success).toBe(true);

      const tooShort = validator.safeParse('usr');
      expect(tooShort.success).toBe(false);

      const tooLong = validator.safeParse('a'.repeat(21));
      expect(tooLong.success).toBe(false);
    });

    it('should reject XSS patterns', () => {
      const validator = createSafeStringValidator(1, 100, 'input');

      const result = validator.safeParse('<script>alert("xss")</script>');
      expect(result.success).toBe(false);
    });

    it('should reject SQL injection patterns', () => {
      const validator = createSafeStringValidator(1, 100, 'input');

      const result = validator.safeParse("' or '1'='1");
      expect(result.success).toBe(false);
    });
  });

  describe('safeTextValidator', () => {
    it('should validate safe text', () => {
      const result = safeTextValidator.safeParse('This is safe text');
      expect(result.success).toBe(true);
    });

    it('should reject dangerous content', () => {
      const result = safeTextValidator.safeParse('<script>evil</script>');
      expect(result.success).toBe(false);
    });
  });

  describe('safeNameValidator', () => {
    it('should validate correct names', () => {
      expect(safeNameValidator.safeParse('John Doe').success).toBe(true);
      expect(safeNameValidator.safeParse("Jean-Pierre O'Connor").success).toBe(true);
      expect(safeNameValidator.safeParse('José García').success).toBe(true);
    });

    it('should reject names with numbers', () => {
      const result = safeNameValidator.safeParse('John123');
      expect(result.success).toBe(false);
    });

    it('should reject names that are too short', () => {
      const result = safeNameValidator.safeParse('A');
      expect(result.success).toBe(false);
    });
  });

  describe('safeCompanyNameValidator', () => {
    it('should validate correct company names', () => {
      expect(safeCompanyNameValidator.safeParse('Google Inc.').success).toBe(true);
      expect(safeCompanyNameValidator.safeParse('Johnson & Johnson').success).toBe(true);
      expect(safeCompanyNameValidator.safeParse('AT&T (USA)').success).toBe(true);
    });

    it('should reject company names with XSS', () => {
      const result = safeCompanyNameValidator.safeParse('Evil Corp <script>alert()</script>');
      expect(result.success).toBe(false);
    });

    it('should reject company names that are too short', () => {
      const result = safeCompanyNameValidator.safeParse('A');
      expect(result.success).toBe(false);
    });
  });
});

describe('Edge Cases', () => {
  it('should handle null and undefined gracefully', () => {
    expect(safeJSONParse(null as any)).toBeNull();
    expect(sanitizeString('')).toBe('');
  });

  it('should handle unicode characters', () => {
    const unicode = '你好世界 🚀';
    expect(containsXSS(unicode)).toBe(false);
    expect(sanitizeString(unicode)).toBe(unicode);
  });

  it('should handle very long strings', () => {
    const longString = 'a'.repeat(10000);
    expect(() => containsXSS(longString)).not.toThrow();
    expect(() => sanitizeString(longString)).not.toThrow();
  });

  it('should handle special regex characters in input', () => {
    const specialChars = '.*+?^${}()|[]\\';
    expect(() => containsXSS(specialChars)).not.toThrow();
  });
});
