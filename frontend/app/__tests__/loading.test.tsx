import { render, screen } from '@testing-library/react';
import Loading from '../loading';

// Mock LoadingSpinner component
jest.mock('@/components/shared/LoadingSpinner', () => ({
  LoadingSpinner: ({ size }: { size: string }) => (
    <div data-testid="loading-spinner" data-size={size}>
      Loading Spinner
    </div>
  ),
}));

describe('Loading Page', () => {
  it('should render loading spinner', () => {
    render(<Loading />);

    const spinner = screen.getByTestId('loading-spinner');
    expect(spinner).toBeInTheDocument();
  });

  it('should render loading spinner with large size', () => {
    render(<Loading />);

    const spinner = screen.getByTestId('loading-spinner');
    expect(spinner).toHaveAttribute('data-size', 'lg');
  });

  it('should render loading text', () => {
    render(<Loading />);

    expect(screen.getByText(/Chargement.../i)).toBeInTheDocument();
  });

  it('should have minimum height of screen', () => {
    const { container } = render(<Loading />);

    const mainDiv = container.querySelector('.min-h-screen');
    expect(mainDiv).toBeInTheDocument();
  });

  it('should center content vertically and horizontally', () => {
    const { container } = render(<Loading />);

    const mainDiv = container.querySelector('.min-h-screen');
    expect(mainDiv).toHaveClass('flex', 'items-center', 'justify-center');
  });

  it('should have text centered', () => {
    const { container } = render(<Loading />);

    const textCenter = container.querySelector('.text-center');
    expect(textCenter).toBeInTheDocument();
  });

  it('should have proper spacing for loading text', () => {
    const { container } = render(<Loading />);

    const loadingText = screen.getByText(/Chargement.../i);
    expect(loadingText).toHaveClass('mt-4');
  });

  it('should have muted foreground color for text', () => {
    const { container } = render(<Loading />);

    const loadingText = screen.getByText(/Chargement.../i);
    expect(loadingText).toHaveClass('text-muted-foreground');
  });

  it('should render spinner before text', () => {
    const { container } = render(<Loading />);

    const textCenter = container.querySelector('.text-center');
    const children = textCenter?.children;

    expect(children?.[0]).toHaveAttribute('data-testid', 'loading-spinner');
    expect(children?.[1]).toHaveTextContent('Chargement...');
  });

  it('should have proper structure', () => {
    const { container } = render(<Loading />);

    // Outer div with flex centering
    const outerDiv = container.querySelector('.min-h-screen.flex');
    expect(outerDiv).toBeInTheDocument();

    // Inner div with text-center
    const innerDiv = outerDiv?.querySelector('.text-center');
    expect(innerDiv).toBeInTheDocument();

    // Spinner inside inner div
    const spinner = innerDiv?.querySelector('[data-testid="loading-spinner"]');
    expect(spinner).toBeInTheDocument();
  });
});
