import React from 'react';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import ExportPDFButton, { pdfLang } from '../ExportPDFButton';

// Mock environment variable
const mockApiUrl = 'http://localhost:8080';
process.env.NEXT_PUBLIC_API_URL = mockApiUrl;

// Note: lucide-react is already mocked globally in __mocks__/lucide-react.tsx
// No need to mock it here - the global mock provides download-icon and loader2-icon testids

// Note: Button component is NOT mocked - we let it render normally
// The actual Button component from shadcn/ui will be used

describe('ExportPDFButton', () => {
  let mockAnchor: any;
  let mockFetch: jest.Mock;
  let mockCreateObjectURL: jest.Mock;
  let mockRevokeObjectURL: jest.Mock;
  let createElementSpy: jest.SpyInstance;
  let appendChildSpy: jest.SpyInstance;
  let removeChildSpy: jest.SpyInstance;

  // Store the REAL createElement function outside beforeEach so tests can access it
  const realCreateElement = document.createElement.bind(document);

  beforeEach(() => {
    // Use fake timers to control setTimeout
    jest.useFakeTimers();

    // Restore all mocks FIRST to ensure clean state
    jest.restoreAllMocks();

    // Mock fetch
    mockFetch = jest.fn();
    global.fetch = mockFetch;

    // Mock URL methods
    mockCreateObjectURL = jest.fn(() => 'blob:mock-url');
    mockRevokeObjectURL = jest.fn();
    global.URL.createObjectURL = mockCreateObjectURL;
    global.URL.revokeObjectURL = mockRevokeObjectURL;

    // Mock anchor element - create fresh instance for each test
    mockAnchor = {
      click: jest.fn(),
      setAttribute: jest.fn(),
      href: '',
      download: '',
      style: {},
    };

    // Mock document.createElement
    createElementSpy = jest.spyOn(document, 'createElement').mockImplementation((tagName: string) => {
      if (tagName === 'a') {
        // Return a fresh anchor each time
        return {
          click: jest.fn(),
          setAttribute: jest.fn(),
          href: '',
          download: '',
          style: {},
        } as any;
      }
      return realCreateElement(tagName);
    });

    // Mock appendChild/removeChild
    appendChildSpy = jest.spyOn(document.body, 'appendChild').mockImplementation((node: any) => node);
    removeChildSpy = jest.spyOn(document.body, 'removeChild').mockImplementation((node: any) => node);
  });

  afterEach(() => {
    // Run all pending timers before cleanup
    jest.runOnlyPendingTimers();
    // Clear all timers and restore real timers
    jest.clearAllTimers();
    jest.useRealTimers();
  });

  it('should render button with correct text', async () => {
    const { container } = render(<ExportPDFButton theme="technical" />);

    // Wait for button to render (component might render asynchronously)
    await waitFor(() => {
      const button = container.querySelector('button');
      expect(button).not.toBeNull();
    });

    const button = container.querySelector('button');
    expect(button).toHaveTextContent('Télécharger le PDF');
  });

  it('should render Download icon when not loading', async () => {
    const { container } = render(<ExportPDFButton theme="technical" />);

    // Wait for icon to render
    await waitFor(() => {
      const downloadIcon = container.querySelector('[data-testid="download-icon"]');
      expect(downloadIcon).not.toBeNull();
    });
  });

  it('should trigger PDF export on button click', async () => {
    const mockBlob = new Blob(['PDF content'], { type: 'application/pdf' });

    mockFetch.mockResolvedValueOnce({
      ok: true,
      headers: {
        get: (name: string) => {
          if (name === 'Content-Disposition') {
            return 'attachment; filename="CV_technical.pdf"';
          }
          return null;
        },
      },
      blob: async () => mockBlob,
    });

    const { container } = render(<ExportPDFButton theme="technical" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        `${mockApiUrl}/api/v1/cv/export?theme=technical&format=pdf&lang=fr`
      );
    });
  });

  it('should show loading state during export', async () => {
    mockFetch.mockImplementation(
      () =>
        new Promise((resolve) =>
          setTimeout(
            () =>
              resolve({
                ok: true,
                headers: { get: () => null },
                blob: async () => new Blob(['PDF']),
              }),
            100
          )
        )
    );

    const { container } = render(<ExportPDFButton theme="creative" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    // Should show loading text and spinner
    await waitFor(() => {
      expect(container.textContent).toContain('Préparation du PDF…');
    });

    // Button should be disabled during loading
    expect(button).toBeDisabled();

    // Advance timers to complete the setTimeout
    jest.advanceTimersByTime(100);
    await waitFor(() => {
      expect(button).not.toBeDisabled();
    });
  });

  it('should use custom filename from Content-Disposition header', async () => {
    const mockBlob = new Blob(['PDF content'], { type: 'application/pdf' });
    const customFilename = 'MonCV_2024.pdf';
    let capturedAnchor: any;

    mockFetch.mockResolvedValueOnce({
      ok: true,
      headers: {
        get: () => `attachment; filename="${customFilename}"`,
      },
      blob: async () => mockBlob,
    });

    // Store original implementation
    const originalAppendChild = appendChildSpy.getMockImplementation();

    // Override appendChild to capture the anchor element
    appendChildSpy.mockImplementation((node: any) => {
      capturedAnchor = node;
      return node;
    });

    const { container } = render(<ExportPDFButton theme="technical" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    await waitFor(() => {
      expect(appendChildSpy).toHaveBeenCalled();
      expect(capturedAnchor.download).toBe(customFilename);
    });

    // Restore original implementation
    if (originalAppendChild) {
      appendChildSpy.mockImplementation(originalAppendChild);
    }
  });

  it('should use fallback filename if Content-Disposition missing', async () => {
    const mockBlob = new Blob(['PDF content'], { type: 'application/pdf' });
    const capturedAnchors: any[] = [];

    mockFetch.mockResolvedValueOnce({
      ok: true,
      headers: {
        get: () => null,
      },
      blob: async () => mockBlob,
    });

    // Store original implementations
    const originalCreateElement = createElementSpy.getMockImplementation();

    // Clear call history
    createElementSpy.mockClear();

    // Override createElement to capture ALL anchor elements
    createElementSpy.mockImplementation((tagName: string) => {
      if (tagName === 'a') {
        // Create a new anchor and add to array
        const anchor = {
          click: jest.fn(),
          setAttribute: jest.fn(),
          href: '',
          download: '',
          style: {},
        } as any;
        capturedAnchors.push(anchor);
        return anchor;
      }
      return realCreateElement(tagName);
    });

    const { container } = render(<ExportPDFButton theme="business" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    await waitFor(() => {
      // Wait for at least one anchor to be created
      expect(capturedAnchors.length).toBeGreaterThan(0);
    });

    // Find the anchor with the correct filename
    const businessAnchor = capturedAnchors.find(a => a.download === 'CV_business.pdf');
    expect(businessAnchor).toBeDefined();

    // Restore original implementation
    if (originalCreateElement) {
      createElementSpy.mockImplementation(originalCreateElement);
    }
  });

  it('should handle API error gracefully', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
    });

    const { container } = render(<ExportPDFButton theme="technical" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    // Message traduit, annoncé (role=alert), état porté par le glyphe ✕ ET la teinte du thème.
    const alert = await within(container).findByRole('alert');
    expect(alert).toHaveTextContent("Le PDF n'a pas pu être généré");
    expect(alert).toHaveTextContent('✕');
    expect(alert).toHaveClass('text-destructive');
    consoleErrorSpy.mockRestore();
  });

  // Le texte technique d'une erreur réseau (en anglais brut) ne doit JAMAIS atteindre le visiteur :
  // il lit le message traduit, le détail part dans la console.
  it('should show the translated message on network error, not the raw error text', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    const { container } = render(<ExportPDFButton theme="technical" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    const alert = await within(container).findByRole('alert');
    expect(alert).toHaveTextContent("Le PDF n'a pas pu être généré");
    expect(container.textContent).not.toContain('Network error');
    expect(consoleErrorSpy).toHaveBeenCalled();
    consoleErrorSpy.mockRestore();
  });

  it('should handle unknown error types', async () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();
    mockFetch.mockRejectedValueOnce('Unknown error');

    const { container } = render(<ExportPDFButton theme="technical" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    const alert = await within(container).findByRole('alert');
    expect(alert).toHaveTextContent("Le PDF n'a pas pu être généré");
    consoleErrorSpy.mockRestore();
  });

  it('should create and trigger download link correctly', async () => {
    const mockBlob = new Blob(['PDF content'], { type: 'application/pdf' });

    mockFetch.mockResolvedValueOnce({
      ok: true,
      headers: { get: () => 'attachment; filename="test.pdf"' },
      blob: async () => mockBlob,
    });

    const { container } = render(<ExportPDFButton theme="technical" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    await waitFor(() => {
      expect(createElementSpy).toHaveBeenCalledWith('a');
      expect(appendChildSpy).toHaveBeenCalled();
      expect(removeChildSpy).toHaveBeenCalled();
      expect(mockCreateObjectURL).toHaveBeenCalledWith(mockBlob);
      expect(mockRevokeObjectURL).toHaveBeenCalled();
    });
  });

  it('should clear error on new export attempt', async () => {
    // First attempt - error
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
    });

    const { container } = render(<ExportPDFButton theme="technical" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    await waitFor(() => {
      expect(container.textContent).toContain("Le PDF n'a pas pu être généré");
    });

    // Second attempt - success
    mockFetch.mockResolvedValueOnce({
      ok: true,
      headers: { get: () => null },
      blob: async () => new Blob(['PDF']),
    });

    if (button) fireEvent.click(button);

    // Error should be cleared
    await waitFor(() => {
      expect(container.textContent).not.toContain("Le PDF n'a pas pu être généré");
    });
  });

  // Doctrine de l'or : l'export est L'action principale de /cv, donc le seul aplat d'or de l'écran.
  it('should render as the gold primary action', async () => {
    const { container } = render(<ExportPDFButton theme="technical" />);

    await waitFor(() => {
      expect(container.querySelector('button')).not.toBeNull();
    });

    const button = within(container).getByTestId('cv-export-pdf');
    expect(button).toHaveClass('bg-gold', 'text-primary-foreground');
    expect(button.className).not.toMatch(/blue|purple|gradient/);
  });

  it('should show Loader2 icon when loading', async () => {
    mockFetch.mockImplementation(
      () =>
        new Promise((resolve) =>
          setTimeout(
            () =>
              resolve({
                ok: true,
                headers: { get: () => null },
                blob: async () => new Blob(['PDF']),
              }),
            100
          )
        )
    );

    const { container } = render(<ExportPDFButton theme="technical" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    // Wait for the loader icon to appear
    await waitFor(() => {
      const spinner = container.querySelector('[data-testid="loader2-icon"]');
      expect(spinner).not.toBeNull();
    });

    // Now verify it has the animate-spin class
    const spinner = container.querySelector('[data-testid="loader2-icon"]');
    expect(spinner).toHaveClass('animate-spin');

    // Advance timers to complete the setTimeout
    jest.advanceTimersByTime(100);
  });

  it('should disable button only when loading', async () => {
    const { container } = render(<ExportPDFButton theme="technical" />);

    const button = container.querySelector('button');

    // Initially enabled
    expect(button).not.toBeDisabled();

    // Mock delayed response
    mockFetch.mockImplementation(
      () =>
        new Promise((resolve) =>
          setTimeout(
            () =>
              resolve({
                ok: true,
                headers: { get: () => null },
                blob: async () => new Blob(['PDF']),
              }),
            50
          )
        )
    );

    if (button) fireEvent.click(button);

    // Disabled during loading
    await waitFor(() => {
      expect(button).toBeDisabled();
    });

    // Advance timers to complete the setTimeout
    jest.advanceTimersByTime(50);

    // Re-enabled after completion
    await waitFor(() => {
      expect(button).not.toBeDisabled();
    });
  });
});

// Le PDF n'existe qu'en fr/en/zh : de/it (locales du site) partaient tels quels et prenaient un 400.
describe('pdfLang', () => {
  it('garde les langues que le PDF sait rendre', () => {
    expect(pdfLang('fr')).toBe('fr');
    expect(pdfLang('en')).toBe('en');
    expect(pdfLang('zh')).toBe('zh');
  });
  it('replie de/it sur l’anglais, jamais le français', () => {
    expect(pdfLang('de')).toBe('en');
    expect(pdfLang('it')).toBe('en');
  });
});
