import { cn, formatDate, sleep } from '../utils';

describe('Utils', () => {
  describe('cn (className merger)', () => {
    it('should merge single class', () => {
      const result = cn('text-red-500');
      expect(result).toBe('text-red-500');
    });

    it('should merge multiple classes', () => {
      const result = cn('text-red-500', 'bg-blue-500');
      expect(result).toContain('text-red-500');
      expect(result).toContain('bg-blue-500');
    });

    it('should handle conditional classes', () => {
      const isActive = true;
      const result = cn('base-class', isActive && 'active-class');
      expect(result).toContain('base-class');
      expect(result).toContain('active-class');
    });

    it('should ignore false conditional classes', () => {
      const isActive = false;
      const result = cn('base-class', isActive && 'active-class');
      expect(result).toBe('base-class');
      expect(result).not.toContain('active-class');
    });

    it('should merge Tailwind conflicting classes', () => {
      // twMerge should keep only the last class when they conflict
      const result = cn('p-4', 'p-8');
      // p-8 should override p-4
      expect(result).toBe('p-8');
    });

    it('should handle array of classes', () => {
      const result = cn(['text-red-500', 'bg-blue-500']);
      expect(result).toContain('text-red-500');
      expect(result).toContain('bg-blue-500');
    });

    it('should handle object with boolean values', () => {
      const result = cn({
        'text-red-500': true,
        'bg-blue-500': false,
        'font-bold': true
      });
      expect(result).toContain('text-red-500');
      expect(result).not.toContain('bg-blue-500');
      expect(result).toContain('font-bold');
    });

    it('should handle undefined and null', () => {
      const result = cn('text-red-500', undefined, null, 'bg-blue-500');
      expect(result).toContain('text-red-500');
      expect(result).toContain('bg-blue-500');
    });

    it('should handle empty string', () => {
      const result = cn('', 'text-red-500');
      expect(result).toBe('text-red-500');
    });

    it('should handle complex tailwind classes', () => {
      const result = cn(
        'flex items-center justify-between',
        'p-4 md:p-6',
        'bg-white dark:bg-gray-900',
        'hover:bg-gray-100'
      );
      expect(result).toContain('flex');
      expect(result).toContain('items-center');
      expect(result).toContain('justify-between');
    });
  });

  describe('formatDate', () => {
    it('should format Date object', () => {
      const date = new Date('2024-01-15T10:30:00.000Z');
      const result = formatDate(date);

      // Result should be in French format
      expect(result).toContain('2024');
      expect(result).toContain('janvier'); // French month
      expect(result).toContain('15');
    });

    it('should format ISO string', () => {
      const dateString = '2024-03-20T15:45:00.000Z';
      const result = formatDate(dateString);

      expect(result).toContain('2024');
      expect(result).toContain('mars'); // March in French
      expect(result).toContain('20');
    });

    it('should handle different months', () => {
      const dates = [
        { date: '2024-01-01', month: 'janvier' },
        { date: '2024-02-01', month: 'février' },
        { date: '2024-03-01', month: 'mars' },
        { date: '2024-04-01', month: 'avril' },
        { date: '2024-05-01', month: 'mai' },
        { date: '2024-06-01', month: 'juin' },
        { date: '2024-07-01', month: 'juillet' },
        { date: '2024-08-01', month: 'août' },
        { date: '2024-09-01', month: 'septembre' },
        { date: '2024-10-01', month: 'octobre' },
        { date: '2024-11-01', month: 'novembre' },
        { date: '2024-12-01', month: 'décembre' }
      ];

      dates.forEach(({ date, month }) => {
        const result = formatDate(date);
        expect(result.toLowerCase()).toContain(month);
      });
    });

    it('should handle year 2000', () => {
      const date = '2000-06-15';
      const result = formatDate(date);
      expect(result).toContain('2000');
    });

    it('should handle current year', () => {
      const now = new Date();
      const result = formatDate(now);
      expect(result).toContain(now.getFullYear().toString());
    });

    it('should handle leap year date', () => {
      const leapDate = '2024-02-29'; // 2024 is a leap year
      const result = formatDate(leapDate);
      expect(result).toContain('29');
      expect(result).toContain('février');
    });

    it('should be consistent for same date', () => {
      const date = '2024-05-10';
      const result1 = formatDate(date);
      const result2 = formatDate(date);
      expect(result1).toBe(result2);
    });

    it('should handle timestamp number', () => {
      const timestamp = new Date('2024-06-15T12:00:00.000Z').getTime();
      const result = formatDate(new Date(timestamp));
      expect(result).toContain('2024');
    });
  });

  describe('sleep', () => {
    it('should delay for specified milliseconds', async () => {
      const start = Date.now();
      await sleep(100);
      const end = Date.now();

      const elapsed = end - start;
      // Allow some tolerance (±20ms) due to timer precision
      expect(elapsed).toBeGreaterThanOrEqual(90);
      expect(elapsed).toBeLessThan(150);
    });

    it('should work with 0 milliseconds', async () => {
      const start = Date.now();
      await sleep(0);
      const end = Date.now();

      const elapsed = end - start;
      expect(elapsed).toBeLessThan(50); // Should be nearly instant
    });

    it('should work with very small delays', async () => {
      const start = Date.now();
      await sleep(10);
      const end = Date.now();

      const elapsed = end - start;
      expect(elapsed).toBeGreaterThanOrEqual(0);
    });

    it('should return a Promise', () => {
      const result = sleep(10);
      expect(result).toBeInstanceOf(Promise);
    });

    it('should resolve to undefined', async () => {
      const result = await sleep(10);
      expect(result).toBeUndefined();
    });

    it('should work with longer delays', async () => {
      const start = Date.now();
      await sleep(200);
      const end = Date.now();

      const elapsed = end - start;
      expect(elapsed).toBeGreaterThanOrEqual(190);
      expect(elapsed).toBeLessThan(250);
    });

    it('should be cancellable with Promise.race', async () => {
      const timeout = new Promise((resolve) => setTimeout(resolve, 50));
      const longSleep = sleep(1000);

      const start = Date.now();
      await Promise.race([timeout, longSleep]);
      const end = Date.now();

      const elapsed = end - start;
      // Should complete in ~50ms, not 1000ms
      expect(elapsed).toBeLessThan(100);
    });

    it('should work in parallel', async () => {
      const start = Date.now();

      await Promise.all([
        sleep(100),
        sleep(100),
        sleep(100)
      ]);

      const end = Date.now();
      const elapsed = end - start;

      // Should take ~100ms total (parallel), not 300ms (sequential)
      expect(elapsed).toBeLessThan(200);
    });

    it('should work sequentially', async () => {
      const start = Date.now();

      await sleep(50);
      await sleep(50);
      await sleep(50);

      const end = Date.now();
      const elapsed = end - start;

      // Should take ~150ms total (sequential)
      expect(elapsed).toBeGreaterThanOrEqual(140);
      expect(elapsed).toBeLessThan(200);
    });
  });

  describe('Edge Cases', () => {
    it('cn should handle very long class strings', () => {
      const longClass = 'class-' + 'a'.repeat(1000);
      const result = cn(longClass, 'text-red-500');
      expect(result).toContain('text-red-500');
    });

    it('formatDate should handle invalid date string gracefully', () => {
      // Invalid date should still create a Date object (might be "Invalid Date")
      const result = formatDate('invalid-date');
      // The function will call toLocaleDateString which might return "Invalid Date" or throw
      // We just verify it doesn't crash
      expect(typeof result).toBe('string');
    });

    it('formatDate should handle very old dates', () => {
      const oldDate = '1900-01-01';
      const result = formatDate(oldDate);
      expect(result).toContain('1900');
    });

    it('formatDate should handle far future dates', () => {
      const futureDate = '2099-12-31';
      const result = formatDate(futureDate);
      expect(result).toContain('2099');
    });

    it('sleep should handle negative values as 0', async () => {
      const start = Date.now();
      // setTimeout with negative value is treated as 0
      await sleep(-100 as any);
      const end = Date.now();

      const elapsed = end - start;
      expect(elapsed).toBeLessThan(50);
    });

    it('cn should handle mixed types', () => {
      const result = cn(
        'base',
        { 'conditional-1': true, 'conditional-2': false },
        ['array-1', 'array-2'],
        undefined,
        null,
        false,
        'final'
      );

      expect(result).toContain('base');
      expect(result).toContain('conditional-1');
      expect(result).not.toContain('conditional-2');
      expect(result).toContain('array-1');
      expect(result).toContain('array-2');
      expect(result).toContain('final');
    });
  });

  describe('Performance', () => {
    it('cn should be fast with many classes', () => {
      const start = Date.now();

      for (let i = 0; i < 1000; i++) {
        cn('class1', 'class2', 'class3', 'class4', 'class5');
      }

      const end = Date.now();
      const elapsed = end - start;

      // Should complete 1000 iterations in less than 100ms
      expect(elapsed).toBeLessThan(100);
    });

    it('formatDate should be fast', () => {
      const start = Date.now();
      const date = new Date('2024-01-01');

      for (let i = 0; i < 1000; i++) {
        formatDate(date);
      }

      const end = Date.now();
      const elapsed = end - start;

      // Should complete 1000 iterations reasonably fast
      expect(elapsed).toBeLessThan(500);
    });
  });

  describe('Type Safety', () => {
    it('cn should accept various input types', () => {
      // This test verifies that TypeScript types work correctly
      const stringClass = cn('string');
      const arrayClass = cn(['array1', 'array2']);
      const objectClass = cn({ active: true });
      const mixedClass = cn('string', ['array'], { active: true });

      expect(typeof stringClass).toBe('string');
      expect(typeof arrayClass).toBe('string');
      expect(typeof objectClass).toBe('string');
      expect(typeof mixedClass).toBe('string');
    });

    it('formatDate should accept Date object', () => {
      const result = formatDate(new Date());
      expect(typeof result).toBe('string');
    });

    it('formatDate should accept string', () => {
      const result = formatDate('2024-01-01');
      expect(typeof result).toBe('string');
    });

    it('sleep should return Promise<void>', async () => {
      const promise = sleep(10);
      expect(promise).toBeInstanceOf(Promise);
      const result = await promise;
      expect(result).toBeUndefined();
    });
  });
});
