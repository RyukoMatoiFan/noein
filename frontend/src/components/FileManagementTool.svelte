<script>
import { currentVideo } from '../stores/videoStore.js';
import { DeleteVideoFile, MoveVideoToFolder, SelectFolder } from '../../wailsjs/go/app/App.js';

let selectedFolder = '';
let mode = 'move'; // 'move' or 'delete'
let isProcessing = false;
let lastResult = null;

async function handleSelectFolder() {
    try {
        const folder = await SelectFolder();
        if (folder) {
            selectedFolder = folder;
        }
    } catch (error) {
        console.error('Failed to select folder:', error);
        alert('Failed to select folder');
    }
}

async function handleDelete() {
    if (!$currentVideo) {
        alert('No video selected');
        return;
    }

    const confirmed = confirm(`Are you sure you want to delete "${$currentVideo.name}"?\n\nThis action cannot be undone.`);
    if (!confirmed) {
        return;
    }

    isProcessing = true;
    lastResult = null;

    try {
        await DeleteVideoFile($currentVideo.id);
        lastResult = { success: true, message: 'File deleted successfully' };
    } catch (error) {
        console.error('Failed to delete file:', error);
        lastResult = { success: false, message: error.message || 'Failed to delete file' };
    } finally {
        isProcessing = false;
    }
}

async function handleMove() {
    if (!$currentVideo) {
        alert('No video selected');
        return;
    }

    if (!selectedFolder) {
        alert('Please select a destination folder first');
        return;
    }

    isProcessing = true;
    lastResult = null;

    try {
        const newPath = await MoveVideoToFolder($currentVideo.id, selectedFolder);
        lastResult = { success: true, message: `File moved to: ${newPath}` };
    } catch (error) {
        console.error('Failed to move file:', error);
        lastResult = { success: false, message: error.message || 'Failed to move file' };
    } finally {
        isProcessing = false;
    }
}

function handleExecute() {
    if (mode === 'delete') {
        handleDelete();
    } else {
        handleMove();
    }
}

// Auto-clear result after 5 seconds
$: if (lastResult) {
    setTimeout(() => {
        lastResult = null;
    }, 5000);
}
</script>

<div class="file-management">
    <div class="mode-selector">
        <label class="mode-option">
            <input type="radio" bind:group={mode} value="move" />
            <span>Move to Folder</span>
        </label>
        <label class="mode-option">
            <input type="radio" bind:group={mode} value="delete" />
            <span>Delete File</span>
        </label>
    </div>

    {#if mode === 'move'}
        <div class="move-section">
            <button class="folder-select-btn" on:click={handleSelectFolder}>
                {selectedFolder ? 'Change Folder' : 'Select Destination Folder'}
            </button>
            {#if selectedFolder}
                <div class="selected-folder">
                    <div class="folder-label">Destination:</div>
                    <div class="folder-path" title={selectedFolder}>{selectedFolder}</div>
                </div>
            {/if}
        </div>
    {/if}

    {#if mode === 'delete'}
        <div class="warning-box">
            <strong>Warning:</strong> This will permanently delete the current video file from your disk.
        </div>
    {/if}

    {#if $currentVideo}
        <div class="current-file">
            <div class="file-label">Current file:</div>
            <div class="file-name">{$currentVideo.name}</div>
        </div>
    {/if}

    <button
        class="execute-btn"
        class:delete-mode={mode === 'delete'}
        class:move-mode={mode === 'move'}
        disabled={isProcessing || !$currentVideo || (mode === 'move' && !selectedFolder)}
        on:click={handleExecute}
    >
        {#if isProcessing}
            Processing...
        {:else if mode === 'delete'}
            Delete Current File
        {:else}
            Move Current File
        {/if}
    </button>

    {#if lastResult}
        <div class="result-message" class:success={lastResult.success} class:error={!lastResult.success}>
            {lastResult.message}
        </div>
    {/if}
</div>

<style>
.file-management {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.mode-selector {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 10px;
    background: var(--bg-primary);
    border-radius: 4px;
}

.mode-option {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    font-size: 13px;
    color: var(--text-primary);
}

.mode-option input[type="radio"] {
    cursor: pointer;
}

.move-section {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.folder-select-btn {
    padding: 8px 12px;
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s ease;
}

.folder-select-btn:hover {
    background: #333;
    border-color: #555;
}

.selected-folder {
    background: var(--bg-primary);
    padding: 8px;
    border-radius: 4px;
    font-size: 11px;
}

.folder-label {
    color: var(--text-secondary);
    margin-bottom: 4px;
    font-size: 10px;
    text-transform: uppercase;
}

.folder-path {
    color: var(--text-primary);
    word-break: break-all;
    font-family: 'Consolas', 'Monaco', monospace;
}

.warning-box {
    background: rgba(255, 59, 48, 0.1);
    border: 1px solid #ff3b30;
    padding: 10px;
    border-radius: 4px;
    font-size: 11px;
    color: #ff6b6b;
}

.warning-box strong {
    color: #ff3b30;
}

.current-file {
    background: var(--bg-primary);
    padding: 8px;
    border-radius: 4px;
    font-size: 11px;
}

.file-label {
    color: var(--text-secondary);
    margin-bottom: 4px;
    font-size: 10px;
    text-transform: uppercase;
}

.file-name {
    color: var(--text-primary);
    font-weight: 600;
    word-break: break-all;
}

.execute-btn {
    padding: 10px 16px;
    border-radius: 4px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
    border: none;
}

.execute-btn.delete-mode {
    background: #ff3b30;
    color: white;
}

.execute-btn.delete-mode:hover:not(:disabled) {
    background: #ff6b6b;
}

.execute-btn.move-mode {
    background: var(--accent-blue);
    color: white;
}

.execute-btn.move-mode:hover:not(:disabled) {
    background: #50b4eb;
}

.execute-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

.result-message {
    padding: 10px;
    border-radius: 4px;
    font-size: 11px;
    animation: slideIn 0.2s ease;
}

.result-message.success {
    background: rgba(81, 207, 102, 0.15);
    border: 1px solid #51cf66;
    color: #51cf66;
}

.result-message.error {
    background: rgba(255, 59, 48, 0.1);
    border: 1px solid #ff3b30;
    color: #ff6b6b;
}

@keyframes slideIn {
    from {
        opacity: 0;
        transform: translateY(-5px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}
</style>
