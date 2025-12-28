<script>
import { currentVideo } from '../stores/videoStore.js';
import { SaveCurrentMark, DeleteMark, ExportAllMarks, GetProjectState, SelectOutputDirectory, SetCurrentVideo } from '../../wailsjs/go/app/App.js';
import { inPoint, outPoint, hasSelection } from '../stores/projectStore.js';

let marks = [];
let markDescription = '';
let isExporting = false;
let exportMessage = '';

async function refreshMarks() {
    try {
        const state = await GetProjectState();
        marks = state.marks || [];
    } catch (error) {
        console.error('Failed to refresh marks:', error);
    }
}

async function handleSaveMark() {
    if (!$hasSelection) return;

    try {
        await SaveCurrentMark(markDescription || `Segment ${marks.length + 1}`);
        markDescription = '';
        await refreshMarks();

        // Clear the current in/out points
        inPoint.set(null);
        outPoint.set(null);
    } catch (error) {
        console.error('Failed to save mark:', error);
        exportMessage = `Error: ${error.message || error}`;
    }
}

async function handleDeleteMark(markId) {
    try {
        await DeleteMark(markId);
        await refreshMarks();
    } catch (error) {
        console.error('Failed to delete mark:', error);
    }
}

async function handleBatchExport() {
    if (marks.length === 0) {
        exportMessage = 'No marks to export';
        return;
    }

    isExporting = true;
    exportMessage = '';

    try {
        const outputDir = await SelectOutputDirectory();
        if (!outputDir) {
            exportMessage = 'Export cancelled';
            isExporting = false;
            return;
        }

        exportMessage = `Exporting ${marks.length} segments...`;

        await ExportAllMarks(outputDir);

        exportMessage = `Successfully exported ${marks.length} segments!`;

        // Clear marks after successful export
        setTimeout(() => {
            exportMessage = '';
        }, 3000);
    } catch (error) {
        exportMessage = `Error: ${error.message || error}`;
    } finally {
        isExporting = false;
    }
}

function formatDuration(inFrame, outFrame, frameRate) {
    const duration = (outFrame - inFrame) / frameRate;
    return duration.toFixed(2) + 's';
}

// Refresh marks when component mounts and when video changes
$: if ($currentVideo) {
    SetCurrentVideo($currentVideo.id);
    refreshMarks();
}
</script>

<div class="marks-list">
    <div class="section">
        <h3>Save Current Mark</h3>

        {#if $hasSelection}
            <div class="save-mark-form">
                <input
                    type="text"
                    placeholder="Description (optional)"
                    bind:value={markDescription}
                    class="mark-input"
                />
                <button class="btn btn-primary" on:click={handleSaveMark}>
                    Save Mark
                </button>
            </div>
        {:else}
            <p class="hint">Set in/out points first</p>
        {/if}
    </div>

    <div class="section">
        <h3>Saved Marks ({marks.length})</h3>

        {#if marks.length === 0}
            <p class="hint">No saved marks yet</p>
        {:else}
            <div class="marks-items">
                {#each marks as mark, index (mark.id)}
                    <div class="mark-item">
                        <div class="mark-header">
                            <span class="mark-number">#{index + 1}</span>
                            <button class="btn-delete" on:click={() => handleDeleteMark(mark.id)} title="Delete">
                                ×
                            </button>
                        </div>
                        <div class="mark-info">
                            {#if mark.description}
                                <div class="mark-description">{mark.description}</div>
                            {/if}
                            <div class="mark-details">
                                <span>In: {mark.inFrame}</span>
                                <span>Out: {mark.outFrame}</span>
                                <span>{formatDuration(mark.inFrame, mark.outFrame, $currentVideo?.frameRate || 30)}</span>
                            </div>
                        </div>
                    </div>
                {/each}
            </div>

            <button
                class="btn btn-export"
                on:click={handleBatchExport}
                disabled={isExporting}
            >
                {isExporting ? 'Exporting...' : `Export All ${marks.length} Segments`}
            </button>
        {/if}

        {#if exportMessage}
            <p class="message" class:success={exportMessage.includes('Success')} class:error={exportMessage.includes('Error')}>
                {exportMessage}
            </p>
        {/if}
    </div>
</div>

<style>
.marks-list {
    display: flex;
    flex-direction: column;
    gap: 20px;
}

.section h3 {
    font-size: 12px;
    text-transform: uppercase;
    color: var(--text-secondary);
    margin-bottom: 10px;
    letter-spacing: 0.5px;
}

.hint {
    font-size: 12px;
    color: var(--text-secondary);
    font-style: italic;
}

.save-mark-form {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.mark-input {
    padding: 8px 12px;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    color: var(--text-primary);
    font-size: 13px;
}

.mark-input::placeholder {
    color: var(--text-secondary);
    opacity: 0.6;
}

.btn {
    padding: 10px 14px;
    border-radius: 4px;
    font-size: 13px;
    font-weight: 500;
    transition: all 0.2s ease;
    border: none;
    cursor: pointer;
}

.btn-primary {
    background: var(--accent-green);
    color: white;
}

.btn-primary:hover {
    background: #45a049;
}

.btn-export {
    width: 100%;
    background: var(--accent-blue);
    color: var(--bg-primary);
    font-weight: 600;
    padding: 12px;
    margin-top: 12px;
}

.btn-export:hover:not(:disabled) {
    background: #50b4eb;
}

.btn-export:disabled {
    opacity: 0.5;
    cursor: not-allowed;
}

.marks-items {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 300px;
    overflow-y: auto;
    margin-bottom: 8px;
}

.mark-item {
    background: rgba(0, 0, 0, 0.3);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    padding: 10px;
}

.mark-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 6px;
}

.mark-number {
    font-weight: 600;
    color: var(--accent-blue);
    font-size: 12px;
}

.btn-delete {
    background: none;
    border: none;
    color: var(--accent-red);
    font-size: 20px;
    line-height: 1;
    cursor: pointer;
    padding: 0;
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 2px;
    transition: background 0.2s ease;
}

.btn-delete:hover {
    background: rgba(244, 67, 54, 0.2);
}

.mark-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.mark-description {
    font-size: 13px;
    color: var(--text-primary);
    margin-bottom: 4px;
}

.mark-details {
    display: flex;
    gap: 12px;
    font-size: 11px;
    color: var(--text-secondary);
    font-family: 'Consolas', 'Monaco', monospace;
}

.message {
    font-size: 12px;
    padding: 8px 12px;
    border-radius: 4px;
    text-align: center;
    margin-top: 8px;
}

.message.success {
    background: rgba(76, 175, 80, 0.2);
    color: var(--accent-green);
}

.message.error {
    background: rgba(244, 67, 54, 0.2);
    color: var(--accent-red);
}
</style>
