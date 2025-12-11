import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import ExportPDFButton from '../ExportPDFButton';

// Mock environment variable
const mockApiUrl = 'http://localhost:8080';
process.env.NEXT_PUBLIC_API_URL = mockApiUrl;

// Mock lucide-react icons
jest.mock('lucide-react', () => ({
  Download: () => <svg data-testid="download-icon" />,
  Loader2: ({ className }: any) => <svg data-testid="loader-icon" className={className} />,
}));

// Mock UI components
jest.mock('@/components/ui/button', () => ({
  Button: ({ children, onClick, disabled, className, ...props }: any) => (
    <button onClick={onClick} disabled={disabled} className={className} {...props}>
      {children}
    </button>
  ),
}));

describe('ExportPDFButton', () => {
  let mockAnchor: any;
  let mockFetch: jest.Mock;
  let mockCreateObjectURL: jest.Mock;
  let mockRevokeObjectURL: jest.Mock;

  beforeEach(() => {
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
    const originalCreateElement = document.createElement.bind(document);
    jest.spyOn(document, 'createElement').mockImplementation((tagName: string) => {
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
      return originalCreateElement(tagName);
    });

    // Mock appendChild/removeChild
    jest.spyOn(document.body, 'appendChild').mockImplementation((node: any) => node);
    jest.spyOn(document.body, 'removeChild').mockImplementation((node: any) => node);
  });

  afterEach(() => {
    // Only clear mock calls, don't restore to avoid breaking component rendering
    jest.clearAllMocks();
  });

  it('should render button with correct text', async () => {
    const { container } = render(<ExportPDFButton theme="technical" />);

    // Wait for component to render
    await waitFor(() => {
      const button = container.querySelector('button');
      expect(button).toBeInTheDocument();
      expect(button).toHaveTextContent('Télécharger PDF');
    });
  });

  it('should render Download icon when not loading', async () => {
    const { container } = render(<ExportPDFButton theme="technical" />);

    // Wait for component to render
    await waitFor(() => {
      const downloadIcon = container.querySelector('[data-testid="download-icon"]');
      expect(downloadIcon).toBeInTheDocument();
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
        `${mockApiUrl}/api/cv/export?theme=technical&format=pdf`
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
      expect(container.textContent).toContain('Export en cours...');
    });

    // Button should be disabled during loading
    expect(button).toBeDisabled();
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

    // Override appendChild to capture the anchor element
    const appendChildSpy = jest.spyOn(document.body, 'appendChild').mockImplementation((node: any) => {
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
  });

  it('should use fallback filename if Content-Disposition missing', async () => {
    const mockBlob = new Blob(['PDF content'], { type: 'application/pdf' });
    let capturedAnchor: any;

    mockFetch.mockResolvedValueOnce({
      ok: true,
      headers: {
        get: () => null,
      },
      blob: async () => mockBlob,
    });

    // Override appendChild to capture the anchor element
    const appendChildSpy = jest.spyOn(document.body, 'appendChild').mockImplementation((node: any) => {
      capturedAnchor = node;
      return node;
    });

    const { container } = render(<ExportPDFButton theme="business" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    await waitFor(() => {
      expect(appendChildSpy).toHaveBeenCalled();
      expect(capturedAnchor.download).toBe('CV_business.pdf');
    });
  });

  it('should handle API error gracefully', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
    });

    const { container } = render(<ExportPDFButton theme="technical" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    await waitFor(() => {
      expect(container.textContent).toContain('Échec de l\'export PDF');
    });

    // Error message should be displayed - check for the paragraph with error text
    const errorElements = container.querySelectorAll('p');
    const errorElement = Array.from(errorElements).find(el =>
      el.textContent?.includes('Échec de l\'export PDF')
    );
    expect(errorElement).toBeInTheDocument();
    expect(errorElement).toHaveClass('text-red-600');
  });

  it('should handle network error', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    const { container } = render(<ExportPDFButton theme="technical" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    await waitFor(() => {
      expect(container.textContent).toContain('Network error');
    });
  });

  it('should handle unknown error types', async () => {
    mockFetch.mockRejectedValueOnce('Unknown error');

    const { container } = render(<ExportPDFButton theme="technical" />);

    const button = container.querySelector('button');
    if (button) fireEvent.click(button);

    await waitFor(() => {
      expect(container.textContent).toContain('Une erreur est survenue');
    });
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
      expect(document.createElement).toHaveBeenCalledWith('a');
      expect(mockAnchor.click).toHaveBeenCalled();
      expect(document.body.appendChild).toHaveBeenCalled();
      expect(document.body.removeChild).toHaveBeenCalled();
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
      expect(container.textContent).toContain('Échec de l\'export PDF');
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
      expect(container.textContent).not.toContain('Échec de l\'export PDF');
    });
  });

  it('should render with gradient styling', async () => {
    const { container } = render(<ExportPDFButton theme="technical" />);

    await waitFor(() => {
      const button = container.querySelector('button');
      expect(button).toBeInTheDocument();
      expect(button).toHaveClass('bg-gradient-to-r', 'from-blue-600', 'to-purple-600');
    });
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

    await waitFor(() => {
      const spinner = container.querySelector('[data-testid="loader-icon"]');
      expect(spinner).toBeInTheDocument();
      expect(spinner).toHaveClass('animate-spin');
    });
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

    // Re-enabled after completion
    await waitFor(() => {
      expect(button).not.toBeDisabled();
    });
  });
});
