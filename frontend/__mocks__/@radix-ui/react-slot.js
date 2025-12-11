const React = require('react');

const Slot = React.forwardRef(({ children, ...props }, ref) => {
  if (React.isValidElement(children)) {
    return React.cloneElement(children, {
      ...props,
      ...children.props,
      ref,
    });
  }

  if (React.Children.count(children) > 1) {
    return React.Children.only(null);
  }

  return React.createElement('div', { ...props, ref }, children);
});

Slot.displayName = 'Slot';

module.exports = {
  Slot,
  Slottable: ({ children }) => children,
};
