<script>
import { currentVideo } from '../stores/videoStore.js';
import { CaptionVideo } from '../../wailsjs/go/app/App.js';

let isCaptioning = false;
let errorMessage = '';
let captionResult = '';

let apiBase = localStorage.getItem('noein.captionApiBase') || 'http://localhost:11434/v1';
let apiKey = localStorage.getItem('noein.captionApiKey') || '';
let model = localStorage.getItem('noein.captionModel') || 'llava';
let numFrames = Number(localStorage.getItem('noein.captionNumFrames') || '3');
let prompt = localStorage.getItem('noein.captionPrompt') || 'Describe what is happening in these video frames.';

$: localStorage.setItem('noein.captionApiBase', apiBase || '');
$: localStorage.setItem('noein.captionApiKey', apiKey || '');
$: localStorage.setItem('noein.captionModel', model || '');
$: localStorage.setItem('noein.captionNumFrames', String(numFrames || 3));
$: localStorage.setItem('noein.captionPrompt', prompt || '');

async function captionCurrentVideo() {
    if (!$currentVideo) return;
    errorMessage = '';
    captionResult = '';
    isCaptioning = true;
    try {
        captionResult = await CaptionVideo($currentVideo.id, apiBase, apiKey, model, prompt, numFrames);
    } catch (e) {
        errorMessage = e?.message || String(e);
    } finally {
        isCaptioning = false;
    }
}
</script>

<div class="caption-tool">
    <label>
        <div class="label">API Base URL</div>
        <input type="text" bind:value={apiBase} placeholder="http://localhost:11434/v1" />
    </label>

    <label>
        <div class="label">API Key (optional)</div>
        <input type="password" bind:value={apiKey} placeholder="sk-..." />
    </label>

    <label>
        <div class="label">Model</div>
        <input type="text" bind:value={model} placeholder="llava" />
    </label>

    <div class="row">
        <label class="flex-1">
            <div class="label">Frames to send</div>
            <select class="select" bind:value={numFrames}>
                <option value={1}>1 frame</option>
                <option value={3}>3 frames</option>
                <option value={5}>5 frames</option>
                <option value={8}>8 frames</option>
            </select>
        </label>
    </div>

    <label>
        <div class="label">Prompt</div>
        <textarea bind:value={prompt} rows="3" placeholder="Describe what is happening in these video frames."></textarea>
    </label>

    <button class="btn btn-primary" on:click={captionCurrentVideo} disabled={!$currentVideo || !model || isCaptioning}>
        {isCaptioning ? 'Captioning...' : 'Caption Current Video'}
    </button>

    {#if errorMessage}
        <div class="error">{errorMessage}</div>
    {/if}

    {#if captionResult}
        <div class="result">
            <div class="label">Caption</div>
            <div class="caption-text">{captionResult}</div>
        </div>
    {/if}

    {#if !$currentVideo}
        <div class="hint">Load a video to caption.</div>
    {/if}
</div>

<style>
.caption-tool {
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.row {
    display: flex;
    gap: 8px;
    align-items: flex-end;
}

.flex-1 {
    flex: 1;
}

.label {
    font-size: 11px;
    color: var(--text-secondary);
    margin-bottom: 4px;
}

input, textarea {
    width: 100%;
    padding: 8px 10px;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    color: var(--text-primary);
    font-size: 13px;
    font-family: inherit;
    resize: vertical;
}

.select {
    width: 100%;
    padding: 8px 10px;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    color: var(--text-primary);
    font-size: 12px;
}

.btn {
    padding: 8px 10px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 500;
    transition: all 0.2s ease;
    border: 1px solid var(--border-color);
    cursor: pointer;
    background: var(--bg-secondary);
    color: var(--text-primary);
}

.btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
}

.btn-primary {
    background: var(--accent-green);
    border-color: transparent;
    color: white;
}

.hint {
    font-size: 12px;
    color: var(--text-secondary);
    font-style: italic;
}

.error {
    font-size: 12px;
    color: #ff6b6b;
    background: rgba(255, 107, 107, 0.1);
    border: 1px solid rgba(255, 107, 107, 0.2);
    padding: 8px 10px;
    border-radius: 4px;
    word-break: break-word;
}

.result {
    padding: 10px;
    border: 1px solid var(--border-color);
    border-radius: 6px;
    background: var(--bg-secondary);
}

.caption-text {
    font-size: 12px;
    color: var(--text-primary);
    line-height: 1.5;
    word-break: break-word;
    white-space: pre-wrap;
}
</style>
