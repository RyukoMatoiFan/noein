<script>
  import { onMount, onDestroy } from 'svelte';
  import { UndoLastEdit, ClearEditStack, SaveToEditedFolder, GetProjectState, GetVideoMetadata, ApplyEditStackToVideos, GetFrame } from '../../wailsjs/go/app/App.js';
  import { currentVideo, currentFrame } from '../stores/videoStore.js';
  import { selectedVideos, skipFrameReset, frameBeforeEdits } from '../stores/projectStore.js';

  export let expanded = false; // Receive expanded state from parent

  let editStack = [];
  let saving = false;
  let batchSaving = false;
  let saveMessage = '';
  let refreshInterval = null;
  let applyToAllSelected = false; // Checkbox state for batch mode

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

  async function refreshEditStack() {
    try {
      const projectState = await GetProjectState();
      editStack = projectState.editStack || [];
    } catch (error) {
      console.error('Failed to load edit stack:', error);
    }
  }

  async function handleUndo() {
    try {
      await UndoLastEdit();
      await refreshEditStack();
      // Reload video to show previous state
      await reloadVideo();
    } catch (error) {
      console.error('Failed to undo:', error);
      alert(error);
    }
  }

  async function handleClearAll() {
    if (!confirm('Clear all edits? This cannot be undone.')) {
      return;
    }

    try {
      await ClearEditStack();
      await refreshEditStack();
      // Reload video to show original
      await reloadVideo();
    } catch (error) {
      console.error('Failed to clear edits:', error);
      alert(error);
    }
  }

  async function handleSave() {
    if (editStack.length === 0) {
      return;
    }

    saving = true;
    saveMessage = `Saving edited video...`;

    try {
      const outputPath = await SaveToEditedFolder();
      saveMessage = `Saved to: ${outputPath}`;

      // Reload video to show original after save and restore to pre-edit frame position
      const projectState = await GetProjectState();
      if (projectState.currentVideoId) {
        const video = await GetVideoMetadata(projectState.currentVideoId);

        // Use saved frame position from before edits, or current frame if no edits were made
        const targetFrame = $frameBeforeEdits !== null ? $frameBeforeEdits : $currentFrame;
        const clampedFrame = Math.min(targetFrame, video.totalFrames - 1);

        // Set flag to preserve frame position
        skipFrameReset.set(true);

        // Update video - reactive statement will preserve frame position
        currentVideo.set(video);

        // Explicitly set the frame to the target position
        currentFrame.set(clampedFrame);

        // Load the frame image at that position
        try {
          await GetFrame(projectState.currentVideoId, clampedFrame);
        } catch (error) {
          console.error('Failed to load frame after save:', error);
        }

        // Clear the saved frame position and reset flag
        frameBeforeEdits.set(null);
        setTimeout(() => {
          skipFrameReset.set(false);
        }, 200);
      }

      await refreshEditStack();

      setTimeout(() => {
        saveMessage = '';
        saving = false;
      }, 5000);
    } catch (error) {
      console.error('Failed to save:', error);
      saveMessage = `Error: ${error}`;
      saving = false;

      setTimeout(() => {
        saveMessage = '';
      }, 5000);
    }
  }

  async function handleBatchSave() {
    if (editStack.length === 0) {
      alert('No edits to apply');
      return;
    }

    const selectedArray = [...$selectedVideos];
    if (selectedArray.length === 0) {
      alert('No videos selected');
      return;
    }

    if (!confirm(`Apply all ${editStack.length} edit operation(s) to ${selectedArray.length} selected video(s)?`)) {
      return;
    }

    batchSaving = true;
    saveMessage = `Processing ${selectedArray.length} video(s)...`;

    try {
      const results = await ApplyEditStackToVideos(selectedArray);
      const successCount = results ? results.filter(r => r.success).length : 0;
      const failCount = results ? results.filter(r => !r.success).length : 0;

      if (failCount > 0) {
        saveMessage = `Processed: ${successCount} succeeded, ${failCount} failed`;
      } else {
        saveMessage = `Successfully processed ${successCount} video(s)`;
      }

      // Clear selection after successful batch processing
      selectedVideos.set(new Set());

      setTimeout(() => {
        saveMessage = '';
        batchSaving = false;
      }, 5000);
    } catch (error) {
      console.error('Failed to batch process:', error);
      saveMessage = `Error: ${error.message || error}`;
      batchSaving = false;

      setTimeout(() => {
        saveMessage = '';
      }, 5000);
    }
  }

  function getOperationIcon(type) {
    switch (type) {
      case 'trim_external':
        return '✂️';
      case 'trim_internal':
        return '⌧';
      case 'crop':
        return '⊡';
      case 'scale':
        return '⇲';
      case 'rotate':
        return '↻';
      case 'grayscale':
        return '◐';
      case 'frame_skip':
        return '⬇';
      case 'fps_change':
        return '⏱';
      case 'brightness_contrast':
        return '☀';
      case 'remove_audio':
        return '🔇';
      case 'speed_change':
        return '⚡';
      case 'add_padding':
        return '▭';
      case 'trim_duration':
        return '⏱';
      case 'format_conversion':
        return '🔄';
      default:
        return '•';
    }
  }

  function getOperationColor(type) {
    switch (type) {
      case 'trim_external':
        return '#4a9eff';
      case 'trim_internal':
        return '#ff6b6b';
      case 'crop':
        return '#51cf66';
      case 'scale':
        return '#4a9eff';
      case 'rotate':
        return '#ff922b';
      case 'grayscale':
        return '#868e96';
      case 'frame_skip':
        return '#9775fa';
      case 'fps_change':
        return '#ff922b';
      case 'brightness_contrast':
        return '#ffd43b';
      case 'remove_audio':
        return '#fa5252';
      case 'speed_change':
        return '#845ef7';
      case 'add_padding':
        return '#339af0';
      case 'trim_duration':
        return '#20c997';
      case 'format_conversion':
        return '#cc5de8';
      default:
        return '#999';
    }
  }

  onMount(() => {
    // Load initial state
    refreshEditStack();
  });

  onDestroy(() => {
    // Clean up interval when component is destroyed
    if (refreshInterval) {
      clearInterval(refreshInterval);
      refreshInterval = null;
    }
  });

  // Only poll when the panel is expanded
  $: if (expanded) {
    // Start refreshing every 3 seconds (increased from 2)
    if (!refreshInterval) {
      refreshEditStack(); // Immediate refresh
      refreshInterval = setInterval(refreshEditStack, 3000);
    }
  } else {
    // Stop refreshing when collapsed
    if (refreshInterval) {
      clearInterval(refreshInterval);
      refreshInterval = null;
    }
  }
</script>

<div class="edit-history">
  <div class="header">
    <h3>Edit Operations ({editStack.length})</h3>
    {#if editStack.length > 0}
      <button class="btn-clear" on:click={handleClearAll} title="Clear all edits">
        Clear All
      </button>
    {/if}
  </div>

  {#if editStack.length === 0}
    <div class="empty-state">
      <p>No edit operations yet</p>
      <p class="hint">Add trim or crop operations to build your edit</p>
    </div>
  {:else}
    <div class="operations-list">
      {#each editStack as operation, index}
        <div class="operation-item" style="border-left-color: {getOperationColor(operation.type)}">
          <div class="operation-header">
            <span class="operation-icon">{getOperationIcon(operation.type)}</span>
            <span class="operation-number">#{index + 1}</span>
          </div>
          <div class="operation-description">
            {operation.description}
          </div>
        </div>
      {/each}
    </div>

    <div class="actions">
      <button
        class="btn-undo"
        on:click={handleUndo}
        disabled={editStack.length === 0}
        title="Undo last operation"
      >
        ↶ Undo Last
      </button>

      {#if $selectedVideos.size > 0 && editStack.length > 0}
        <label class="batch-checkbox">
          <input type="checkbox" bind:checked={applyToAllSelected} />
          <span>Apply to all {$selectedVideos.size} selected video{$selectedVideos.size === 1 ? '' : 's'}</span>
        </label>
      {/if}

      <button
        class="btn-save"
        on:click={applyToAllSelected && $selectedVideos.size > 0 ? handleBatchSave : handleSave}
        disabled={!$currentVideo || editStack.length === 0 || saving || batchSaving}
      >
        {#if saving || batchSaving}
          {applyToAllSelected ? `Processing ${$selectedVideos.size} videos...` : 'Saving...'}
        {:else if applyToAllSelected && $selectedVideos.size > 0}
          📦 Apply to All Selected
        {:else}
          💾 Save to "edited" folder
        {/if}
      </button>
    </div>

    {#if saveMessage}
      <div class="save-message" class:error={saveMessage.startsWith('Error')}>
        {saveMessage}
      </div>
    {/if}
  {/if}
</div>

<style>
  .edit-history {
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  .header h3 {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .btn-clear {
    padding: 4px 10px;
    background: transparent;
    color: var(--text-secondary);
    border: 1px solid var(--border-color);
    border-radius: 3px;
    font-size: 11px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-clear:hover {
    background: #d33;
    border-color: #d33;
    color: white;
  }

  .empty-state {
    text-align: center;
    padding: 32px 16px;
    color: var(--text-secondary);
  }

  .empty-state p {
    margin: 0 0 8px 0;
    font-size: 12px;
  }

  .empty-state .hint {
    font-size: 11px;
    opacity: 0.7;
  }

  .operations-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 300px;
    overflow-y: auto;
  }

  .operation-item {
    background: var(--bg-secondary);
    border-left: 3px solid #999;
    border-radius: 4px;
    padding: 10px 12px;
    transition: all 0.15s ease;
  }

  .operation-item:hover {
    background: #2a2a2a;
  }

  .operation-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 6px;
  }

  .operation-icon {
    font-size: 14px;
  }

  .operation-number {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
    font-family: 'Consolas', 'Monaco', monospace;
  }

  .operation-description {
    font-size: 12px;
    color: var(--text-primary);
    line-height: 1.4;
  }

  .actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: 8px;
  }

  .btn-undo,
  .btn-save {
    padding: 10px 16px;
    border-radius: 4px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-undo {
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
  }

  .btn-undo:hover:not(:disabled) {
    background: #333;
    border-color: #555;
  }

  .btn-undo:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-save {
    background: #2a5;
    border: 1px solid #2a5;
    color: white;
  }

  .btn-save:hover:not(:disabled) {
    background: #3b6;
    border-color: #3b6;
  }

  .btn-save:disabled {
    opacity: 0.5;
    cursor: not-allowed;
    background: #555;
    border-color: #555;
  }

  .batch-checkbox {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background: rgba(81, 207, 102, 0.1);
    border: 1px solid #51cf66;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .batch-checkbox:hover {
    background: rgba(81, 207, 102, 0.15);
  }

  .batch-checkbox input[type="checkbox"] {
    cursor: pointer;
    width: 16px;
    height: 16px;
    accent-color: #51cf66;
  }

  .batch-checkbox span {
    font-size: 12px;
    color: #51cf66;
    font-weight: 600;
  }

  .batch-actions {
    margin-top: 8px;
  }

  .btn-batch {
    width: 100%;
    padding: 12px 16px;
    background: #51cf66;
    border: 1px solid #51cf66;
    border-radius: 4px;
    font-size: 13px;
    font-weight: 600;
    color: white;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .btn-batch:hover:not(:disabled) {
    background: #61df76;
    border-color: #61df76;
  }

  .btn-batch:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .save-message {
    padding: 10px 12px;
    background: rgba(42, 170, 85, 0.2);
    border: 1px solid #2a5;
    border-radius: 4px;
    font-size: 11px;
    color: #2a5;
    line-height: 1.4;
    word-break: break-all;
  }

  .save-message.error {
    background: rgba(255, 107, 107, 0.2);
    border-color: #ff6b6b;
    color: #ff6b6b;
  }
</style>
