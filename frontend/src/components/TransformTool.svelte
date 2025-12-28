<script>
  import { currentVideo, currentFrame } from '../stores/videoStore.js';
  import { AddScaleOperation, AddRotateOperation, GetProjectState, GetVideoMetadata } from '../../wailsjs/go/app/App.js';

  let message = '';
  let scaleWidth = 0;
  let scaleHeight = 0;

  // Initialize scale inputs with current video dimensions
  $: if ($currentVideo) {
    scaleWidth = $currentVideo.width;
    scaleHeight = $currentVideo.height;
  }

  async function reloadVideo() {
    try {
      const projectState = await GetProjectState();
      if (projectState.currentVideoId) {
        const video = await GetVideoMetadata(projectState.currentVideoId);
        currentVideo.set(video);
        currentFrame.set(0);
      }
    } catch (error) {
      console.error('Failed to reload video:', error);
    }
  }

  async function applyScale() {
    if (scaleWidth <= 0 || scaleHeight <= 0) {
      message = 'Error: Width and height must be positive';
      setTimeout(() => { message = ''; }, 3000);
      return;
    }

    const originalRes = `${$currentVideo.width}x${$currentVideo.height}`;
    const newRes = `${scaleWidth}x${scaleHeight}`;

    try {
      message = 'Processing scale...';
      await AddScaleOperation(scaleWidth, scaleHeight);
      await reloadVideo();
      message = `✓ Scaled from ${originalRes} to ${newRes}`;
      setTimeout(() => { message = ''; }, 5000);
    } catch (error) {
      message = `Error: ${error.message || error}`;
      setTimeout(() => { message = ''; }, 5000);
    }
  }

  async function applyRotate(rotateType) {
    const rotateDescriptions = {
      '90': '90° clockwise',
      '180': '180°',
      '270': '270° clockwise',
      'hflip': 'horizontally',
      'vflip': 'vertically'
    };

    try {
      message = 'Processing rotation...';
      await AddRotateOperation(rotateType);
      await reloadVideo();
      message = `✓ Rotated/flipped ${rotateDescriptions[rotateType]}`;
      setTimeout(() => { message = ''; }, 5000);
    } catch (error) {
      message = `Error: ${error.message || error}`;
      setTimeout(() => { message = ''; }, 5000);
    }
  }

  function setCommonResolution(width, height) {
    scaleWidth = width;
    scaleHeight = height;
  }
</script>

<div class="transform-tool">
  <div class="section">
    <h3>Resolution Scaling</h3>
    <div class="scale-controls">
      <div class="input-row">
        <input
          type="number"
          bind:value={scaleWidth}
          disabled={!$currentVideo}
          placeholder="Width"
          min="1"
        />
        <span class="separator">×</span>
        <input
          type="number"
          bind:value={scaleHeight}
          disabled={!$currentVideo}
          placeholder="Height"
          min="1"
        />
      </div>

      <div class="preset-buttons">
        <button class="btn-preset" on:click={() => setCommonResolution(1920, 1080)} disabled={!$currentVideo}>
          1080p
        </button>
        <button class="btn-preset" on:click={() => setCommonResolution(1280, 720)} disabled={!$currentVideo}>
          720p
        </button>
        <button class="btn-preset" on:click={() => setCommonResolution(640, 480)} disabled={!$currentVideo}>
          480p
        </button>
      </div>

      <button class="btn-apply" on:click={applyScale} disabled={!$currentVideo}>
        ⇲ Apply Scale
      </button>
    </div>
  </div>

  <div class="section">
    <h3>Rotation & Flip</h3>
    <div class="rotate-controls">
      <div class="rotate-buttons">
        <button class="btn-rotate" on:click={() => applyRotate('90')} disabled={!$currentVideo} title="Rotate 90° clockwise">
          ↻ 90°
        </button>
        <button class="btn-rotate" on:click={() => applyRotate('180')} disabled={!$currentVideo} title="Rotate 180°">
          ↻ 180°
        </button>
        <button class="btn-rotate" on:click={() => applyRotate('270')} disabled={!$currentVideo} title="Rotate 270° clockwise">
          ↻ 270°
        </button>
      </div>

      <div class="flip-buttons">
        <button class="btn-flip" on:click={() => applyRotate('hflip')} disabled={!$currentVideo} title="Flip horizontally">
          ↔ Flip H
        </button>
        <button class="btn-flip" on:click={() => applyRotate('vflip')} disabled={!$currentVideo} title="Flip vertically">
          ↕ Flip V
        </button>
      </div>
    </div>
  </div>

  {#if message}
    <div class="message" class:success={message.includes('✓')} class:error={message.includes('Error')}>
      {message}
    </div>
  {/if}
</div>

<style>
  .transform-tool {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .section h3 {
    font-size: 12px;
    text-transform: uppercase;
    color: var(--text-secondary);
    margin: 0 0 10px 0;
    letter-spacing: 0.5px;
  }

  .scale-controls {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .input-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .input-row input {
    flex: 1;
    padding: 8px;
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    font-size: 13px;
    font-family: 'Consolas', 'Monaco', monospace;
  }

  .input-row input:focus {
    outline: none;
    border-color: var(--accent-blue);
  }

  .input-row input:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .separator {
    color: var(--text-secondary);
    font-weight: 600;
  }

  .preset-buttons {
    display: flex;
    gap: 4px;
  }

  .btn-preset {
    flex: 1;
    padding: 6px;
    background: var(--bg-secondary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 3px;
    font-size: 11px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-preset:hover:not(:disabled) {
    background: #333;
    border-color: #555;
  }

  .btn-preset:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-apply {
    padding: 10px 16px;
    background: #4a9eff;
    border: 1px solid #4a9eff;
    color: white;
    border-radius: 4px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-apply:hover:not(:disabled) {
    background: #5ab0ff;
    border-color: #5ab0ff;
  }

  .btn-apply:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background: #555;
    border-color: #555;
  }

  .rotate-controls {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .rotate-buttons,
  .flip-buttons {
    display: flex;
    gap: 4px;
  }

  .btn-rotate,
  .btn-flip {
    flex: 1;
    padding: 8px 12px;
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-rotate:hover:not(:disabled),
  .btn-flip:hover:not(:disabled) {
    background: #333;
    border-color: #555;
  }

  .btn-rotate:disabled,
  .btn-flip:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .message {
    padding: 10px 12px;
    border-radius: 4px;
    font-size: 11px;
    line-height: 1.4;
  }

  .message.success {
    background: rgba(81, 207, 102, 0.2);
    border: 1px solid #51cf66;
    color: #51cf66;
  }

  .message.error {
    background: rgba(255, 107, 107, 0.2);
    border: 1px solid #ff6b6b;
    color: #ff6b6b;
  }
</style>
