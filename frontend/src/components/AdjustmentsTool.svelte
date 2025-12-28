<script>
  import { currentVideo, currentFrame } from '../stores/videoStore.js';
  import { AddFPSChangeOperation, AddBrightnessContrastOperation, AddRemoveAudioOperation, GetProjectState, GetVideoMetadata } from '../../wailsjs/go/app/App.js';

  let message = '';
  let targetFPS = 30;
  let brightness = 0;
  let contrast = 0;
  let lastVideoId = null;

  // Initialize FPS with current video frame rate only when video changes
  $: if ($currentVideo && $currentVideo.id !== lastVideoId) {
    targetFPS = $currentVideo.frameRate;
    lastVideoId = $currentVideo.id;
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

  async function applyFPSChange() {
    if (targetFPS <= 0 || targetFPS > 240) {
      message = 'Error: FPS must be between 0 and 240';
      setTimeout(() => { message = ''; }, 3000);
      return;
    }

    const originalFPS = $currentVideo.frameRate.toFixed(2);

    try {
      message = 'Processing FPS change...';
      await AddFPSChangeOperation(targetFPS);
      await reloadVideo();
      message = `✓ Changed frame rate from ${originalFPS} fps to ${targetFPS} fps`;
      setTimeout(() => { message = ''; }, 5000);
    } catch (error) {
      message = `Error: ${error.message || error}`;
      setTimeout(() => { message = ''; }, 5000);
    }
  }

  async function applyBrightnessContrast() {
    if (brightness < -1 || brightness > 1) {
      message = 'Error: Brightness must be between -1.0 and 1.0';
      setTimeout(() => { message = ''; }, 3000);
      return;
    }

    if (contrast < -1 || contrast > 1) {
      message = 'Error: Contrast must be between -1.0 and 1.0';
      setTimeout(() => { message = ''; }, 3000);
      return;
    }

    try {
      message = 'Processing brightness/contrast...';
      await AddBrightnessContrastOperation(brightness, contrast);
      await reloadVideo();
      message = `✓ Adjusted brightness: ${brightness >= 0 ? '+' : ''}${brightness.toFixed(2)}, contrast: ${contrast >= 0 ? '+' : ''}${contrast.toFixed(2)}`;
      setTimeout(() => { message = ''; }, 5000);
    } catch (error) {
      message = `Error: ${error.message || error}`;
      setTimeout(() => { message = ''; }, 5000);
    }
  }

  async function applyRemoveAudio() {
    try {
      message = 'Removing audio track...';
      await AddRemoveAudioOperation();
      await reloadVideo();
      message = `✓ Removed audio track`;
      setTimeout(() => { message = ''; }, 5000);
    } catch (error) {
      message = `Error: ${error.message || error}`;
      setTimeout(() => { message = ''; }, 5000);
    }
  }

  function resetBrightnessContrast() {
    brightness = 0;
    contrast = 0;
  }
</script>

<div class="adjustments-tool">
  <div class="section">
    <h3>Frame Rate Conversion</h3>
    <div class="tool-description">
      <p>Change video frame rate (affects playback speed)</p>
    </div>

    <div class="fps-controls">
      <div class="input-row">
        <label>Target FPS:</label>
        <input
          type="number"
          bind:value={targetFPS}
          disabled={!$currentVideo}
          min="1"
          max="240"
          step="0.01"
        />
      </div>

      <div class="preset-buttons">
        <button class="btn-preset" on:click={() => targetFPS = 24} disabled={!$currentVideo}>
          24 fps
        </button>
        <button class="btn-preset" on:click={() => targetFPS = 30} disabled={!$currentVideo}>
          30 fps
        </button>
        <button class="btn-preset" on:click={() => targetFPS = 60} disabled={!$currentVideo}>
          60 fps
        </button>
      </div>

      <button class="btn-apply" on:click={applyFPSChange} disabled={!$currentVideo}>
        ⏱ Apply FPS Change
      </button>
    </div>
  </div>

  <div class="section">
    <h3>Brightness & Contrast</h3>
    <div class="tool-description">
      <p>Adjust brightness and contrast (-1.0 to +1.0)</p>
    </div>

    <div class="brightness-controls">
      <div class="slider-row">
        <label>Brightness: {brightness >= 0 ? '+' : ''}{brightness.toFixed(2)}</label>
        <input
          type="range"
          bind:value={brightness}
          disabled={!$currentVideo}
          min="-1"
          max="1"
          step="0.05"
        />
      </div>

      <div class="slider-row">
        <label>Contrast: {contrast >= 0 ? '+' : ''}{contrast.toFixed(2)}</label>
        <input
          type="range"
          bind:value={contrast}
          disabled={!$currentVideo}
          min="-1"
          max="1"
          step="0.05"
        />
      </div>

      <div class="button-row">
        <button class="btn-reset" on:click={resetBrightnessContrast} disabled={!$currentVideo}>
          Reset
        </button>
        <button class="btn-apply" on:click={applyBrightnessContrast} disabled={!$currentVideo}>
          ☀ Apply Adjustments
        </button>
      </div>
    </div>
  </div>

  <div class="section">
    <h3>Audio Operations</h3>
    <div class="tool-description">
      <p>Remove audio track to reduce file size</p>
    </div>

    <div class="audio-controls">
      <button class="btn-remove-audio" on:click={applyRemoveAudio} disabled={!$currentVideo}>
        🔇 Remove Audio Track
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
  .adjustments-tool {
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

  .fps-controls,
  .brightness-controls,
  .audio-controls {
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
    min-width: 80px;
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

  .slider-row {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .slider-row label {
    font-size: 11px;
    color: var(--text-primary);
    font-family: 'Consolas', 'Monaco', monospace;
  }

  .slider-row input[type="range"] {
    width: 100%;
    height: 6px;
    background: var(--bg-secondary);
    border-radius: 3px;
    outline: none;
    -webkit-appearance: none;
  }

  .slider-row input[type="range"]::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: 16px;
    height: 16px;
    background: var(--accent-blue);
    border-radius: 50%;
    cursor: pointer;
  }

  .slider-row input[type="range"]::-moz-range-thumb {
    width: 16px;
    height: 16px;
    background: var(--accent-blue);
    border-radius: 50%;
    cursor: pointer;
    border: none;
  }

  .slider-row input[type="range"]:disabled {
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

  .button-row {
    display: flex;
    gap: 8px;
  }

  .btn-reset {
    padding: 8px 12px;
    background: var(--bg-secondary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-reset:hover:not(:disabled) {
    background: #333;
    border-color: #555;
  }

  .btn-reset:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-apply {
    flex: 1;
    padding: 10px 16px;
    background: #ff922b;
    border: 1px solid #ff922b;
    color: white;
    border-radius: 4px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-apply:hover:not(:disabled) {
    background: #ffa23b;
    border-color: #ffa23b;
  }

  .btn-apply:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background: #555;
    border-color: #555;
  }

  .btn-remove-audio {
    padding: 10px 16px;
    background: #fa5252;
    border: 1px solid #fa5252;
    color: white;
    border-radius: 4px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-remove-audio:hover:not(:disabled) {
    background: #ff6262;
    border-color: #ff6262;
  }

  .btn-remove-audio:disabled {
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
