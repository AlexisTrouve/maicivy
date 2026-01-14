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

const Slottable = ({ children }) => children;
Slottable.__radixId = Symbol('Slottable');

// Mock createSlot - returns the Slot component
const createSlot = (ownerName) => {
  const CustomSlot = React.forwardRef(({ children, ...props }, ref) => {
    return React.createElement(Slot, { ...props, ref }, children);
  });
  CustomSlot.displayName = `${ownerName}Slot`;
  return CustomSlot;
};

// Mock createSlottable - returns a Slottable component
const createSlottable = (ownerName) => {
  const CustomSlottable = ({ children }) => children;
  CustomSlottable.__radixId = Symbol(`${ownerName}Slottable`);
  CustomSlottable.displayName = `${ownerName}Slottable`;
  return CustomSlottable;
};

module.exports = {
  Slot,
  Root: Slot,
  Slottable,
  createSlot,
  createSlottable,
};
