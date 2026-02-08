import { SendMouseMove, SendMouseButton, SendMouseScroll } from '../../../wailsjs/go/main/App';

let capturing = false;

// Accumulate mouse deltas and send at throttled rate (~60Hz)
let accDX = 0;
let accDY = 0;
let moveTimer: ReturnType<typeof setTimeout> | null = null;
const MOUSE_THROTTLE_MS = 16; // ~60 fps

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
  accDX += e.movementX;
  accDY += e.movementY;
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

function onPointerLockChange() {
  capturing = document.pointerLockElement !== null;
}

export function requestPointerLock(element: HTMLElement) {
  element.requestPointerLock();
}

export function exitPointerLock() {
  if (document.pointerLockElement) {
    document.exitPointerLock();
  }
  capturing = false;
}

export function startMouseCapture(element: HTMLElement) {
  // During pointer lock, mouse events fire on document, not the element.
  // Listen on both to handle locked and unlocked states.
  document.addEventListener('mousemove', onMouseMove);
  document.addEventListener('mousedown', onMouseDown);
  document.addEventListener('mouseup', onMouseUp);
  element.addEventListener('wheel', onWheel, { passive: false });
  document.addEventListener('pointerlockchange', onPointerLockChange);
}

export function stopMouseCapture(element: HTMLElement) {
  capturing = false;
  if (moveTimer) {
    clearTimeout(moveTimer);
    moveTimer = null;
  }
  accDX = 0;
  accDY = 0;
  exitPointerLock();
  document.removeEventListener('mousemove', onMouseMove);
  document.removeEventListener('mousedown', onMouseDown);
  document.removeEventListener('mouseup', onMouseUp);
  element.removeEventListener('wheel', onWheel);
  document.removeEventListener('pointerlockchange', onPointerLockChange);
}
