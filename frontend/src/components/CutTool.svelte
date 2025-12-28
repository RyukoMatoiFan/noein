<script>
import { currentVideo, currentFrame } from '../stores/videoStore.js';
import { inPoint, outPoint, hasSelection } from '../stores/projectStore.js';
import { AddTrimExternal, AddTrimInternal, SetInPoint, SetOutPoint, ClearMarks, GetProjectState, GetVideoMetadata } from '../../wailsjs/go/app/App.js';

let message = '';

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

async function setIn() {
    if (!$currentVideo) return;
    await SetInPoint($currentFrame);
    inPoint.set($currentFrame);
}

async function setOut() {
    if (!$currentVideo) return;
    await SetOutPoint($currentFrame);
    outPoint.set($currentFrame);
}

async function clearPoints() {
    await ClearMarks();
    inPoint.set(null);
    outPoint.set(null);
    message = '';
}

async function addExternal() {
    if (!$currentVideo || $inPoint === null || $outPoint === null) {
        message = 'Set in/out points first';
        return;
    }

    if ($inPoint >= $outPoint) {
        message = 'In point must be before out point';
        return;
    }

    const originalFrames = $currentVideo.totalFrames;
    const keptFrames = $outPoint - $inPoint;
    const originalDuration = ($currentVideo.totalFrames / $currentVideo.frameRate).toFixed(2);
    const newDuration = (keptFrames / $currentVideo.frameRate).toFixed(2);

    try {
        message = 'Processing trim...';
        await AddTrimExternal();

        // Reload the video to show the edited version
        await reloadVideo();

        message = `✓ Kept frames ${$inPoint}-${$outPoint} (${keptFrames} frames, ${newDuration}s). Removed ${originalFrames - keptFrames} frames.`;
        // Clear points after adding
        inPoint.set(null);
        outPoint.set(null);

        setTimeout(() => {
            message = '';
        }, 5000);
    } catch (error) {
        message = `Error: ${error.message || error}`;
    }
}

async function addInternal() {
    if (!$currentVideo || $inPoint === null || $outPoint === null) {
        message = 'Set in/out points first';
        return;
    }

    if ($inPoint >= $outPoint) {
        message = 'In point must be before out point';
        return;
    }

    const originalFrames = $currentVideo.totalFrames;
    const removedFrames = $outPoint - $inPoint;
    const removedDuration = (removedFrames / $currentVideo.frameRate).toFixed(2);

    try {
        message = 'Processing cut...';
        await AddTrimInternal();

        // Reload the video to show the edited version
        await reloadVideo();

        message = `✓ Removed frames ${$inPoint}-${$outPoint} (${removedFrames} frames, ${removedDuration}s). Now ${originalFrames - removedFrames} frames total.`;
        // Clear points after adding
        inPoint.set(null);
        outPoint.set(null);

        setTimeout(() => {
            message = '';
        }, 5000);
    } catch (error) {
        message = `Error: ${error.message || error}`;
    }
}

$: selectionDuration = $hasSelection && $currentVideo
    ? (($outPoint - $inPoint) / $currentVideo.frameRate).toFixed(3)
    : '0.000';

$: selectionFrames = $hasSelection ? $outPoint - $inPoint : 0;
</script>

<div class="cut-tool">
    <div class="section">
        <h3>Mark Points</h3>
        <div class="button-group">
            <button class="btn btn-primary" on:click={setIn} disabled={!$currentVideo}>
                Set In Point [I]
            </button>
            <button class="btn btn-primary" on:click={setOut} disabled={!$currentVideo}>
                Set Out Point [O]
            </button>
            <button class="btn btn-secondary" on:click={clearPoints} disabled={!$hasSelection}>
                Clear Marks
            </button>
        </div>
    </div>

    {#if $hasSelection}
        <div class="section">
            <h3>Selection</h3>
            <div class="selection-info">
                <div class="info-row">
                    <span class="label">In:</span>
                    <span class="value">Frame {$inPoint}</span>
                </div>
                <div class="info-row">
                    <span class="label">Out:</span>
                    <span class="value">Frame {$outPoint}</span>
                </div>
                <div class="info-row">
                    <span class="label">Duration:</span>
                    <span class="value">{selectionDuration}s ({selectionFrames} frames)</span>
                </div>
            </div>
        </div>

        <div class="section">
            <h3>Trim Selection</h3>
            <div class="trim-info">
                <p><strong>Keep Selection:</strong> Keep only frames {$inPoint}-{$outPoint}, discard the rest</p>
                <p><strong>Cut Out Selection:</strong> Remove frames {$inPoint}-{$outPoint}, keep the rest</p>
            </div>
            <div class="button-group">
                <button class="btn btn-trim-external" on:click={addExternal} title="Keep only the selected frames">
                    ✂️ Keep Selection (IN-OUT)
                </button>
                <button class="btn btn-trim-internal" on:click={addInternal} title="Remove the selected frames">
                    ⌧ Cut Out Selection (Remove IN-OUT)
                </button>
            </div>
        </div>
    {/if}

    {#if message}
        <p class="message" class:success={message.includes('added')} class:error={message.includes('Error')}>
            {message}
        </p>
    {/if}
</div>

<style>
.cut-tool {
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

.button-group {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.btn {
    padding: 10px 14px;
    border-radius: 4px;
    font-size: 13px;
    font-weight: 500;
    transition: all 0.2s ease;
}

.btn-primary {
    background: var(--accent-green);
    color: white;
}

.btn-primary:hover:not(:disabled) {
    background: #45a049;
}

.btn-secondary {
    background: var(--bg-primary);
    color: var(--text-primary);
}

.btn-secondary:hover:not(:disabled) {
    background: #333;
}

.btn-trim-external {
    width: 100%;
    background: #4a9eff;
    color: white;
    font-weight: 600;
}

.btn-trim-external:hover:not(:disabled) {
    background: #5ab0ff;
}

.btn-trim-internal {
    width: 100%;
    background: #ff6b6b;
    color: white;
    font-weight: 600;
}

.btn-trim-internal:hover:not(:disabled) {
    background: #ff7b7b;
}

.trim-info {
    background: rgba(0, 0, 0, 0.3);
    padding: 10px 12px;
    border-radius: 4px;
    border: 1px solid var(--border-color);
    margin-bottom: 12px;
}

.trim-info p {
    font-size: 11px;
    color: var(--text-secondary);
    margin: 4px 0;
    line-height: 1.5;
}

.trim-info strong {
    color: var(--text-primary);
}

.selection-info {
    background: rgba(0, 0, 0, 0.3);
    padding: 12px;
    border-radius: 4px;
    border: 1px solid var(--border-color);
}

.info-row {
    display: flex;
    justify-content: space-between;
    padding: 4px 0;
    font-size: 12px;
    font-family: 'Consolas', 'Monaco', monospace;
}

.info-row .label {
    color: var(--text-secondary);
}

.info-row .value {
    color: var(--text-primary);
    font-weight: 500;
}

.progress-message {
    font-size: 11px;
    color: var(--text-secondary);
    margin-top: 8px;
    text-align: center;
}

.message {
    font-size: 12px;
    padding: 8px 12px;
    border-radius: 4px;
    text-align: center;
    margin-top: 12px;
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
