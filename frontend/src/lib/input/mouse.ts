import { SendMouseMove, SendMouseButton, SendMouseScroll, SendMouseAbs } from '../../../wailsjs/go/main/App';

let capturing = false;
let targetElement: HTMLElement | null = null;
let onCaptureExit: (() => void) | null = null;

// Accumulate mouse deltas and send at throttled rate (~60Hz)
let accDX = 0;
let accDY = 0;
let moveTimer: ReturnType<typeof setTimeout> | null = null;
const MOUSE_THROTTLE_MS = 16; // ~60 fps

// Track last position for delta calculation (fallback when movementX/Y unavailable)
let lastX = 0;
let lastY = 0;
let hasLast = false;

// Skip the first mousemove delta after re-entering the video container
// (prevents large movementX/Y jump)
let skipNextDelta = false;

function flushMouseMove() {
  moveTimer = null;
  if (accDX === 0 && accDY === 0) return;
  const dx = accDX;
  const dy = accDY;
  accDX = 0;
  accDY = 0;
  SendMouseMove(dx, dy).catch(console.error);
}

function onMouseEnter(e: MouseEvent) {
  if (!capturing) return;
  if (document.pointerLockElement) return;
  const abs = mapToAbsolute(e.clientX, e.clientY);
  if (abs) {
    SendMouseAbs(abs.x, abs.y).catch(console.error);
  }
  skipNextDelta = true;
}

function onMouseMove(e: MouseEvent) {
  if (!capturing) return;
  if (!document.pointerLockElement && !isInsideTarget(e)) return;

  if (skipNextDelta) {
    skipNextDelta = false;
    hasLast = false;
    accDX = 0;
    accDY = 0;
    return;
  }

  // Use movementX/Y if available (works both with and without pointer lock)
  let dx: number;
  let dy: number;
  if (e.movementX !== undefined) {
    dx = e.movementX;
    dy = e.movementY;
  } else if (hasLast) {
    dx = e.clientX - lastX;
    dy = e.clientY - lastY;
  } else {
    dx = 0;
    dy = 0;
  }
  lastX = e.clientX;
  lastY = e.clientY;
  hasLast = true;

  accDX += dx;
  accDY += dy;
  if (!moveTimer) {
    moveTimer = setTimeout(flushMouseMove, MOUSE_THROTTLE_MS);
  }
}

function isInsideTarget(e: MouseEvent): boolean {
  if (!targetElement) return false;
  return targetElement.contains(e.target as Node);
}

function onMouseDown(e: MouseEvent) {
  if (!capturing) return;
  // Click outside video container → exit capture, don't send to remote
  if (!isInsideTarget(e)) {
    exitCapture();
    if (onCaptureExit) onCaptureExit();
    return;
  }
  e.preventDefault();
  SendMouseButton(e.button, true).catch(console.error);
}

function onMouseUp(e: MouseEvent) {
  if (!capturing) return;
  if (!isInsideTarget(e)) return;
  e.preventDefault();
  SendMouseButton(e.button, false).catch(console.error);
}

function onWheel(e: WheelEvent) {
  // Wheel works without capture — just hovering over the video is enough
  e.preventDefault();
  // Normalize to -1/+1 (browser deltaY>0 = scroll down, but Linux REL_WHEEL>0 = scroll up)
  const delta = e.deltaY > 0 ? -1 : e.deltaY < 0 ? 1 : 0;
  if (delta !== 0) {
    SendMouseScroll(delta).catch(console.error);
  }
}

const ABS_MAX = 32767;

/**
 * Map a click position on the video container to normalized absolute
 * coordinates (0-32767), accounting for object-fit:contain letterboxing.
 */
function mapToAbsolute(clientX: number, clientY: number): { x: number; y: number } | null {
  if (!targetElement) return null;
  const container = targetElement;
  const img = container.querySelector('img.video-stream') as HTMLImageElement | null;
  const containerRect = container.getBoundingClientRect();

  let renderX = 0, renderY = 0;
  let renderW = containerRect.width, renderH = containerRect.height;

  if (img && img.naturalWidth && img.naturalHeight) {
    const videoAspect = img.naturalWidth / img.naturalHeight;
    const containerAspect = containerRect.width / containerRect.height;
    if (videoAspect > containerAspect) {
      // Letterbox top/bottom
      renderW = containerRect.width;
      renderH = containerRect.width / videoAspect;
      renderX = 0;
      renderY = (containerRect.height - renderH) / 2;
    } else {
      // Letterbox left/right
      renderH = containerRect.height;
      renderW = containerRect.height * videoAspect;
      renderX = (containerRect.width - renderW) / 2;
      renderY = 0;
    }
  }

  const relX = clientX - containerRect.left - renderX;
  const relY = clientY - containerRect.top - renderY;
  const normX = Math.max(0, Math.min(1, relX / renderW));
  const normY = Math.max(0, Math.min(1, relY / renderH));

  return { x: Math.round(normX * ABS_MAX), y: Math.round(normY * ABS_MAX) };
}

/** Called by VideoDisplay to enter capture mode.
 *  If clientX/Y are provided (from the click), send an absolute position
 *  to sync the remote cursor to where the user clicked.
 */
export function enterCapture(clientX?: number, clientY?: number) {
  capturing = true;
  hasLast = false;
  skipNextDelta = false;

  // Sync remote cursor to the click position via absolute coords
  if (clientX !== undefined && clientY !== undefined) {
    const abs = mapToAbsolute(clientX, clientY);
    if (abs) {
      SendMouseAbs(abs.x, abs.y).catch(console.error);
    }
  }

  // Try pointer lock for best experience, but don't depend on it
  if (targetElement) {
    try {
      targetElement.requestPointerLock();
    } catch (_) {
      // Pointer lock not supported — still works via movementX/clientX fallback
    }
  }
}

/** Called by VideoDisplay to exit capture mode */
export function exitCapture() {
  capturing = false;
  hasLast = false;
  skipNextDelta = false;
  if (moveTimer) {
    clearTimeout(moveTimer);
    moveTimer = null;
  }
  accDX = 0;
  accDY = 0;
  try {
    if (document.pointerLockElement) {
      document.exitPointerLock();
    }
  } catch (_) {
    // ignore
  }
}

export function isCapturing(): boolean {
  return capturing;
}

export function startMouseCapture(element: HTMLElement, onExit?: () => void) {
  targetElement = element;
  onCaptureExit = onExit || null;
  // Listen on document so events fire even with pointer lock
  document.addEventListener('mousemove', onMouseMove);
  document.addEventListener('mousedown', onMouseDown);
  document.addEventListener('mouseup', onMouseUp);
  element.addEventListener('mouseenter', onMouseEnter);
  element.addEventListener('wheel', onWheel, { passive: false });
  // Sync capturing state when pointer lock changes (e.g. user presses Esc)
  document.addEventListener('pointerlockchange', () => {
    if (!document.pointerLockElement && capturing) {
      // Pointer lock was exited (e.g. Esc) — exit capture
      capturing = false;
      hasLast = false;
      if (onCaptureExit) onCaptureExit();
    }
  });
}

export function stopMouseCapture(element: HTMLElement) {
  exitCapture();
  targetElement = null;
  document.removeEventListener('mousemove', onMouseMove);
  document.removeEventListener('mousedown', onMouseDown);
  document.removeEventListener('mouseup', onMouseUp);
  element.removeEventListener('mouseenter', onMouseEnter);
  element.removeEventListener('wheel', onWheel);
}
