<script>
  import { currentVideo, currentFrame } from '../stores/videoStore.js';
  import { cropEnabled, cropClearSignal } from '../stores/projectStore.js';
  import { ClearCropRegion, AddCropOperation, GetProjectState, GetVideoMetadata } from '../../wailsjs/go/app/App.js';

  let isActive = false;
  let message = '';

  $: isActive = $cropEnabled;

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

  function toggleCrop() {
    if (isActive) {
      // Disable crop mode and clear crop region
      cropEnabled.set(false);
      ClearCropRegion();
      isActive = false;
    } else {
      // Enable crop mode
      cropEnabled.set(true);
      isActive = true;
    }
  }

  function clearCrop() {
    ClearCropRegion();
  }

  async function addCrop() {
    try {
      // Get crop details before applying
      const projectState = await GetProjectState();
      const crop = projectState.currentCrop;
      const originalRes = `${$currentVideo.width}x${$currentVideo.height}`;
      const newRes = crop ? `${crop.width}x${crop.height}` : 'unknown';

      message = 'Processing crop...';
      await AddCropOperation();

      // Reload the video to show the cropped version
      await reloadVideo();

      // Clear the crop rectangle from the UI
      await ClearCropRegion();
      // Signal to CropSelector to clear its rectangle
      cropClearSignal.update(n => n + 1);

      message = `✓ Cropped from ${originalRes} to ${newRes} at position (${crop.x}, ${crop.y})`;
      // Disable crop mode after adding
      cropEnabled.set(false);
      isActive = false;

      setTimeout(() => {
        message = '';
      }, 5000);
    } catch (error) {
      message = `Error: ${error.message || error}`;
    }
  }
</script>

<div class="crop-tool">
  <div class="tool-description">
    <p>Select a rectangular region of the video to crop. The crop will be applied when exporting segments.</p>
  </div>

  <div class="control-group">
    <button
      class="btn-primary"
      class:active={isActive}
      on:click={toggleCrop}
      disabled={!$currentVideo}
      title={isActive ? "Disable crop mode" : "Enable crop mode"}
    >
      {isActive ? '✓ Crop Mode Active' : 'Enable Crop Mode'}
    </button>
  </div>

  {#if isActive}
    <div class="crop-instructions">
      <h4>How to use:</h4>
      <ol>
        <li>Click and drag on the video to draw a crop rectangle</li>
        <li>Drag the rectangle to reposition it</li>
        <li>Drag the corner/edge handles to resize</li>
        <li>Click "Add Crop" to add this crop operation</li>
      </ol>

      <div class="crop-buttons">
        <button class="btn-add-crop" on:click={addCrop}>
          ⊡ Add Crop to Edit Stack
        </button>

        <button class="btn-secondary" on:click={clearCrop}>
          Clear Crop Region
        </button>
      </div>
    </div>
  {/if}

  {#if message}
    <div class="message" class:success={message.includes('added')} class:error={message.includes('Error')}>
      {message}
    </div>
  {/if}
</div>

<style>
  .crop-tool {
    padding: 16px;
  }

  .tool-description {
    margin-bottom: 16px;
    font-size: 12px;
    color: var(--text-secondary);
    line-height: 1.5;
  }

  .tool-description p {
    margin: 0;
  }

  .control-group {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .btn-primary {
    padding: 10px 16px;
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-primary:hover:not(:disabled) {
    background: #333;
    border-color: #555;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary.active {
    background: #2a5;
    border-color: #2a5;
    color: white;
  }

  .btn-primary.active:hover {
    background: #3b6;
    border-color: #3b6;
  }

  .btn-secondary {
    padding: 8px 14px;
    background: var(--bg-secondary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-secondary:hover {
    background: #333;
    border-color: #555;
  }

  .crop-instructions {
    margin-top: 16px;
    padding: 12px;
    background: var(--bg-secondary);
    border-radius: 4px;
    font-size: 11px;
  }

  .crop-instructions h4 {
    margin: 0 0 8px 0;
    font-size: 11px;
    text-transform: uppercase;
    color: var(--text-secondary);
    letter-spacing: 0.5px;
  }

  .crop-instructions ol {
    margin: 0 0 12px 0;
    padding-left: 20px;
    color: var(--text-secondary);
    line-height: 1.6;
  }

  .crop-instructions li {
    margin-bottom: 4px;
  }

  .crop-buttons {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .btn-add-crop {
    padding: 10px 16px;
    background: #51cf66;
    border: 1px solid #51cf66;
    color: white;
    border-radius: 4px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-add-crop:hover {
    background: #61df76;
    border-color: #61df76;
  }

  .message {
    margin-top: 12px;
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
