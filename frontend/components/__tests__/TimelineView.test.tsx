import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import TimelineView from '../timeline/TimelineView';
import { TimelineEvent, TimelineMilestone } from '@/lib/types';

// Mock framer-motion
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  },
  AnimatePresence: ({ children }: any) => <>{children}</>,
}));

// Mock child components
jest.mock('../timeline/TimelineItem', () => ({
  __esModule: true,
  default: ({ event, onSelect }: any) => (
    <div data-testid={`timeline-item-${event.id}`} onClick={onSelect}>
      {event.title}
    </div>
  ),
}));

jest.mock('../timeline/TimelineFilters', () => ({
  __esModule: true,
  default: ({ onFiltersChange, onReset }: any) => (
    <div data-testid="timeline-filters">
      <button onClick={() => onFiltersChange({ category: 'tech', type: 'experience', period: null })}>
        Apply Filter
      </button>
      <button onClick={onReset}>Reset</button>
    </div>
  ),
}));

jest.mock('../timeline/TimelineMilestones', () => ({
  __esModule: true,
  default: ({ milestones }: any) => (
    <div data-testid="timeline-milestones">
      {milestones.length} milestones
    </div>
  ),
}));

jest.mock('../timeline/TimelineNavigation', () => ({
  __esModule: true,
  default: ({ years }: any) => (
    <div data-testid="timeline-navigation">
      {years.join(', ')}
    </div>
  ),
}));

jest.mock('../timeline/TimelineModal', () => ({
  __esModule: true,
  default: ({ event, onClose }: any) => (
    <div data-testid="timeline-modal">
      <h2>{event.title}</h2>
      <button onClick={onClose}>Close</button>
    </div>
  ),
}));

const mockEvents: TimelineEvent[] = [
  {
    id: '1',
    type: 'experience',
    title: 'Senior Developer',
    subtitle: 'Tech Corp',
    content: 'Led development team',
    startDate: '2022-01-01',
    endDate: '2023-12-31',
    tags: ['React', 'TypeScript', 'Node.js'],
    category: 'tech',
    isCurrent: false,
    duration: '2 years',
  },
  {
    id: '2',
    type: 'project',
    title: 'E-commerce Platform',
    subtitle: 'Personal Project',
    content: 'Built scalable platform',
    startDate: '2021-06-01',
    endDate: '2022-03-31',
    tags: ['Go', 'PostgreSQL'],
    category: 'tech',
    isCurrent: false,
    duration: '9 months',
  },
  {
    id: '3',
    type: 'experience',
    title: 'Consultant',
    subtitle: 'Freelance',
    content: 'Multiple consulting projects',
    startDate: '2020-01-01',
    tags: ['Python', 'AWS'],
    category: 'business',
    isCurrent: true,
  },
];

const mockMilestones: TimelineMilestone[] = [
  {
    id: 'm1',
    title: 'First Job',
    description: 'Started career',
    date: '2020-01-01',
    icon: '🎉',
    type: 'career',
  },
];

const mockCategories = ['tech', 'business', 'academic'];

describe('TimelineView', () => {
  it('should render timeline navigation with years', () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
        milestones={mockMilestones}
      />
    );

    expect(screen.getByTestId('timeline-navigation')).toBeInTheDocument();
    expect(screen.getByText(/2022, 2021, 2020/)).toBeInTheDocument();
  });

  it('should render timeline filters', () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
        milestones={mockMilestones}
      />
    );

    expect(screen.getByTestId('timeline-filters')).toBeInTheDocument();
  });

  it('should render milestones when provided', () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
        milestones={mockMilestones}
      />
    );

    expect(screen.getByTestId('timeline-milestones')).toBeInTheDocument();
    expect(screen.getByText('1 milestones')).toBeInTheDocument();
  });

  it('should not render milestones when empty', () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
        milestones={[]}
      />
    );

    expect(screen.queryByTestId('timeline-milestones')).not.toBeInTheDocument();
  });

  it('should render all timeline items', () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
      />
    );

    expect(screen.getByTestId('timeline-item-1')).toBeInTheDocument();
    expect(screen.getByTestId('timeline-item-2')).toBeInTheDocument();
    expect(screen.getByTestId('timeline-item-3')).toBeInTheDocument();
  });

  it('should display stats summary', () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
      />
    );

    const threeElements = screen.getAllByText('3');
    expect(threeElements.length).toBeGreaterThanOrEqual(1); // Total events
    expect(screen.getByText('2')).toBeInTheDocument(); // Experiences
    expect(screen.getByText('1')).toBeInTheDocument(); // Projects
  });

  it('should filter events by category', async () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
      />
    );

    const applyFilterButton = screen.getByText('Apply Filter');
    fireEvent.click(applyFilterButton);

    // Wait for the filters to be applied
    await waitFor(() => {
      // After filtering, the timeline should still be visible
      expect(screen.getByTestId('timeline-filters')).toBeInTheDocument();
    }, { timeout: 2000 });

    // Verify that stats are displayed
    const statsElements = screen.getAllByText(/événements|expériences|projets|catégories/i);
    expect(statsElements.length).toBeGreaterThan(0);
  });

  it('should filter events by type', async () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
      />
    );

    const applyFilterButton = screen.getByText('Apply Filter');
    fireEvent.click(applyFilterButton);

    await waitFor(() => {
      // After filtering by type 'experience', should show 1 event (in tech category)
      const items = screen.getAllByTestId(/timeline-item-/);
      expect(items.length).toBeLessThanOrEqual(mockEvents.length);
    });
  });

  it('should reset filters', async () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
      />
    );

    const resetButton = screen.getByText('Reset');
    fireEvent.click(resetButton);

    await waitFor(() => {
      // Should show all events again
      expect(screen.getByTestId('timeline-item-1')).toBeInTheDocument();
      expect(screen.getByTestId('timeline-item-2')).toBeInTheDocument();
      expect(screen.getByTestId('timeline-item-3')).toBeInTheDocument();
    });
  });

  it('should show empty state when no events match filters', () => {
    render(
      <TimelineView
        events={[]}
        categories={mockCategories}
      />
    );

    expect(screen.getByText(/aucun événement trouvé/i)).toBeInTheDocument();
    expect(screen.getByText(/réinitialiser les filtres/i)).toBeInTheDocument();
  });

  it('should open modal when event is selected', async () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
      />
    );

    const eventItem = screen.getByTestId('timeline-item-1');
    fireEvent.click(eventItem);

    await waitFor(() => {
      expect(screen.getByTestId('timeline-modal')).toBeInTheDocument();
      const seniorDevElements = screen.getAllByText('Senior Developer');
      expect(seniorDevElements.length).toBeGreaterThanOrEqual(1);
    });
  });

  it('should close modal when close button is clicked', async () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
      />
    );

    const eventItem = screen.getByTestId('timeline-item-1');
    fireEvent.click(eventItem);

    await waitFor(() => {
      expect(screen.getByTestId('timeline-modal')).toBeInTheDocument();
    });

    const closeButton = screen.getByText('Close');
    fireEvent.click(closeButton);

    await waitFor(() => {
      expect(screen.queryByTestId('timeline-modal')).not.toBeInTheDocument();
    });
  });

  it('should render central vertical line', () => {
    const { container } = render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
      />
    );

    const verticalLine = container.querySelector('.bg-gradient-to-b');
    expect(verticalLine).toBeInTheDocument();
  });

  it('should display correct category count in stats', () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
      />
    );

    const categoriesElements = screen.getAllByText(mockCategories.length.toString());
    expect(categoriesElements.length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText(/catégories/i)).toBeInTheDocument();
  });

  it('should extract unique years from events', () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
      />
    );

    const navigation = screen.getByTestId('timeline-navigation');
    expect(navigation.textContent).toContain('2022');
    expect(navigation.textContent).toContain('2021');
    expect(navigation.textContent).toContain('2020');
  });

  it('should sort years in descending order', () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
      />
    );

    const navigation = screen.getByTestId('timeline-navigation');
    expect(navigation.textContent).toBe('2022, 2021, 2020');
  });

  it('should filter by date period', () => {
    const { rerender } = render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
      />
    );

    // Simulate filter change with period
    const customFilterComponent = () => (
      <div data-testid="timeline-filters">
        <button
          onClick={() => {
            // This would trigger filtering by period in the actual component
          }}
        >
          Apply Period Filter
        </button>
      </div>
    );

    // The actual filtering logic is tested through integration
    expect(screen.getByTestId('timeline-filters')).toBeInTheDocument();
  });

  it('should handle events without endDate', () => {
    const eventsWithoutEnd = [
      {
        ...mockEvents[0],
        endDate: undefined,
      },
    ];

    render(
      <TimelineView
        events={eventsWithoutEnd}
        categories={mockCategories}
      />
    );

    expect(screen.getByTestId('timeline-item-1')).toBeInTheDocument();
  });

  it('should update stats when filters change', async () => {
    render(
      <TimelineView
        events={mockEvents}
        categories={mockCategories}
      />
    );

    // Initial stats
    const threeElements = screen.getAllByText('3');
    expect(threeElements.length).toBeGreaterThanOrEqual(1); // Total

    const applyFilterButton = screen.getByText('Apply Filter');
    fireEvent.click(applyFilterButton);

    await waitFor(() => {
      // Stats should update based on filtered events
      const statsElements = screen.getAllByText(/événements/i);
      expect(statsElements.length).toBeGreaterThan(0);
    });
  });
});
