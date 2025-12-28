<script>
  import { SetCropRegion, ClearCropRegion } from '../../wailsjs/go/app/App.js';
  import { currentVideo } from '../stores/videoStore.js';
  import { cropClearSignal } from '../stores/projectStore.js';

  export let imageWidth = 0;
  export let imageHeight = 0;
  export let naturalWidth = 0;
  export let naturalHeight = 0;
  export let enabled = false;

  let cropRect = null; // {x, y, width, height} in display pixels
  let isDragging = false;
  let isResizing = false;
  let dragStart = null;
  let resizeHandle = null; // 'nw', 'ne', 'sw', 'se', 'n', 's', 'e', 'w'
  let container;
  let lastClearSignal = 0;

  // Watch for clear signal and reset crop rectangle
  $: if ($cropClearSignal !== lastClearSignal) {
    lastClearSignal = $cropClearSignal;
    cropRect = null;
  }

  function handleMouseDown(e) {
    if (!enabled || !container) return;

    e.preventDefault();
    const rect = container.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    // Check if clicking on existing crop handles
    if (cropRect) {
      const handle = getResizeHandle(x, y);
      if (handle) {
        isResizing = true;
        resizeHandle = handle;
        dragStart = { x, y, rect: { ...cropRect } };
        return;
      }

      // Check if clicking inside crop rect to move it
      if (isInsideCrop(x, y)) {
        isDragging = true;
        dragStart = { x, y, rect: { ...cropRect } };
        return;
      }
    }

    // Start new crop selection
    cropRect = { x, y, width: 0, height: 0 };
    isResizing = true;
    resizeHandle = 'se'; // bottom-right corner
    dragStart = { x, y, rect: { ...cropRect } };
  }

  function handleMouseMove(e) {
    if (!enabled || !container || (!isDragging && !isResizing)) return;

    e.preventDefault();
    const rect = container.getBoundingClientRect();
    const x = Math.max(0, Math.min(imageWidth, e.clientX - rect.left));
    const y = Math.max(0, Math.min(imageHeight, e.clientY - rect.top));

    const dx = x - dragStart.x;
    const dy = y - dragStart.y;

    if (isDragging) {
      // Move the crop rect
      cropRect = {
        x: Math.max(0, Math.min(imageWidth - dragStart.rect.width, dragStart.rect.x + dx)),
        y: Math.max(0, Math.min(imageHeight - dragStart.rect.height, dragStart.rect.y + dy)),
        width: dragStart.rect.width,
        height: dragStart.rect.height
      };
    } else if (isResizing) {
      // Resize the crop rect
      cropRect = calculateResize(dragStart.rect, dx, dy, resizeHandle);
    }
  }

  function handleMouseUp(e) {
    if (isDragging || isResizing) {
      isDragging = false;
      isResizing = false;

      // Apply crop to backend if valid
      if (cropRect && cropRect.width > 10 && cropRect.height > 10) {
        // Convert from display coordinates to natural coordinates
        const scaleX = naturalWidth / imageWidth;
        const scaleY = naturalHeight / imageHeight;

        const x = Math.round(cropRect.x * scaleX);
        const y = Math.round(cropRect.y * scaleY);
        const width = Math.round(cropRect.width * scaleX);
        const height = Math.round(cropRect.height * scaleY);
        SetCropRegion(x, y, width, height);
      } else if (cropRect && (cropRect.width <= 10 || cropRect.height <= 10)) {
        // Too small, clear it
        cropRect = null;
        ClearCropRegion();
      }
    }
  }

  function calculateResize(originalRect, dx, dy, handle) {
    let { x, y, width, height } = originalRect;

    switch (handle) {
      case 'nw': // top-left
        x = Math.max(0, x + dx);
        y = Math.max(0, y + dy);
        width = Math.max(10, width - dx);
        height = Math.max(10, height - dy);
        break;
      case 'ne': // top-right
        y = Math.max(0, y + dy);
        width = Math.max(10, width + dx);
        height = Math.max(10, height - dy);
        break;
      case 'sw': // bottom-left
        x = Math.max(0, x + dx);
        width = Math.max(10, width - dx);
        height = Math.max(10, height + dy);
        break;
      case 'se': // bottom-right
        width = Math.max(10, width + dx);
        height = Math.max(10, height + dy);
        break;
      case 'n': // top edge
        y = Math.max(0, y + dy);
        height = Math.max(10, height - dy);
        break;
      case 's': // bottom edge
        height = Math.max(10, height + dy);
        break;
      case 'e': // right edge
        width = Math.max(10, width + dx);
        break;
      case 'w': // left edge
        x = Math.max(0, x + dx);
        width = Math.max(10, width - dx);
        break;
    }

    // Constrain to image bounds
    if (x + width > imageWidth) width = imageWidth - x;
    if (y + height > imageHeight) height = imageHeight - y;

    return { x, y, width, height };
  }

  function getResizeHandle(mx, my) {
    if (!cropRect) return null;

    const handleSize = 8;
    const { x, y, width, height } = cropRect;

    // Check corners first
    if (isNear(mx, x, handleSize) && isNear(my, y, handleSize)) return 'nw';
    if (isNear(mx, x + width, handleSize) && isNear(my, y, handleSize)) return 'ne';
    if (isNear(mx, x, handleSize) && isNear(my, y + height, handleSize)) return 'sw';
    if (isNear(mx, x + width, handleSize) && isNear(my, y + height, handleSize)) return 'se';

    // Check edges
    if (isNear(my, y, handleSize) && mx > x && mx < x + width) return 'n';
    if (isNear(my, y + height, handleSize) && mx > x && mx < x + width) return 's';
    if (isNear(mx, x, handleSize) && my > y && my < y + height) return 'w';
    if (isNear(mx, x + width, handleSize) && my > y && my < y + height) return 'e';

    return null;
  }

  function isNear(a, b, threshold) {
    return Math.abs(a - b) <= threshold;
  }

  function isInsideCrop(mx, my) {
    if (!cropRect) return false;
    const { x, y, width, height } = cropRect;
    return mx >= x && mx <= x + width && my >= y && my <= y + height;
  }

  function clearCrop() {
    cropRect = null;
    ClearCropRegion();
  }

  // Exported function to clear crop rectangle from parent components
  export function clearCropRectangle() {
    cropRect = null;
  }

  export function getCropRect() {
    return cropRect;
  }

  // Window event handlers that only act when crop mode is enabled
  function handleWindowMouseMove(e) {
    if (!enabled) return;
    handleMouseMove(e);
  }

  function handleWindowMouseUp(e) {
    if (!enabled) return;
    handleMouseUp(e);
  }
</script>

<svelte:window on:mouseup={handleWindowMouseUp} on:mousemove={handleWindowMouseMove} />

<!-- Always show crop rectangle if it exists, but only show hint when enabled -->
{#if (cropRect || enabled) && imageWidth > 0 && imageHeight > 0}
  <div
    class="crop-overlay"
    class:interactive={enabled}
    bind:this={container}
    on:mousedown={enabled ? handleMouseDown : null}
    style="width: {imageWidth}px; height: {imageHeight}px;"
  >
    {#if cropRect}
      <!-- Darkened overlay outside crop region -->
      <svg class="crop-svg" width={imageWidth} height={imageHeight}>
        <defs>
          <mask id="crop-mask">
            <rect x="0" y="0" width={imageWidth} height={imageHeight} fill="white"/>
            <rect x={cropRect.x} y={cropRect.y} width={cropRect.width} height={cropRect.height} fill="black"/>
          </mask>
        </defs>
        <rect x="0" y="0" width={imageWidth} height={imageHeight} fill="rgba(0,0,0,0.5)" mask="url(#crop-mask)"/>
      </svg>

      <!-- Crop rectangle border -->
      <div
        class="crop-rect"
        class:interactive={enabled}
        style="left: {cropRect.x}px; top: {cropRect.y}px; width: {cropRect.width}px; height: {cropRect.height}px;"
      >
        <!-- Corner handles (only interactive when enabled) -->
        <div class="handle nw" class:interactive={enabled}></div>
        <div class="handle ne" class:interactive={enabled}></div>
        <div class="handle sw" class:interactive={enabled}></div>
        <div class="handle se" class:interactive={enabled}></div>

        <!-- Edge handles -->
        <div class="handle n" class:interactive={enabled}></div>
        <div class="handle s" class:interactive={enabled}></div>
        <div class="handle e" class:interactive={enabled}></div>
        <div class="handle w" class:interactive={enabled}></div>

        <!-- Crop dimensions display -->
        <div class="crop-info">
          {Math.round(cropRect.width * (naturalWidth / imageWidth))} × {Math.round(cropRect.height * (naturalHeight / imageHeight))}
        </div>
      </div>
    {/if}
  </div>
{/if}

<style>
  .crop-overlay {
    position: absolute;
    top: 0;
    left: 0;
    z-index: 10;
    pointer-events: none;
  }

  .crop-overlay.interactive {
    cursor: crosshair;
    pointer-events: all;
  }

  .crop-svg {
    position: absolute;
    top: 0;
    left: 0;
    pointer-events: none;
  }

  .crop-rect {
    position: absolute;
    border: 2px solid #00ff00;
    box-sizing: border-box;
    pointer-events: none;
  }

  .crop-rect.interactive {
    border-color: #00ff00;
  }

  .handle {
    position: absolute;
    width: 8px;
    height: 8px;
    background: #00ff00;
    border: 1px solid #000;
    pointer-events: none;
    opacity: 0.5;
  }

  .handle.interactive {
    pointer-events: all;
    opacity: 1;
  }

  .handle.nw { top: -4px; left: -4px; cursor: nw-resize; }
  .handle.ne { top: -4px; right: -4px; cursor: ne-resize; }
  .handle.sw { bottom: -4px; left: -4px; cursor: sw-resize; }
  .handle.se { bottom: -4px; right: -4px; cursor: se-resize; }
  .handle.n { top: -4px; left: 50%; transform: translateX(-50%); cursor: n-resize; }
  .handle.s { bottom: -4px; left: 50%; transform: translateX(-50%); cursor: s-resize; }
  .handle.e { top: 50%; right: -4px; transform: translateY(-50%); cursor: e-resize; }
  .handle.w { top: 50%; left: -4px; transform: translateY(-50%); cursor: w-resize; }

  .crop-info {
    position: absolute;
    bottom: -25px;
    left: 0;
    background: rgba(0, 0, 0, 0.7);
    color: #00ff00;
    padding: 2px 6px;
    font-size: 11px;
    border-radius: 3px;
    pointer-events: none;
    white-space: nowrap;
  }
</style>
