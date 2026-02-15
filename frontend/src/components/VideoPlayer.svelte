<script>
import { onMount, onDestroy } from 'svelte';
import Timeline from './Timeline.svelte';
import CropSelector from './CropSelector.svelte';
import { currentVideo, currentFrame, isPlaying } from '../stores/videoStore.js';
import { inPoint, outPoint, cropEnabled, cropClearSignal, skipFrameReset, frameBeforeEdits } from '../stores/projectStore.js';
import { SetInPoint, SetOutPoint, GetFrame, GetVideoURL, AddTrimExternal, AddTrimInternal, AddCropOperation, SaveToEditedFolder, UndoLastEdit, SetCurrentVideo, GetProjectState, GetVideoMetadata } from '../../wailsjs/go/app/App.js';

let currentFrameImage = null;
let hoverFrameImage = null;
let isHovering = false;
let isScrubbing = false;
let playbackInterval = null;
let videoElement; // HTML5 video element for smooth playback
let imageElement;
let imageWidth = 0;
let imageHeight = 0;
let displayWidth = 0;
let displayHeight = 0;
let cachedDisplayWidth = 0; // Cache for crop overlay during playback
let cachedDisplayHeight = 0; // Cache for crop overlay during playback
let isMousePressed = false;
let framePreviewNotification = '';
let previousVideoId = null; // Track when video actually changes
let useVideoPlayback = true; // Use HTML5 video for smooth playback
let isMuted = true; // Sound off by default
let notificationMessage = '';
let notificationTimeout = null;
let requestedFrame = null;

async function handleFrameChange(event) {
    const newFrame = event.detail;
    currentFrame.set(newFrame);

    // Load current frame image
    await loadCurrentFrame(newFrame);
}

async function loadCurrentFrame(frameNum) {
    if (!$currentVideo) return;
    requestedFrame = frameNum;

    try {
        const frame = await GetFrame($currentVideo.id, frameNum);
        currentFrameImage = frame.imageData;
    } catch (error) {
        console.error('Failed to load frame:', error);
    }
}

$: if ($currentVideo && !$isPlaying && !isScrubbing && $currentFrame !== requestedFrame) {
    loadCurrentFrame($currentFrame);
    if (videoElement && useVideoPlayback) {
        videoElement.currentTime = $currentFrame / $currentVideo.frameRate;
    }
}

async function handleVideoHover(event) {
    // Only show hover preview when video is paused AND mouse button is pressed
    if ($isPlaying || isScrubbing || !$currentVideo || !isMousePressed) {
        isHovering = false;
        hoverFrameImage = null;
        framePreviewNotification = '';
        return;
    }

    const viewport = event.currentTarget;
    const rect = viewport.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const halfWidth = rect.width / 2;

    isHovering = true;

    // Left half = previous frame, Right half = next frame
    const isLeft = x < halfWidth;
    const targetFrame = isLeft
        ? Math.max(0, $currentFrame - 1)
        : Math.min($currentVideo.totalFrames - 1, $currentFrame + 1);

    // Only load if different from current
    if (targetFrame !== $currentFrame) {
        try {
            const frame = await GetFrame($currentVideo.id, targetFrame);
            hoverFrameImage = frame.imageData;
            // Show notification
            framePreviewNotification = isLeft
                ? `← Previous Frame (${targetFrame})`
                : `Next Frame (${targetFrame}) →`;
        } catch (error) {
            console.error('Failed to load hover frame:', error);
        }
    } else {
        hoverFrameImage = null;
        framePreviewNotification = '';
    }
}

function handleMouseDown(event) {
    if ($isPlaying || isScrubbing || !$currentVideo) return;
    isMousePressed = true;
    handleVideoHover(event);
}

function handleMouseUp() {
    isMousePressed = false;
    isHovering = false;
    hoverFrameImage = null;
    framePreviewNotification = '';
}

function handleVideoLeave() {
    isHovering = false;
    hoverFrameImage = null;
}

function handleScrubStart() {
    isScrubbing = true;
    stopPlayback();
}

function handleScrubEnd() {
    isScrubbing = false;
}

function togglePlayback() {
    if ($isPlaying) {
        stopPlayback();
    } else {
        startPlayback();
    }
}

async function startPlayback() {
    if (!$currentVideo || $isPlaying) return;

    // Disable crop mode when starting playback
    if ($cropEnabled) {
        cropEnabled.set(false);
    }

    // Use HTML5 video for smooth playback (hardware accelerated)
    if (videoElement && useVideoPlayback) {
        try {
            // Always reload video URL (ensures correct video after switching)
            const url = await GetVideoURL($currentVideo.id);

            // Force reload by clearing and setting src
            videoElement.src = '';
            videoElement.load(); // Reset video element state
            videoElement.src = url;

            // Wait for video to be ready
            await new Promise((resolve, reject) => {
                if (videoElement.readyState >= 2) { // HAVE_CURRENT_DATA
                    resolve();
                } else {
                    const onCanPlay = () => {
                        cleanup();
                        resolve();
                    };
                    const onError = (e) => {
                        cleanup();
                        reject(new Error('Failed to load video: ' + e.message));
                    };
                    const cleanup = () => {
                        videoElement.removeEventListener('canplay', onCanPlay);
                        videoElement.removeEventListener('error', onError);
                    };
                    videoElement.addEventListener('canplay', onCanPlay, { once: true });
                    videoElement.addEventListener('error', onError, { once: true });
                }
            });

            // Set video time to current frame position
            const startTime = $currentFrame / $currentVideo.frameRate;
            videoElement.currentTime = startTime;

            // Start playback
            await videoElement.play();
            isPlaying.set(true);

            // Update frame counter during playback
            playbackInterval = setInterval(() => {
                if (videoElement && $isPlaying) {
                    const currentFrameNum = Math.floor(videoElement.currentTime * $currentVideo.frameRate);
                    currentFrame.set(currentFrameNum);

                    // Stop at end
                    if (currentFrameNum >= $currentVideo.totalFrames - 1) {
                        stopPlayback();
                    }
                }
            }, 1000 / 30); // Update 30 times per second (smooth UI)
        } catch (err) {
            console.error('Video playback failed:', err);
            isPlaying.set(false);
        }
    }
}

function stopPlayback() {
    isPlaying.set(false);

    // Pause HTML5 video if playing
    if (videoElement && useVideoPlayback) {
        videoElement.pause();
        // Extract current frame when stopping (for frame-perfect display)
        const finalFrame = $currentFrame;
        loadCurrentFrame(finalFrame);
    }

    if (playbackInterval) {
        clearInterval(playbackInterval);
        playbackInterval = null;
    }

    // Re-enable crop mode if it was active before playback
    // (Check if we have cached dimensions, which indicates crop was used)
    if (cachedDisplayWidth > 0 && cachedDisplayHeight > 0) {
        cropEnabled.set(true);
    }
}

// Watch isPlaying store and stop playback if it becomes false externally
$: if (!$isPlaying && playbackInterval) {
    clearInterval(playbackInterval);
    playbackInterval = null;
}

async function handleSetInPoint() {
    if ($currentVideo) {
        await SetInPoint($currentFrame);
        inPoint.set($currentFrame);
    }
}

async function handleSetOutPoint() {
    if ($currentVideo) {
        await SetOutPoint($currentFrame);
        outPoint.set($currentFrame);
    }
}

async function nextFrame() {
    if ($currentVideo && $currentFrame < $currentVideo.totalFrames - 1) {
        const newFrame = $currentFrame + 1;
        currentFrame.set(newFrame);
        await loadCurrentFrame(newFrame);
    }
}

async function prevFrame() {
    if ($currentVideo && $currentFrame > 0) {
        const newFrame = $currentFrame - 1;
        currentFrame.set(newFrame);
        await loadCurrentFrame(newFrame);
    }
}

async function skip10Forward() {
    if ($currentVideo) {
        const newFrame = Math.min($currentVideo.totalFrames - 1, $currentFrame + 10);
        currentFrame.set(newFrame);
        await loadCurrentFrame(newFrame);
    }
}

async function skip10Backward() {
    if ($currentVideo) {
        const newFrame = Math.max(0, $currentFrame - 10);
        currentFrame.set(newFrame);
        await loadCurrentFrame(newFrame);
    }
}

async function moveToStart() {
    if ($currentVideo) {
        currentFrame.set(0);
        await loadCurrentFrame(0);
    }
}

async function moveToEnd() {
    if ($currentVideo) {
        const lastFrame = $currentVideo.totalFrames - 1;
        currentFrame.set(lastFrame);
        await loadCurrentFrame(lastFrame);
    }
}

function toggleMute() {
    isMuted = !isMuted;
}

function showNotification(message, duration = 3000) {
    notificationMessage = message;
    if (notificationTimeout) {
        clearTimeout(notificationTimeout);
    }
    notificationTimeout = setTimeout(() => {
        notificationMessage = '';
    }, duration);
}

async function quickSetIn() {
    if (!$currentVideo) return;
    await SetInPoint($currentFrame);
    inPoint.set($currentFrame);
}

async function quickSetOut() {
    if (!$currentVideo) return;
    await SetOutPoint($currentFrame);
    outPoint.set($currentFrame);
}

async function reloadCurrentVideo() {
    try {
        // Get project state to find the current video ID (may have changed after operations)
        const projectState = await GetProjectState();
        if (projectState.currentVideoId) {
            // Get the updated video metadata
            const video = await GetVideoMetadata(projectState.currentVideoId);
            currentVideo.set(video);

            // Reset to first frame to show the edited result
            currentFrame.set(0);

            // Load the first frame image to display
            await loadCurrentFrame(0);

            // Force reload video element with new video URL
            if (videoElement) {
                const videoUrl = await GetVideoURL(projectState.currentVideoId);
                videoElement.src = '';
                videoElement.load();
                videoElement.src = videoUrl;
                videoElement.load();
            }
        }
    } catch (error) {
        console.error('Failed to reload video:', error);
        showNotification('Error reloading video');
    }
}

async function quickKeep() {
    if ($inPoint === null || $outPoint === null) {
        showNotification('Please set IN and OUT points first');
        return;
    }
    try {
        // Remember frame position before first edit
        if ($frameBeforeEdits === null) {
            frameBeforeEdits.set($currentFrame);
        }
        await AddTrimExternal();
        await reloadCurrentVideo();
        // Clear trim marks after operation
        inPoint.set(null);
        outPoint.set(null);
        showNotification('Section kept successfully');
    } catch (error) {
        showNotification(`Failed to keep section: ${error.message || error}`);
    }
}

async function quickCutout() {
    if ($inPoint === null || $outPoint === null) {
        showNotification('Please set IN and OUT points first');
        return;
    }
    try {
        // Remember frame position before first edit
        if ($frameBeforeEdits === null) {
            frameBeforeEdits.set($currentFrame);
        }
        await AddTrimInternal();
        await reloadCurrentVideo();
        // Clear trim marks after operation
        inPoint.set(null);
        outPoint.set(null);
        showNotification('Section cut out successfully');
    } catch (error) {
        showNotification(`Failed to cut out section: ${error.message || error}`);
    }
}

async function quickCrop() {
    try {
        // Remember frame position before first edit
        if ($frameBeforeEdits === null) {
            frameBeforeEdits.set($currentFrame);
        }
        await AddCropOperation();
        await reloadCurrentVideo();
        cropEnabled.set(false);
        // Clear the crop rectangle
        cropClearSignal.update(n => n + 1);
        showNotification('Crop applied successfully');
    } catch (error) {
        showNotification(`Failed to crop: ${error.message || error}`);
    }
}

async function quickSave() {
    try {
        const outputPath = await SaveToEditedFolder();

        // Reload video (will be original after save) and restore to pre-edit frame position
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
            await loadCurrentFrame(clampedFrame);

            // Reload video element with new video URL
            if (videoElement) {
                const videoUrl = await GetVideoURL(projectState.currentVideoId);
                videoElement.src = '';
                videoElement.load();
                videoElement.src = videoUrl;
                videoElement.load();
            }

            // Clear the saved frame position and reset flag
            frameBeforeEdits.set(null);
            setTimeout(() => {
                skipFrameReset.set(false);
            }, 200);
        }

        showNotification(`Saved to: ${outputPath}`, 5000);
    } catch (error) {
        showNotification(`Failed to save: ${error.message || error}`);
    }
}

async function quickUndo() {
    try {
        await UndoLastEdit();
        await reloadCurrentVideo();
        showNotification('Undo successful');
    } catch (error) {
        showNotification(`Failed to undo: ${error.message || error}`);
    }
}

function handleKeydown(e) {
    if (!$currentVideo) return;

    switch (e.key) {
        case ' ':
            e.preventDefault();
            togglePlayback();
            break;
        case 'ArrowLeft':
            e.preventDefault();
            if (e.shiftKey) {
                skip10Backward();
            } else {
                prevFrame();
            }
            break;
        case 'ArrowRight':
            e.preventDefault();
            if (e.shiftKey) {
                skip10Forward();
            } else {
                nextFrame();
            }
            break;
        case 'Home':
            e.preventDefault();
            moveToStart();
            break;
        case 'End':
            e.preventDefault();
            moveToEnd();
            break;
        case 'i':
        case 'I':
            e.preventDefault();
            handleSetInPoint();
            break;
        case 'o':
        case 'O':
            e.preventDefault();
            handleSetOutPoint();
            break;
        case 'm':
        case 'M':
            e.preventDefault();
            toggleMute();
            break;
    }
}

function updateImageDimensions() {
    if (imageElement) {
        imageWidth = imageElement.naturalWidth;
        imageHeight = imageElement.naturalHeight;
        // Get the actual displayed size
        const newDisplayWidth = imageElement.clientWidth;
        const newDisplayHeight = imageElement.clientHeight;

        // Always cache the latest valid dimensions
        if (newDisplayWidth > 0 && newDisplayHeight > 0) {
            cachedDisplayWidth = newDisplayWidth;
            cachedDisplayHeight = newDisplayHeight;
        }

        // Only update display dimensions if not playing, or if crop mode is enabled
        if (!$isPlaying || $cropEnabled) {
            displayWidth = newDisplayWidth;
            displayHeight = newDisplayHeight;
        }
    }
}

onMount(() => {
    window.addEventListener('keydown', handleKeydown);
    window.addEventListener('mouseup', handleMouseUp);
    window.addEventListener('resize', updateImageDimensions);
});

onDestroy(() => {
    window.removeEventListener('keydown', handleKeydown);
    window.removeEventListener('mouseup', handleMouseUp);
    window.removeEventListener('resize', updateImageDimensions);
    stopPlayback();
});

// Load first frame when video changes (only when video ID changes)
$: if ($currentVideo && $currentVideo.id !== previousVideoId) {
    // Stop any ongoing playback
    if (playbackInterval) {
        clearInterval(playbackInterval);
        playbackInterval = null;
    }
    if (videoElement) {
        videoElement.pause();
        videoElement.src = ''; // Clear src to free memory
    }
    isPlaying.set(false);

    // Only reset to frame 0 if skipFrameReset is false
    // Otherwise preserve current frame (clamped to new video length)
    if (!$skipFrameReset) {
        currentFrame.set(0);
        loadCurrentFrame(0);
    } else {
        // Preserve frame position, clamped to new video length
        const clampedFrame = Math.min($currentFrame, $currentVideo.totalFrames - 1);
        if (clampedFrame !== $currentFrame) {
            currentFrame.set(clampedFrame);
        }
        loadCurrentFrame(clampedFrame);
    }

    previousVideoId = $currentVideo.id;
}

$: timecode = $currentVideo ? formatTimecode($currentFrame, $currentVideo.frameRate) : '00:00:00:00';

function formatTimecode(frame, frameRate) {
    const totalSeconds = frame / frameRate;
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = Math.floor(totalSeconds % 60);
    const frames = frame % Math.round(frameRate);

    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}:${frames.toString().padStart(2, '0')}`;
}
</script>

<div class="video-player">
    <div
        class="video-viewport"
        on:mousedown={!$cropEnabled && !$isPlaying ? handleMouseDown : null}
        on:mousemove={!$cropEnabled && !$isPlaying ? handleVideoHover : null}
        on:mouseleave={handleVideoLeave}
    >
        {#if $currentVideo}
            <div class="image-container">
                <!-- HTML5 video element for smooth playback (always in DOM, hidden when not playing) -->
                <video
                    bind:this={videoElement}
                    bind:muted={isMuted}
                    class="frame-video"
                    class:hidden={!$isPlaying || !useVideoPlayback}
                    on:timeupdate={() => {
                        if (videoElement && $isPlaying) {
                            const frameNum = Math.floor(videoElement.currentTime * $currentVideo.frameRate);
                            currentFrame.set(frameNum);
                        }
                    }}
                    on:ended={stopPlayback}
                ></video>

                <!-- Image element for frame-perfect display when paused -->
                {#if hoverFrameImage || currentFrameImage}
                    <img
                        bind:this={imageElement}
                        src={hoverFrameImage || currentFrameImage}
                        alt="Frame {$currentFrame}"
                        class="frame-image"
                        class:hidden={$isPlaying && useVideoPlayback}
                        class:hover-preview={isHovering && hoverFrameImage}
                        on:load={updateImageDimensions}
                    />
                {/if}

                <CropSelector
                    enabled={$cropEnabled && !$isPlaying}
                    imageWidth={displayWidth || 0}
                    imageHeight={displayHeight || 0}
                    naturalWidth={imageWidth || 0}
                    naturalHeight={imageHeight || 0}
                />

                {#if framePreviewNotification}
                    <div class="frame-preview-notification">
                        {framePreviewNotification}
                    </div>
                {/if}
            </div>
        {:else if $currentVideo}
            <div class="loading">
                <p>Loading video...</p>
            </div>
        {:else}
            <div class="no-video">
                <div class="placeholder-icon">▶</div>
                <p>Select a video to begin</p>
            </div>
        {/if}
    </div>

    <div class="controls">
        <button class="control-btn" on:click={togglePlayback} disabled={!$currentVideo}>
            {$isPlaying ? '⏸' : '▶'}
        </button>

        <button class="control-btn" on:click={toggleMute} disabled={!$currentVideo} title="Toggle Mute (M)">
            {isMuted ? '🔇' : '🔊'}
        </button>

        <div class="frame-controls">
            <button class="control-btn small" on:click={moveToStart} disabled={!$currentVideo} title="Jump to Start (Home)">
                ⏮
            </button>
            <button class="control-btn small" on:click={skip10Backward} disabled={!$currentVideo} title="Skip 10 Frames Back (Shift+←)">
                ◀◀
            </button>
            <button class="control-btn small" on:click={prevFrame} disabled={!$currentVideo} title="Previous Frame (←)">
                ◀
            </button>
            <button class="control-btn small" on:click={nextFrame} disabled={!$currentVideo} title="Next Frame (→)">
                ▶
            </button>
            <button class="control-btn small" on:click={skip10Forward} disabled={!$currentVideo} title="Skip 10 Frames Forward (Shift+→)">
                ▶▶
            </button>
            <button class="control-btn small" on:click={moveToEnd} disabled={!$currentVideo} title="Jump to End (End)">
                ⏭
            </button>
        </div>

        <div class="timecode-display">
            <span class="timecode">{timecode}</span>
            <span class="frame-count">
                Frame {$currentFrame} / {$currentVideo?.totalFrames || 0}
            </span>
        </div>

        <div class="mark-controls">
            <button class="control-btn small" on:click={handleSetInPoint} disabled={!$currentVideo} title="Set In Point (I)">
                IN
            </button>
            <button class="control-btn small" on:click={handleSetOutPoint} disabled={!$currentVideo} title="Set Out Point (O)">
                OUT
            </button>
        </div>
    </div>

    <div class="quick-actions">
        <div class="quick-section">
            <span class="section-label">Mark:</span>
            <button class="quick-btn mark-btn" on:click={quickSetIn} disabled={!$currentVideo} title="Set IN point at current frame">
                IN{#if $inPoint !== null} ({$inPoint}){/if}
            </button>
            <button class="quick-btn mark-btn" on:click={quickSetOut} disabled={!$currentVideo} title="Set OUT point at current frame">
                OUT{#if $outPoint !== null} ({$outPoint}){/if}
            </button>
        </div>

        <div class="quick-section">
            <span class="section-label">Trim:</span>
            <button class="quick-btn trim-btn" on:click={quickKeep} disabled={!$currentVideo || $inPoint === null || $outPoint === null} title="Keep only IN to OUT section">
                ✂️ Keep
            </button>
            <button class="quick-btn trim-btn" on:click={quickCutout} disabled={!$currentVideo || $inPoint === null || $outPoint === null} title="Remove IN to OUT section">
                Cutout
            </button>
        </div>

        <div class="quick-section">
            <span class="section-label">Crop:</span>
            <button class="quick-btn" on:click={() => cropEnabled.update(v => !v)} disabled={!$currentVideo} title="Toggle Crop Mode">
                {$cropEnabled ? '✓ Crop Mode' : 'Crop Mode'}
            </button>
            <button class="quick-btn crop-btn" on:click={quickCrop} disabled={!$currentVideo || !$cropEnabled} title="Apply crop region">
                ✅ Apply
            </button>
        </div>

        <button class="quick-btn undo" on:click={quickUndo} disabled={!$currentVideo} title="Undo Last Edit">
            ↶ Undo
        </button>
        <button class="quick-btn save" on:click={quickSave} disabled={!$currentVideo} title="Save to 'edited' folder">
            💾 Save
        </button>
    </div>

    <Timeline
        on:scrubStart={handleScrubStart}
        on:scrubEnd={handleScrubEnd}
        on:frameChange={handleFrameChange}
    />

    <!-- Notification overlay (doesn't affect layout) -->
    {#if notificationMessage}
        <div class="notification-overlay">
            {notificationMessage}
        </div>
    {/if}
</div>

<style>
.video-player {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--bg-primary);
    position: relative;
}

.video-viewport {
    flex: 1;
    min-height: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #000;
    position: relative;
    overflow: hidden;
}

.image-container {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    max-width: 100%;
    max-height: 100%;
    width: 100%;
    height: 100%;
}

.frame-image {
    max-width: 100%;
    max-height: 100%;
    display: block;
    object-fit: contain;
    image-rendering: crisp-edges;
    transition: opacity 0.1s ease;
}

.frame-video {
    max-width: 100%;
    max-height: 100%;
    display: block;
    object-fit: contain;
}

.frame-video.hidden,
.frame-image.hidden {
    display: none;
}

.frame-image.hover-preview {
    opacity: 0.9;
}

.loading {
    text-align: center;
    color: var(--text-secondary);
}

.loading p {
    font-size: 14px;
}

.no-video {
    text-align: center;
    color: var(--text-secondary);
    opacity: 0.5;
}

.placeholder-icon {
    font-size: 64px;
    margin-bottom: 16px;
}

.notification-overlay {
    position: absolute;
    bottom: 16px;
    left: 50%;
    transform: translateX(-50%);
    padding: 12px 20px;
    background: rgba(81, 207, 102, 0.95);
    border: 1px solid #51cf66;
    border-radius: 6px;
    color: white;
    font-size: 13px;
    font-weight: 500;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
    z-index: 1000;
    pointer-events: none;
    max-width: 80%;
    text-align: center;
    animation: slideUp 0.3s ease-out;
}

.quick-actions {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 8px 16px;
    background: var(--bg-secondary);
    border-top: 1px solid var(--border-color);
    flex-wrap: wrap;
}

.quick-section {
    display: flex;
    align-items: center;
    gap: 6px;
}

.section-label {
    font-size: 11px;
    color: var(--text-secondary);
    margin-right: 4px;
    font-weight: 500;
}

.quick-btn {
    padding: 5px 10px;
    background: transparent;
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 2px;
    font-size: 11px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s ease;
}

.quick-btn:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.05);
}

.quick-btn:disabled {
    opacity: 0.3;
    cursor: not-allowed;
}

.quick-btn.mark-btn {
    min-width: 55px;
}

.quick-btn.save {
    margin-left: auto;
    border-color: #51cf66;
    color: #51cf66;
}

.quick-btn.save:hover:not(:disabled) {
    background: rgba(81, 207, 102, 0.1);
}

.quick-btn.undo {
    border-color: #ff922b;
    color: #ff922b;
}

.quick-btn.undo:hover:not(:disabled) {
    background: rgba(255, 146, 43, 0.1);
}

.controls {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    background: var(--bg-secondary);
    border-top: 1px solid var(--border-color);
}

.control-btn {
    padding: 8px 16px;
    background: var(--bg-primary);
    color: var(--text-primary);
    border-radius: 4px;
    font-size: 16px;
    font-weight: 600;
    min-width: 48px;
}

.control-btn.small {
    padding: 6px 12px;
    font-size: 12px;
    min-width: auto;
}

.control-btn:hover:not(:disabled) {
    background: #333;
}

.frame-controls {
    display: flex;
    gap: 4px;
}

.mark-controls {
    display: flex;
    gap: 4px;
    margin-left: auto;
}

.timecode-display {
    display: flex;
    flex-direction: column;
    gap: 2px;
    font-family: 'Consolas', 'Monaco', monospace;
    font-size: 12px;
    color: var(--text-primary);
}

.timecode {
    font-size: 16px;
    font-weight: 600;
    letter-spacing: 0.05em;
}

.frame-count {
    font-size: 11px;
    color: var(--text-secondary);
}

.frame-preview-notification {
    position: absolute;
    top: 20px;
    left: 50%;
    transform: translateX(-50%);
    background: rgba(0, 0, 0, 0.85);
    color: #fff;
    padding: 8px 16px;
    border-radius: 4px;
    font-size: 14px;
    font-weight: 600;
    pointer-events: none;
    z-index: 100;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}

@keyframes slideUp {
    from {
        opacity: 0;
        transform: translateX(-50%) translateY(20px);
    }
    to {
        opacity: 1;
        transform: translateX(-50%) translateY(0);
    }
}
</style>
