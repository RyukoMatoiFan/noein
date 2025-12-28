<script>
  import { currentVideo, currentFrame } from '../stores/videoStore.js';
  import { AddSpeedChangeOperation, AddPaddingOperation, AddTrimDurationOperation, GetProjectState, GetVideoMetadata } from '../../wailsjs/go/app/App.js';

  let message = '';
  let speedFactor = 1.0;
  let paddingWidth = 1920;
  let paddingHeight = 1080;
  let paddingColor = 'black';
  let trimDuration = 10;

  // Initialize padding with current video dimensions
  $: if ($currentVideo) {
    paddingWidth = $currentVideo.width;
    paddingHeight = $currentVideo.height;
    trimDuration = Math.min(10, $currentVideo.duration);
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

  async function applySpeedChange() {
    if (speedFactor <= 0 || speedFactor > 10) {
      message = 'Error: Speed must be between 0 and 10';
      setTimeout(() => { message = ''; }, 3000);
      return;
    }

    const originalDuration = $currentVideo.duration.toFixed(2);
    const newDuration = ($currentVideo.duration / speedFactor).toFixed(2);

    try {
      message = 'Processing speed change...';
      await AddSpeedChangeOperation(speedFactor);
      await reloadVideo();
      message = `✓ Changed speed to ${speedFactor}x. Duration: ${originalDuration}s → ${newDuration}s`;
      setTimeout(() => { message = ''; }, 5000);
    } catch (error) {
      message = `Error: ${error.message || error}`;
      setTimeout(() => { message = ''; }, 5000);
    }
  }

  async function applyPadding() {
    if (paddingWidth <= 0 || paddingHeight <= 0) {
      message = 'Error: Padding dimensions must be positive';
      setTimeout(() => { message = ''; }, 3000);
      return;
    }

    const originalRes = `${$currentVideo.width}x${$currentVideo.height}`;
    const newRes = `${paddingWidth}x${paddingHeight}`;

    try {
      message = 'Processing padding...';
      await AddPaddingOperation(paddingWidth, paddingHeight, paddingColor);
      await reloadVideo();
      message = `✓ Added ${paddingColor} padding from ${originalRes} to ${newRes}`;
      setTimeout(() => { message = ''; }, 5000);
    } catch (error) {
      message = `Error: ${error.message || error}`;
      setTimeout(() => { message = ''; }, 5000);
    }
  }

  async function applyTrimDuration() {
    if (trimDuration <= 0) {
      message = 'Error: Duration must be positive';
      setTimeout(() => { message = ''; }, 3000);
      return;
    }

    const originalDuration = $currentVideo.duration.toFixed(2);

    try {
      message = 'Processing duration trim...';
      await AddTrimDurationOperation(trimDuration);
      await reloadVideo();
      message = `✓ Kept first ${trimDuration.toFixed(2)}s of ${originalDuration}s video`;
      setTimeout(() => { message = ''; }, 5000);
    } catch (error) {
      message = `Error: ${error.message || error}`;
      setTimeout(() => { message = ''; }, 5000);
    }
  }
</script>

<div class="advanced-tool">
  <div class="section">
    <h3>Speed Adjustment</h3>
    <div class="tool-description">
      <p>Change playback speed (affects duration)</p>
    </div>

    <div class="speed-controls">
      <div class="input-row">
        <label>Speed:</label>
        <input
          type="number"
          bind:value={speedFactor}
          disabled={!$currentVideo}
          min="0.1"
          max="10"
          step="0.1"
        />
        <span class="unit">x</span>
      </div>

      <div class="preset-buttons">
        <button class="btn-preset" on:click={() => speedFactor = 0.5} disabled={!$currentVideo}>
          0.5x
        </button>
        <button class="btn-preset" on:click={() => speedFactor = 1.0} disabled={!$currentVideo}>
          1.0x
        </button>
        <button class="btn-preset" on:click={() => speedFactor = 2.0} disabled={!$currentVideo}>
          2.0x
        </button>
      </div>

      <button class="btn-apply" on:click={applySpeedChange} disabled={!$currentVideo}>
        ⚡ Apply Speed Change
      </button>
    </div>
  </div>

  <div class="section">
    <h3>Add Padding / Letterbox</h3>
    <div class="tool-description">
      <p>Add black/white bars to match aspect ratio</p>
    </div>

    <div class="padding-controls">
      <div class="input-row">
        <input
          type="number"
          bind:value={paddingWidth}
          disabled={!$currentVideo}
          placeholder="Width"
          min="1"
        />
        <span class="separator">×</span>
        <input
          type="number"
          bind:value={paddingHeight}
          disabled={!$currentVideo}
          placeholder="Height"
          min="1"
        />
      </div>

      <div class="color-row">
        <label>Color:</label>
        <select bind:value={paddingColor} disabled={!$currentVideo}>
          <option value="black">Black</option>
          <option value="white">White</option>
          <option value="gray">Gray</option>
        </select>
      </div>

      <button class="btn-apply" on:click={applyPadding} disabled={!$currentVideo}>
        ▭ Apply Padding
      </button>
    </div>
  </div>

  <div class="section">
    <h3>Trim by Duration</h3>
    <div class="tool-description">
      <p>Keep first N seconds of video</p>
    </div>

    <div class="duration-controls">
      <div class="input-row">
        <label>Duration:</label>
        <input
          type="number"
          bind:value={trimDuration}
          disabled={!$currentVideo}
          min="0.1"
          step="0.5"
        />
        <span class="unit">seconds</span>
      </div>

      <div class="preset-buttons">
        <button class="btn-preset" on:click={() => trimDuration = 5} disabled={!$currentVideo}>
          5s
        </button>
        <button class="btn-preset" on:click={() => trimDuration = 10} disabled={!$currentVideo}>
          10s
        </button>
        <button class="btn-preset" on:click={() => trimDuration = 30} disabled={!$currentVideo}>
          30s
        </button>
      </div>

      <button class="btn-apply" on:click={applyTrimDuration} disabled={!$currentVideo}>
        ⏱ Apply Duration Trim
      </button>
    </div>
  </div>

  {#if message}
    <div class="message" class:success={message.includes('✓')} class:error={message.includes('Error')}>
      {message}
    </div>
  {/if}
</div>

<style>
  .advanced-tool {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .section h3 {
    font-size: 12px;
    text-transform: uppercase;
    color: var(--text-secondary);
    margin: 0 0 8px 0;
    letter-spacing: 0.5px;
  }

  .tool-description {
    margin-bottom: 12px;
    font-size: 11px;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .tool-description p {
    margin: 0;
  }

  .speed-controls,
  .padding-controls,
  .duration-controls {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .input-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .input-row label {
    font-size: 12px;
    color: var(--text-primary);
    min-width: 60px;
  }

  .input-row input[type="number"] {
    flex: 1;
    padding: 6px 8px;
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

  .separator,
  .unit {
    color: var(--text-secondary);
    font-size: 12px;
  }

  .color-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .color-row label {
    font-size: 12px;
    color: var(--text-primary);
    min-width: 60px;
  }

  .color-row select {
    flex: 1;
    padding: 6px 8px;
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    font-size: 13px;
    cursor: pointer;
  }

  .color-row select:focus {
    outline: none;
    border-color: var(--accent-blue);
  }

  .color-row select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
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
    background: #845ef7;
    border: 1px solid #845ef7;
    color: white;
    border-radius: 4px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-apply:hover:not(:disabled) {
    background: #946eff;
    border-color: #946eff;
  }

  .btn-apply:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background: #555;
    border-color: #555;
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
