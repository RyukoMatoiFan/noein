<script>
  import { currentVideo, currentFrame } from '../stores/videoStore.js';
  import { AddFrameSkipOperation, AddGrayscaleOperation, GetProjectState, GetVideoMetadata } from '../../wailsjs/go/app/App.js';

  let message = '';
  let frameSkipN = 2;

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

  async function applyFrameSkip() {
    if (frameSkipN <= 1) {
      message = 'Error: Frame skip must be greater than 1';
      setTimeout(() => { message = ''; }, 3000);
      return;
    }

    const originalFrames = $currentVideo.totalFrames;
    const estimatedNewFrames = Math.floor(originalFrames / frameSkipN);
    const originalDuration = ($currentVideo.totalFrames / $currentVideo.frameRate).toFixed(2);
    const estimatedDuration = (estimatedNewFrames / $currentVideo.frameRate).toFixed(2);

    try {
      message = 'Processing frame skip...';
      await AddFrameSkipOperation(frameSkipN);
      await reloadVideo();
      message = `✓ Downsampled to every ${frameSkipN} frame(s). Reduced from ${originalFrames} frames (${originalDuration}s) to ~${estimatedNewFrames} frames (~${estimatedDuration}s)`;
      setTimeout(() => { message = ''; }, 6000);
    } catch (error) {
      message = `Error: ${error.message || error}`;
      setTimeout(() => { message = ''; }, 5000);
    }
  }

  async function applyGrayscale() {
    try {
      message = 'Processing grayscale conversion...';
      await AddGrayscaleOperation();
      await reloadVideo();
      message = `✓ Converted to grayscale`;
      setTimeout(() => { message = ''; }, 5000);
    } catch (error) {
      message = `Error: ${error.message || error}`;
      setTimeout(() => { message = ''; }, 5000);
    }
  }
</script>

<div class="frame-ops-tool">
  <div class="section">
    <h3>Frame Downsampling</h3>
    <div class="tool-description">
      <p>Extract every Nth frame to reduce dataset size</p>
    </div>

    <div class="frame-skip-controls">
      <div class="input-row">
        <label>Extract every</label>
        <input
          type="number"
          bind:value={frameSkipN}
          disabled={!$currentVideo}
          min="2"
          max="100"
        />
        <label>frame(s)</label>
      </div>

      <div class="preset-buttons">
        <button class="btn-preset" on:click={() => frameSkipN = 2} disabled={!$currentVideo}>
          Every 2nd
        </button>
        <button class="btn-preset" on:click={() => frameSkipN = 5} disabled={!$currentVideo}>
          Every 5th
        </button>
        <button class="btn-preset" on:click={() => frameSkipN = 10} disabled={!$currentVideo}>
          Every 10th
        </button>
      </div>

      <button class="btn-apply" on:click={applyFrameSkip} disabled={!$currentVideo || frameSkipN <= 1}>
        ⬇ Apply Frame Skip
      </button>
    </div>
  </div>

  <div class="section">
    <h3>Color Conversion</h3>
    <div class="tool-description">
      <p>Convert video to grayscale for simpler models</p>
    </div>

    <div class="grayscale-controls">
      <button class="btn-grayscale" on:click={applyGrayscale} disabled={!$currentVideo}>
        ◐ Convert to Grayscale
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
  .frame-ops-tool {
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

  .frame-skip-controls,
  .grayscale-controls {
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
  }

  .input-row input {
    width: 60px;
    padding: 6px 8px;
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    font-size: 13px;
    font-family: 'Consolas', 'Monaco', monospace;
    text-align: center;
  }

  .input-row input:focus {
    outline: none;
    border-color: var(--accent-blue);
  }

  .input-row input:disabled {
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
    background: #9775fa;
    border: 1px solid #9775fa;
    color: white;
    border-radius: 4px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-apply:hover:not(:disabled) {
    background: #a785ff;
    border-color: #a785ff;
  }

  .btn-apply:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background: #555;
    border-color: #555;
  }

  .btn-grayscale {
    padding: 10px 16px;
    background: #868e96;
    border: 1px solid #868e96;
    color: white;
    border-radius: 4px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-grayscale:hover:not(:disabled) {
    background: #969ea6;
    border-color: #969ea6;
  }

  .btn-grayscale:disabled {
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
