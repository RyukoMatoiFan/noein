<script>
import { onMount } from 'svelte';
import { videoList, currentVideo, currentFrame, isPlaying } from '../stores/videoStore.js';
import { selectedVideos } from '../stores/projectStore.js';
import { SelectFolder, LoadVideoFolder, SelectVideoFile, LoadVideoFile, SetCurrentVideo, ClearEditStack, ClearMarks } from '../../wailsjs/go/app/App.js';

let selectedCount = 0;

// LocalStorage keys for persistence
const STORAGE_KEYS = {
    LAST_VIDEO_ID: 'noein_last_video_id',
    RECENT_ITEMS: 'noein_recent_items'
};

const MAX_RECENT = 5;

// Recent items: { type: 'folder'|'file', path: string, label: string }
let recentItems = [];

function loadRecentItems() {
    try {
        const raw = localStorage.getItem(STORAGE_KEYS.RECENT_ITEMS);
        recentItems = raw ? JSON.parse(raw) : [];
    } catch {
        recentItems = [];
    }
}

function addRecentItem(type, path) {
    const label = path.split(/[/\\]/).pop() || path;
    recentItems = [
        { type, path, label },
        ...recentItems.filter(r => r.path !== path)
    ].slice(0, MAX_RECENT);
    localStorage.setItem(STORAGE_KEYS.RECENT_ITEMS, JSON.stringify(recentItems));
}

function removeRecentItem(path) {
    recentItems = recentItems.filter(r => r.path !== path);
    localStorage.setItem(STORAGE_KEYS.RECENT_ITEMS, JSON.stringify(recentItems));
}

// Subscribe to selectedVideos to update count
selectedVideos.subscribe(set => {
    selectedCount = set.size;
});

function toggleVideoSelection(videoId, event) {
    event.stopPropagation(); // Prevent video selection when clicking checkbox

    selectedVideos.update(current => {
        const newSet = new Set(current);
        if (newSet.has(videoId)) {
            newSet.delete(videoId);
        } else {
            newSet.add(videoId);
        }
        return newSet;
    });
}

function selectAll() {
    selectedVideos.update(() => {
        const newSet = new Set();
        $videoList.forEach(video => newSet.add(video.id));
        return newSet;
    });
}

function deselectAll() {
    selectedVideos.update(() => new Set());
}

async function selectVideo(video) {
    // Stop playback and reset to first frame when switching videos
    isPlaying.set(false);
    currentFrame.set(0);

    try {
        // Set current video (metadata already loaded)
        const updatedVideo = await SetCurrentVideo(video.id);

        if (!updatedVideo) {
            alert('Failed to load video');
            return;
        }

        currentVideo.set(updatedVideo);

        // Save last selected video ID
        localStorage.setItem(STORAGE_KEYS.LAST_VIDEO_ID, video.id);
    } catch (error) {
        alert(`Failed to load video: ${error.message || error}`);
    }
}

async function clearAndLoad() {
    selectedVideos.set(new Set());
    try {
        await ClearEditStack();
        await ClearMarks();
    } catch (error) {
        console.error('Failed to clear state:', error);
    }
}

async function handleLoadFolder() {
    try {
        const folderPath = await SelectFolder();
        if (folderPath) {
            await clearAndLoad();
            const videos = await LoadVideoFolder(folderPath);
            videoList.set(videos || []);
            addRecentItem('folder', folderPath);
        }
    } catch (error) {
        console.error('Failed to load folder:', error);
    }
}

async function handleLoadFile() {
    try {
        const filePath = await SelectVideoFile();
        if (filePath) {
            await clearAndLoad();
            const videos = await LoadVideoFile(filePath);
            videoList.set(videos || []);
            addRecentItem('file', filePath);
        }
    } catch (error) {
        console.error('Failed to load file:', error);
    }
}

async function loadRecentEntry(item) {
    try {
        await clearAndLoad();
        let videos;
        if (item.type === 'file') {
            videos = await LoadVideoFile(item.path);
        } else {
            videos = await LoadVideoFolder(item.path);
        }
        videoList.set(videos || []);
        addRecentItem(item.type, item.path);
    } catch (error) {
        console.error('Failed to load recent item:', error);
        removeRecentItem(item.path);
    }
}

onMount(() => {
    loadRecentItems();
});

function formatDuration(seconds) {
    if (!seconds || seconds === 0) return '--:--';
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
}

function formatResolution(width, height) {
    if (!width || !height || width === 0 || height === 0) return '×';
    return `${width}×${height}`;
}

function formatFrameRate(fps) {
    if (!fps || fps === 0) return '--fps';
    return `${fps.toFixed(0)}fps`;
}
</script>

<div class="video-list">
    <div class="header">
        <h2>Videos</h2>
        <div class="load-buttons">
            <button class="load-button" on:click={handleLoadFolder}>
                Load Folder
            </button>
            <button class="load-button load-file" on:click={handleLoadFile}>
                Load File
            </button>
        </div>
        {#if $videoList.length > 0}
            <div class="selection-controls">
                <button class="selection-btn" on:click={selectAll} title="Select all videos">
                    ☑ Select All
                </button>
                <button class="selection-btn" on:click={deselectAll} title="Deselect all videos">
                    ☐ Deselect All
                </button>
            </div>
        {/if}
        {#if selectedCount > 0}
            <div class="selection-info">
                {selectedCount} video{selectedCount === 1 ? '' : 's'} selected
            </div>
        {/if}
    </div>

    <div class="list-container">
        {#if $videoList.length === 0}
            <div class="empty-state">
                {#if recentItems.length > 0}
                    <p class="recent-title">Recent</p>
                    <div class="recent-list">
                        {#each recentItems as item (item.path)}
                            <button class="recent-item" on:click={() => loadRecentEntry(item)} title={item.path}>
                                <span class="recent-icon">{item.type === 'folder' ? '📁' : '🎬'}</span>
                                <span class="recent-label">{item.label}</span>
                                <button class="recent-remove" on:click|stopPropagation={() => removeRecentItem(item.path)} title="Remove">×</button>
                            </button>
                        {/each}
                    </div>
                {:else}
                    <p>No videos loaded</p>
                    <p class="hint">Load a folder or file to begin</p>
                {/if}
            </div>
        {:else}
            <div class="video-items">
                {#each $videoList as video (video.id)}
                    <button
                        class="video-item"
                        class:active={$currentVideo?.id === video.id}
                        class:selected={$selectedVideos.has(video.id)}
                        on:click={() => selectVideo(video)}
                        title={video.name}
                    >
                        <div class="video-item-content">
                            <input
                                type="checkbox"
                                class="video-checkbox"
                                checked={$selectedVideos.has(video.id)}
                                on:click={(e) => toggleVideoSelection(video.id, e)}
                            />
                            <div class="video-details">
                                <div class="video-name">{video.name}</div>
                                <div class="video-info">
                                    {#if video.audioOnly}
                                        <span class="audio-badge">audio</span>
                                    {:else}
                                        <span>{formatResolution(video.width, video.height)}</span>
                                    {/if}
                                    <span>•</span>
                                    <span>{formatDuration(video.duration)}</span>
                                    {#if !video.audioOnly}
                                        <span>•</span>
                                        <span>{formatFrameRate(video.frameRate)}</span>
                                    {/if}
                                </div>
                            </div>
                        </div>
                    </button>
                {/each}
            </div>
        {/if}
    </div>
</div>

<style>
.video-list {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.header {
    padding: 16px;
    border-bottom: 1px solid var(--border-color);
}

.header h2 {
    font-size: 18px;
    margin-bottom: 12px;
    color: var(--text-primary);
}

.selection-info {
    margin-top: 8px;
    padding: 6px 10px;
    background: rgba(81, 207, 102, 0.15);
    border: 1px solid #51cf66;
    border-radius: 3px;
    font-size: 11px;
    color: #51cf66;
    font-weight: 600;
}

.load-buttons {
    display: flex;
    gap: 6px;
}

.load-button {
    flex: 1;
    padding: 10px 16px;
    background: var(--accent-blue);
    color: var(--bg-primary);
    font-weight: 600;
    border-radius: 4px;
    font-size: 14px;
}

.load-button:hover {
    background: #50b4eb;
}

.load-button.load-file {
    background: var(--bg-secondary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
}

.load-button.load-file:hover {
    background: #333;
    border-color: #555;
}

.recent-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 8px;
}

.recent-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
    text-align: left;
}

.recent-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 10px;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.15s ease;
    width: 100%;
    text-align: left;
    color: var(--text-primary);
}

.recent-item:hover {
    background: #2a2a2a;
    border-color: #555;
}

.recent-icon {
    flex-shrink: 0;
    font-size: 14px;
}

.recent-label {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 13px;
}

.recent-remove {
    flex-shrink: 0;
    width: 20px;
    height: 20px;
    border-radius: 3px;
    background: transparent;
    color: var(--text-secondary);
    font-size: 14px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    padding: 0;
}

.recent-remove:hover {
    background: rgba(255, 100, 100, 0.2);
    color: #ff6b6b;
}

.selection-controls {
    display: flex;
    gap: 6px;
    margin-top: 8px;
}

.selection-btn {
    flex: 1;
    padding: 6px 10px;
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 3px;
    font-size: 11px;
    cursor: pointer;
    transition: all 0.15s ease;
    white-space: nowrap;
}

.selection-btn:hover {
    background: #333;
    border-color: #555;
}

.list-container {
    flex: 1;
    overflow-y: auto;
}

.empty-state {
    padding: 32px 16px;
    text-align: center;
    color: var(--text-secondary);
}

.empty-state p {
    margin-bottom: 8px;
}

.empty-state .hint {
    font-size: 12px;
    opacity: 0.7;
}

.video-items {
    padding: 4px;
}

.video-item {
    width: 100%;
    padding: 8px 10px;
    margin-bottom: 2px;
    background: var(--bg-primary);
    border-radius: 3px;
    text-align: left;
    border: 1px solid transparent;
    transition: all 0.15s ease;
    cursor: pointer;
}

.video-item:hover {
    background: #2a2a2a;
    border-color: var(--border-color);
}

.video-item.active {
    background: var(--accent-blue);
    color: var(--bg-primary);
    border-color: var(--accent-blue);
}

.video-item.selected {
    background: rgba(81, 207, 102, 0.1);
    border-color: #51cf66;
}

.video-item.selected .video-checkbox {
    accent-color: #51cf66;
}

.video-item.active .video-info {
    color: rgba(0, 0, 0, 0.75);
}

.video-item-content {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
}

.video-checkbox {
    flex-shrink: 0;
    width: 16px;
    height: 16px;
    cursor: pointer;
}

.video-details {
    flex: 1;
    min-width: 0;
}

.video-name {
    font-weight: 600;
    font-size: 13px;
    margin-bottom: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text-primary);
    line-height: 1.3;
}

.video-item.active .video-name {
    color: var(--bg-primary);
}

.video-info {
    font-size: 10px;
    color: var(--text-secondary);
    display: flex;
    gap: 6px;
    font-family: 'Consolas', 'Monaco', monospace;
    align-items: center;
}

.audio-badge {
    background: rgba(81, 207, 102, 0.15);
    color: #51cf66;
    padding: 1px 5px;
    border-radius: 3px;
    font-size: 9px;
    font-weight: 600;
    text-transform: uppercase;
}
</style>
