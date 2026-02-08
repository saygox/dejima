import { SendMouseMove, SendMouseButton, SendMouseScroll } from '../../../wailsjs/go/main/App';

let capturing = false;
let targetElement: HTMLElement | null = null;

// Accumulate mouse deltas and send at throttled rate (~60Hz)
let accDX = 0;
let accDY = 0;
let moveTimer: ReturnType<typeof setTimeout> | null = null;
const MOUSE_THROTTLE_MS = 16; // ~60 fps

// Track last position for delta calculation (fallback when movementX/Y unavailable)
let lastX = 0;
let lastY = 0;
let hasLast = false;

function flushMouseMove() {
  moveTimer = null;
  if (accDX === 0 && accDY === 0) return;
  const dx = accDX;
  const dy = accDY;
  accDX = 0;
  accDY = 0;
  SendMouseMove(dx, dy).catch(console.error);
}

function onMouseMove(e: MouseEvent) {
  if (!capturing) return;

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

function onMouseDown(e: MouseEvent) {
  if (!capturing) return;
  e.preventDefault();
  SendMouseButton(e.button, true).catch(console.error);
}

function onMouseUp(e: MouseEvent) {
  if (!capturing) return;
  e.preventDefault();
  SendMouseButton(e.button, false).catch(console.error);
}

function onWheel(e: WheelEvent) {
  if (!capturing) return;
  e.preventDefault();
  // Normalize to -1/+1
  const delta = e.deltaY > 0 ? 1 : e.deltaY < 0 ? -1 : 0;
  if (delta !== 0) {
    SendMouseScroll(delta).catch(console.error);
  }
}

/** Called by VideoDisplay to enter capture mode */
export function enterCapture() {
  capturing = true;
  hasLast = false;
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

export function startMouseCapture(element: HTMLElement) {
  targetElement = element;
  // Listen on document so events fire even with pointer lock
  document.addEventListener('mousemove', onMouseMove);
  document.addEventListener('mousedown', onMouseDown);
  document.addEventListener('mouseup', onMouseUp);
  element.addEventListener('wheel', onWheel, { passive: false });
  // Sync capturing state when pointer lock changes (e.g. user presses Esc)
  document.addEventListener('pointerlockchange', () => {
    if (!document.pointerLockElement && capturing) {
      // Pointer lock was exited (e.g. Esc) — exit capture
      capturing = false;
      hasLast = false;
    }
  });
}

export function stopMouseCapture(element: HTMLElement) {
  exitCapture();
  targetElement = null;
  document.removeEventListener('mousemove', onMouseMove);
  document.removeEventListener('mousedown', onMouseDown);
  document.removeEventListener('mouseup', onMouseUp);
  element.removeEventListener('wheel', onWheel);
}
