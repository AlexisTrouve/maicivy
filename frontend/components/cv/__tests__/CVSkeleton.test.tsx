import { render, screen } from '@testing-library/react';
import { CVSkeleton } from '../CVSkeleton';

describe('CVSkeleton', () => {
  afterEach(() => {
    // Clear all timers to prevent memory leaks
    jest.clearAllTimers();
  });
  it('should render skeleton component', () => {
    const { container } = render(<CVSkeleton />);

    expect(container.querySelector('.animate-pulse')).toBeInTheDocument();
  });

  it('should render experiences skeleton section', () => {
    const { container } = render(<CVSkeleton />);

    // Should have section header skeleton
    const sectionHeaders = container.querySelectorAll('.h-8.bg-gray-200');
    expect(sectionHeaders.length).toBeGreaterThan(0);

    // Should have 3 experience items
    const experienceItems = container.querySelectorAll('.ml-20.bg-gray-200.rounded-lg.h-48');
    expect(experienceItems.length).toBe(3);
  });

  it('should render skills skeleton section', () => {
    const { container } = render(<CVSkeleton />);

    // Should have 8 skill items
    const skillItems = container.querySelectorAll('.h-10.bg-gray-200.rounded-full');
    expect(skillItems.length).toBe(8);
  });

  it('should render projects skeleton section', () => {
    const { container } = render(<CVSkeleton />);

    // Should have 6 project cards in grid
    const projectCards = container.querySelectorAll('.bg-gray-200.rounded-lg.h-64');
    expect(projectCards.length).toBe(6);
  });

  it('should have proper grid layout for projects', () => {
    const { container } = render(<CVSkeleton />);

    const projectGrid = container.querySelector('.grid.grid-cols-1.md\\:grid-cols-2.lg\\:grid-cols-3');
    expect(projectGrid).toBeInTheDocument();
  });

  it('should render with space-y-16 layout', () => {
    const { container } = render(<CVSkeleton />);

    const mainContainer = container.querySelector('.space-y-16');
    expect(mainContainer).toBeInTheDocument();
  });

  it('should apply animate-pulse to main container', () => {
    const { container } = render(<CVSkeleton />);

    const mainContainer = container.querySelector('.space-y-16.animate-pulse');
    expect(mainContainer).toBeInTheDocument();
  });

  it('should render dark mode classes', () => {
    const { container } = render(<CVSkeleton />);

    const darkElements = container.querySelectorAll('.dark\\:bg-gray-700');
    expect(darkElements.length).toBeGreaterThan(0);
  });

  it('should have variable width skill skeletons', () => {
    const { container } = render(<CVSkeleton />);

    const skillItems = container.querySelectorAll('.h-10.bg-gray-200.rounded-full');

    // Each skill should have inline style with width
    skillItems.forEach((item) => {
      const width = item.getAttribute('style');
      expect(width).toMatch(/width:\s*\d+(\.\d+)?px/);
    });
  });

  it('should render 3 sections total', () => {
    const { container } = render(<CVSkeleton />);

    const sections = container.querySelectorAll('section');
    expect(sections.length).toBe(3);
  });

  it('should have rounded corners on skeleton elements', () => {
    const { container } = render(<CVSkeleton />);

    const roundedElements = container.querySelectorAll('.rounded-lg, .rounded, .rounded-full');
    expect(roundedElements.length).toBeGreaterThan(15);
  });

  it('should render skills in flex wrap container', () => {
    const { container } = render(<CVSkeleton />);

    const skillsContainer = container.querySelector('.flex.flex-wrap.gap-3');
    expect(skillsContainer).toBeInTheDocument();
  });

  it('should have consistent spacing with space-y-8 for experiences', () => {
    const { container } = render(<CVSkeleton />);

    const experienceContainer = container.querySelector('.space-y-8');
    expect(experienceContainer).toBeInTheDocument();
  });

  it('should render section headers with consistent styling', () => {
    const { container } = render(<CVSkeleton />);

    const sectionHeaders = container.querySelectorAll('.h-8.bg-gray-200.rounded');
    // Should have at least 3 section headers (experiences, skills, projects)
    expect(sectionHeaders.length).toBeGreaterThanOrEqual(3);
  });

  it('should have proper gap spacing in grid', () => {
    const { container } = render(<CVSkeleton />);

    const projectGrid = container.querySelector('.gap-6');
    expect(projectGrid).toBeInTheDocument();
  });

  it('should apply mb-8 margin to section headers', () => {
    const { container } = render(<CVSkeleton />);

    const headersWithMargin = container.querySelectorAll('.mb-8');
    expect(headersWithMargin.length).toBeGreaterThanOrEqual(3);
  });

  it('should render compact layout for mobile', () => {
    const { container } = render(<CVSkeleton />);

    // Grid should start with grid-cols-1
    const grid = container.querySelector('[class*="grid-cols-1"]');
    expect(grid).toBeInTheDocument();
  });

  it('should have proper padding on skills container', () => {
    const { container } = render(<CVSkeleton />);

    const skillsWrapper = container.querySelector('.p-8');
    expect(skillsWrapper).toBeInTheDocument();
  });
});
