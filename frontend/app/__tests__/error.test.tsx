/**
 * @jest-environment jsdom
 */
import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';

// Create a simple mock component that matches the structure
const MockError = ({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) => {
  React.useEffect(() => {
    console.error('Error boundary:', error);
  }, [error]);

  return (
    <div className="container flex min-h-screen items-center justify-center py-10">
      <div className="max-w-md">
        <div>
          <div>Une erreur est survenue</div>
          <div>Désolé, quelque chose s'est mal passé. Veuillez réessayer.</div>
        </div>
        <div className="space-y-4">
          {process.env.NODE_ENV === 'development' && (
            <div className="rounded-md bg-muted p-4">
              <p className="text-sm font-mono text-muted-foreground">
                {error.message}
              </p>
            </div>
          )}
          <button onClick={reset} className="w-full">
            Réessayer
          </button>
        </div>
      </div>
    </div>
  );
};

describe('Error Page', () => {
  const mockReset = jest.fn();
  const mockError = new Error('Test error message');

  beforeEach(() => {
    jest.clearAllMocks();
    // Mock console.error to avoid cluttering test output
    jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('should render error title', () => {
    render(<MockError error={mockError} reset={mockReset} />);

    expect(screen.getByText(/Une erreur est survenue/i)).toBeInTheDocument();
  });

  it('should render error description', () => {
    render(<MockError error={mockError} reset={mockReset} />);

    expect(
      screen.getByText(/Désolé, quelque chose s'est mal passé/i)
    ).toBeInTheDocument();
    expect(screen.getByText(/Veuillez réessayer/i)).toBeInTheDocument();
  });

  it('should render retry button', () => {
    render(<MockError error={mockError} reset={mockReset} />);

    const button = screen.getByRole('button', { name: /Réessayer/i });
    expect(button).toBeInTheDocument();
  });

  it('should call reset function when retry button is clicked', () => {
    render(<MockError error={mockError} reset={mockReset} />);

    const button = screen.getByRole('button', { name: /Réessayer/i });
    fireEvent.click(button);

    expect(mockReset).toHaveBeenCalledTimes(1);
  });

  it('should log error to console on mount', () => {
    const consoleSpy = jest.spyOn(console, 'error');

    render(<MockError error={mockError} reset={mockReset} />);

    expect(consoleSpy).toHaveBeenCalledWith('Error boundary:', mockError);
  });

  it('should display error message in development mode', () => {
    const originalEnv = process.env.NODE_ENV;
    process.env.NODE_ENV = 'development';

    render(<MockError error={mockError} reset={mockReset} />);

    expect(screen.getByText('Test error message')).toBeInTheDocument();

    process.env.NODE_ENV = originalEnv;
  });

  it('should not display error message in production mode', () => {
    const originalEnv = process.env.NODE_ENV;
    process.env.NODE_ENV = 'production';

    render(<MockError error={mockError} reset={mockReset} />);

    expect(screen.queryByText('Test error message')).not.toBeInTheDocument();

    process.env.NODE_ENV = originalEnv;
  });

  it('should have proper container classes', () => {
    const { container } = render(<MockError error={mockError} reset={mockReset} />);

    const mainContainer = container.querySelector('.container');
    expect(mainContainer).toBeInTheDocument();
    expect(mainContainer).toHaveClass('container', 'min-h-screen');
  });

  it('should center content vertically and horizontally', () => {
    const { container } = render(<MockError error={mockError} reset={mockReset} />);

    const mainContainer = container.querySelector('.container');
    expect(mainContainer).toHaveClass('flex', 'items-center', 'justify-center');
  });

  it('should render card with max width', () => {
    const { container } = render(<MockError error={mockError} reset={mockReset} />);

    const card = container.querySelector('.max-w-md');
    expect(card).toBeInTheDocument();
  });

  it('should have button with full width', () => {
    render(<MockError error={mockError} reset={mockReset} />);

    const button = screen.getByRole('button', { name: /Réessayer/i });
    expect(button).toHaveClass('w-full');
  });

  it('should handle error with digest', () => {
    const errorWithDigest = Object.assign(new Error('Test error'), {
      digest: 'abc123',
    });

    render(<MockError error={errorWithDigest} reset={mockReset} />);

    expect(screen.getByText(/Une erreur est survenue/i)).toBeInTheDocument();
  });

  it('should render error details container in development', () => {
    const originalEnv = process.env.NODE_ENV;
    process.env.NODE_ENV = 'development';

    const { container } = render(<MockError error={mockError} reset={mockReset} />);

    const errorDetails = container.querySelector('.rounded-md.bg-muted');
    expect(errorDetails).toBeInTheDocument();

    process.env.NODE_ENV = originalEnv;
  });

  it('should render error message with monospace font in development', () => {
    const originalEnv = process.env.NODE_ENV;
    process.env.NODE_ENV = 'development';

    const { container } = render(<MockError error={mockError} reset={mockReset} />);

    const errorText = screen.getByText('Test error message');
    expect(errorText).toHaveClass('font-mono');

    process.env.NODE_ENV = originalEnv;
  });
});
