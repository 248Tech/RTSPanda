# UX Design Brief: Camera Dashboard

Status: Design complete — Ready for Cursor (TASK-007)
Last updated: 2026-03-09

---

## Purpose

This document defines the visual layout, interaction model, and component structure
for the RTSPanda camera dashboard. Cursor should use this as the design source of truth
when implementing TASK-007 (Dashboard) and TASK-008 (Video Player).

---

## Design Principles

- **Speed first.** The page should feel instant. Load camera list fast; stream lazily.
- **Dark by default.** Camera grids look better on dark backgrounds. Match the aesthetic of security/monitoring tools.
- **Dense but not cramped.** Show as many cameras as possible without sacrificing readability.
- **No clutter.** Only show what the user needs: camera name, status, and video.

---

## Color Palette

| Token            | Value       | Use                                   |
|------------------|-------------|---------------------------------------|
| `bg-base`        | `#0f1117`   | Page background                       |
| `bg-card`        | `#1a1d27`   | Camera card background                |
| `bg-card-hover`  | `#22263a`   | Card hover state                      |
| `border`         | `#2e3150`   | Card border, dividers                 |
| `text-primary`   | `#e2e8f0`   | Camera name, main text                |
| `text-muted`     | `#64748b`   | Secondary text, labels                |
| `status-online`  | `#22c55e`   | Green — stream active                 |
| `status-offline` | `#ef4444`   | Red — stream unreachable              |
| `status-connecting` | `#f59e0b` | Amber — attempting connection        |
| `accent`         | `#6366f1`   | Primary action buttons, focus rings   |

Use Tailwind CSS custom config or arbitrary values — do not use a CSS-in-JS library.

---

## Layout

### Responsive Grid

```
Mobile  (< 640px):   1 column
Tablet  (640-1024px): 2 columns
Desktop (1024-1280px): 3 columns
Wide    (> 1280px):   4 columns
```

Tailwind grid: `grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4`

### Page Structure

```
┌──────────────────────────────────────────────────────────┐
│  RTSPanda        [Settings ⚙]         ← Navbar (fixed)  │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐    │
│  │  Camera 1    │ │  Camera 2    │ │  Camera 3    │    │
│  │              │ │              │ │              │    │
│  │  [video/     │ │  [video/     │ │  [video/     │    │
│  │   thumbnail] │ │   thumbnail] │ │   thumbnail] │    │
│  │              │ │              │ │              │    │
│  │ ● Online     │ │ ● Offline    │ │ ○ Connecting │    │
│  │ Front Door   │ │ Backyard     │ │ Garage       │    │
│  └──────────────┘ └──────────────┘ └──────────────┘    │
│                                                          │
│  ┌──────────────┐                                        │
│  │  Camera 4    │                                        │
│  └──────────────┘                                        │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

---

## Component Breakdown

### `Dashboard.tsx` (page)

- Fetches camera list via `getCameras()` on mount
- Renders `<CameraGrid cameras={cameras} />`
- Shows `<EmptyState />` if camera list is empty
- Polls camera list every 30 seconds for changes (simple interval, not WebSocket)

### `CameraGrid.tsx`

```tsx
// Props
interface CameraGridProps {
  cameras: Camera[]
}
```

- Renders the responsive CSS grid
- Maps cameras to `<CameraCard />` components
- Handles empty array (should not reach here — Dashboard handles empty state)

### `CameraCard.tsx`

```tsx
// Props
interface CameraCardProps {
  camera: Camera
}
```

**Layout (within card):**

```
┌─────────────────────────────────┐
│                                 │  ← 16:9 aspect ratio thumbnail area
│    [VideoThumbnail or Placeholder] │
│                                 │
├─────────────────────────────────┤
│  ● Online          Front Door   │  ← Status badge + camera name
└─────────────────────────────────┘
```

**Behavior:**
- Clicking the card navigates to `/cameras/:id`
- Hovering the card shows a subtle border highlight (`bg-card-hover`)
- The thumbnail area uses a fixed 16:9 aspect ratio (`aspect-video` in Tailwind)
- In Phase 1, the thumbnail area is a placeholder (dark grey with a camera icon)
  - Do NOT attempt to load video on the dashboard grid — only on the single camera view
  - This is a performance decision: loading 12 HLS streams simultaneously would be unusable

**Card states:**
- Online: green dot, normal appearance
- Offline: red dot, thumbnail area has a subtle red tint overlay (opacity 10%)
- Connecting: amber dot, thumbnail area shows a spinner

### `StatusBadge.tsx`

```tsx
// Props
interface StatusBadgeProps {
  status: 'online' | 'offline' | 'connecting'
}
```

- Inline component: `● Online` / `● Offline` / `○ Connecting`
- Dot is a filled circle for online/offline, hollow circle for connecting
- Uses `status-online`, `status-offline`, `status-connecting` colors from palette

### `EmptyState.tsx`

Shown on the dashboard when there are no cameras configured.

```
┌──────────────────────────────────────┐
│                                      │
│           📷                         │
│                                      │
│    No cameras configured             │
│    Add your first camera to get      │
│    started.                          │
│                                      │
│    [+ Add Camera]                    │
│                                      │
└──────────────────────────────────────┘
```

- Centered in the page
- "Add Camera" button navigates to `/settings`

---

## Navbar

```
┌──────────────────────────────────────────────────────────┐
│  [🐼] RTSPanda                             [⚙ Settings] │
└──────────────────────────────────────────────────────────┘
```

- Fixed to top, full width
- Background: `bg-card` with bottom border `border-border`
- Logo: text-based "RTSPanda" with a panda emoji (or simple SVG — keep it minimal)
- Settings link: right-aligned, navigates to `/settings`
- Height: 56px (h-14)

---

## Single Camera View (`/cameras/:id`)

Implemented in TASK-008. Design summary:

```
┌──────────────────────────────────────────────────────────┐
│  ← Back to Dashboard                      [⚙ Settings]  │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │                                                  │   │
│  │              [HLS Video Player]                  │   │
│  │                                                  │   │
│  │                                                  │   │
│  └──────────────────────────────────────────────────┘   │
│                                                          │
│  Front Door Camera                    ● Online           │
│  rtsp://192.168.1.100/stream1                            │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

- Video player fills available width, maintains 16:9 aspect ratio
- Camera name and status below player
- RTSP URL shown (read-only) for debugging
- "Back" link at top left returns to dashboard

---

## Interaction Model

| Interaction                   | Result                               |
|-------------------------------|--------------------------------------|
| Click camera card             | Navigate to `/cameras/:id`           |
| Click Settings nav link       | Navigate to `/settings`              |
| Click "Add Camera" (empty state) | Navigate to `/settings`           |
| Back button (camera view)     | Navigate to `/`                      |

No modals on the dashboard itself. The dashboard is read-only — all management is in Settings.

---

## Accessibility

- All interactive elements must be keyboard focusable
- Focus rings use `accent` color (`ring-indigo-500`)
- Status badges must not rely on color alone — include text label
- Images/icons must have `alt` text or `aria-label`

---

## What NOT to Build (Dashboard Scope)

- No drag-and-drop camera reordering in Phase 1
- No per-card mute/fullscreen controls — these are on the single camera view only
- No real-time stream thumbnails on the grid — placeholder only
- No grid layout switcher (list vs grid) — grid only for Phase 1
- No search or filter bar — not needed until there are many cameras

---

## Implementation Order

TASK-007 and TASK-008 should be done in sequence:

1. **TASK-007:** Build Dashboard + CameraGrid + CameraCard + StatusBadge + EmptyState
   - Uses placeholder thumbnail (grey box with icon)
   - Fetches real camera data from API
2. **TASK-008:** Build VideoPlayer + CameraView
   - Add hls.js player to the single camera view
   - Dashboard thumbnails remain as placeholders

Do not add hls.js to CameraCard. Only CameraView gets the real player.
