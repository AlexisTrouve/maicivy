import React from 'react';

// Mock @radix-ui/react-select with simplified components that render with proper ARIA roles

export const Root = ({ children, value, onValueChange, defaultValue, open, defaultOpen, onOpenChange, dir, name, autoComplete, disabled, required, ...props }: any) => {
  // Create a context-like mechanism to pass value/onValueChange only where needed
  // Don't clone all children with these props as they shouldn't go to Portal/Content divs
  return (
    <div data-radix-select-root {...props}>
      {children}
    </div>
  );
};

export const Trigger = React.forwardRef<HTMLButtonElement, any>(
  ({ children, value, onValueChange, className, ...props }, ref) => {
    return (
      <button
        ref={ref}
        role="combobox"
        aria-expanded="false"
        data-radix-select-trigger
        data-state="closed"
        className={className}
        {...props}
      >
        {children}
      </button>
    );
  }
);
Trigger.displayName = 'SelectTrigger';

export const Value = ({ placeholder, ...props }: any) => {
  return <span {...props}>{placeholder || 'Select...'}</span>;
};

export const Icon = ({ children, asChild, ...props }: any) => {
  if (asChild && React.isValidElement(children)) {
    return children;
  }
  return <span {...props}>{children}</span>;
};

export const Portal = ({ children, container, ...props }: any) => {
  // Filter out any props that shouldn't be passed to children
  const cleanChildren = React.Children.map(children, (child) => {
    if (React.isValidElement(child)) {
      // Don't pass down onValueChange or value props
      return child;
    }
    return child;
  });
  return <>{cleanChildren}</>;
};

export const Content = React.forwardRef<HTMLDivElement, any>(
  ({ children, position, className, onCloseAutoFocus, onEscapeKeyDown, onPointerDownOutside, ...props }, ref) => {
    return (
      <div
        ref={ref}
        role="listbox"
        data-radix-select-content
        data-state="open"
        className={className}
        {...props}
      >
        {children}
      </div>
    );
  }
);
Content.displayName = 'SelectContent';

export const Viewport = ({ children, className, ...props }: any) => {
  return (
    <div className={className} {...props}>
      {children}
    </div>
  );
};

export const Item = React.forwardRef<HTMLDivElement, any>(
  ({ children, value, className, ...props }, ref) => {
    return (
      <div
        ref={ref}
        role="option"
        data-radix-select-item
        data-value={value}
        className={className}
        {...props}
      >
        {children}
      </div>
    );
  }
);
Item.displayName = 'SelectItem';

export const ItemText = ({ children, ...props }: any) => {
  return <span {...props}>{children}</span>;
};

export const ItemIndicator = ({ children, ...props }: any) => {
  return <span {...props}>{children}</span>;
};

export const ScrollUpButton = React.forwardRef<HTMLDivElement, any>(
  ({ children, className, ...props }, ref) => {
    return (
      <div ref={ref} className={className} {...props}>
        {children}
      </div>
    );
  }
);
ScrollUpButton.displayName = 'SelectScrollUpButton';

export const ScrollDownButton = React.forwardRef<HTMLDivElement, any>(
  ({ children, className, ...props }, ref) => {
    return (
      <div ref={ref} className={className} {...props}>
        {children}
      </div>
    );
  }
);
ScrollDownButton.displayName = 'SelectScrollDownButton';

export const Group = ({ children, ...props }: any) => {
  return <div {...props}>{children}</div>;
};

export const Label = React.forwardRef<HTMLDivElement, any>(
  ({ children, className, ...props }, ref) => {
    return (
      <div ref={ref} className={className} {...props}>
        {children}
      </div>
    );
  }
);
Label.displayName = 'SelectLabel';

export const Separator = React.forwardRef<HTMLDivElement, any>(
  ({ className, ...props }, ref) => {
    return <div ref={ref} className={className} {...props} />;
  }
);
Separator.displayName = 'SelectSeparator';
