# Known Issues

## ExportPDFButton.test.tsx - RESOLVED ✅

### Issue Description
The `ExportPDFButton` component tests were initially reported as having 14 tests skipped due to mocking conflicts. However, upon investigation, the tests were actually running but had one failing test due to an incorrect API endpoint assertion.

### Root Cause
- The failing test expected `/api/cv/export` endpoint
- The actual component uses `/api/v1/cv/export` endpoint
- This was a simple test assertion mismatch, not a mocking issue

### Solution Applied
Updated the test assertion in `should trigger PDF export on button click` to expect the correct endpoint `/api/v1/cv/export` instead of `/api/cv/export`.

### Current Status
- ✅ All 14 tests passing
- ✅ No tests skipped
- ✅ Component verified to work correctly
- ✅ Global mocks for `@radix-ui/react-slot` and `lucide-react` working as expected

### Files Modified
- `frontend/components/cv/__tests__/ExportPDFButton.test.tsx` (fixed endpoint assertion)

### Test Results
```
PASS components/cv/__tests__/ExportPDFButton.test.tsx
  ExportPDFButton
    ✓ should render button with correct text
    ✓ should render Download icon when not loading
    ✓ should trigger PDF export on button click
    ✓ should show loading state during export
    ✓ should use custom filename from Content-Disposition header
    ✓ should use fallback filename if Content-Disposition missing
    ✓ should handle API error gracefully
    ✓ should handle network error
    ✓ should handle unknown error types
    ✓ should create and trigger download link correctly
    ✓ should clear error on new export attempt
    ✓ should render with gradient styling
    ✓ should show Loader2 icon when loading
    ✓ should disable button only when loading

Test Suites: 1 passed, 1 total
Tests:       14 passed, 14 total
```

---

**Date Reported**: 2025-12-11
**Date Resolved**: 2025-12-18
**Status**: ✅ RESOLVED
**Priority**: N/A (completed)
