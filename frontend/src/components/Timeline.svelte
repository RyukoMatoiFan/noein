<script>
import { createEventDispatcher } from 'svelte';
import { currentVideo, currentFrame } from '../stores/videoStore.js';
import { inPoint, outPoint } from '../stores/projectStore.js';

const dispatch = createEventDispatcher();

let timelineElement;
let isDragging = false;

function handleMouseDown(e) {
    isDragging = true;
    dispatch('scrubStart');
    updateFrameFromMouse(e);
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
}

function handleMouseMove(e) {
    if (!isDragging) return;
    updateFrameFromMouse(e);
}

function handleMouseUp() {
    if (isDragging) {
        isDragging = false;
        dispatch('scrubEnd');
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', handleMouseUp);
    }
}

function updateFrameFromMouse(e) {
    if (!timelineElement || !$currentVideo) return;

    const rect = timelineElement.getBoundingClientRect();
    const x = Math.max(0, Math.min(e.clientX - rect.left, rect.width));
    const percentage = x / rect.width;
    const frame = Math.round(percentage * ($currentVideo.totalFrames - 1));

    dispatch('frameChange', frame);
}

$: progress = $currentVideo ? ($currentFrame / $currentVideo.totalFrames) * 100 : 0;
$: inPointPercent = $currentVideo && $inPoint !== null ? ($inPoint / $currentVideo.totalFrames) * 100 : null;
$: outPointPercent = $currentVideo && $outPoint !== null ? ($outPoint / $currentVideo.totalFrames) * 100 : null;
</script>

<div class="timeline-container">
    <div
        class="timeline"
        bind:this={timelineElement}
        on:mousedown={handleMouseDown}
    >
        <!-- In/Out markers -->
        {#if inPointPercent !== null}
            <div class="marker marker-in" style="left: {inPointPercent}%">
                <div class="marker-label">IN</div>
            </div>
        {/if}
        {#if outPointPercent !== null}
            <div class="marker marker-out" style="left: {outPointPercent}%">
                <div class="marker-label">OUT</div>
            </div>
        {/if}

        <!-- Progress bar -->
        <div class="progress" style="width: {progress}%" />

        <!-- Playhead -->
        <div class="playhead" style="left: {progress}%" />
    </div>
</div>

<style>
.timeline-container {
    padding: 12px;
    background: var(--bg-secondary);
}

.timeline {
    position: relative;
    height: 40px;
    background: #333;
    border-radius: 4px;
    cursor: pointer;
    overflow: visible;
}

.marker {
    position: absolute;
    width: 2px;
    height: 100%;
    top: 0;
    pointer-events: none;
}

.marker-in {
    background: var(--accent-green);
}

.marker-out {
    background: var(--accent-red);
}

.marker-label {
    position: absolute;
    top: -20px;
    left: -12px;
    font-size: 10px;
    font-weight: 600;
    background: inherit;
    padding: 2px 6px;
    border-radius: 2px;
    white-space: nowrap;
}

.progress {
    position: absolute;
    height: 100%;
    background: rgba(255, 255, 255, 0.08);
    pointer-events: none;
}

.playhead {
    position: absolute;
    width: 2px;
    height: 120%;
    top: -10%;
    background: var(--text-primary);
    pointer-events: none;
    box-shadow: 0 0 8px rgba(255, 255, 255, 0.5);
}

.playhead::before {
    content: '';
    position: absolute;
    top: -4px;
    left: -4px;
    width: 10px;
    height: 10px;
    background: var(--text-primary);
    border-radius: 50%;
}
</style>
