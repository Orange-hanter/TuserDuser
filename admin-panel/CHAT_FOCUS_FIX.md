# Chat Optimization - Focus Fix

## Problem

When typing in the chat input field, the focus was lost every 5 seconds during the auto-refresh. This happened because the component was re-rendering completely on each data fetch, even if the comments didn't change.

## Solution

Optimized `EventCommentChat.js` with React performance best practices:

### 1. Added useRef to track data

- `prevCommentsRef` - tracks previous comments to detect changes
- `isMountedRef` - prevents state updates on unmounted components

### 2. Memoized fetchComments with useCallback

- Only updates state if comment data has changed (JSON comparison)
- Prevents unnecessary re-renders when polling

### 3. Memoized handleSendComment

- Uses useCallback to maintain stable function reference
- Dependencies optimized to avoid function recreation

### 4. Memoized renderComment

- Uses useCallback to prevent re-creating render function
- Improves FlatList item rendering performance

### 5. Wrapped component with React.memo

- Prevents re-renders from parent components unless props change
- Shallow comparison of `eventId` and `onClose` props

## Result

✅ Chat input maintains focus while typing
✅ Auto-refresh still works every 5 seconds
✅ No re-renders unless comments actually changed
✅ Smoother typing experience
