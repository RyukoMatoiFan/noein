<script>
import { currentVideo, currentFrame, isPlaying, playbackEndFrame, videoList } from '../stores/videoStore.js';
import { inPoint, outPoint } from '../stores/projectStore.js';
import { DetectSpeechFragments, ExportSegment, ExportSpeechDataset, ExportWhisperTranscript, EnsureWhisperCLI, EnsureWhisperModel, OllamaAnnotateSpeechFragments, OllamaListModels, SelectOutputDirectory, SelectOutputFile, SelectWhisperExecutable, SelectWhisperModel, SetInPoint, SetOutPoint } from '../../wailsjs/go/app/App.js';

let fragments = [];
let isAnalyzing = false;
let isExportingAll = false;
let isDownloading = false;
let isExportingTranscript = false;
let isLoadingOllamaModels = false;
let isAnnotating = false;
let isBatchExtracting = false;
let batchProgress = '';
let batchResults = [];
let errorMessage = '';

let whisperPath = localStorage.getItem('noein.whisperPath') || '';
let modelPath = localStorage.getItem('noein.whisperModelPath') || '';
let modelName = localStorage.getItem('noein.whisperModelName') || 'tiny.en-q5_1';
let manifestName = localStorage.getItem('noein.whisperDatasetManifestName') || 'dataset.jsonl';
let mergeGapMs = Number(localStorage.getItem('noein.whisperMergeGapMs') || '400');
let minFragmentMs = Number(localStorage.getItem('noein.whisperMinFragmentMs') || '800');
let splitOnSilence = localStorage.getItem('noein.whisperSplitOnSilence') !== 'false';
let silenceDurationMs = Number(localStorage.getItem('noein.whisperSilenceDurationMs') || '300');
// Reset any stale threshold; FFmpeg default is -60dB
let _storedThresh = localStorage.getItem('noein.whisperSilenceThresholdDb');
let silenceThresholdDb = (_storedThresh === null || _storedThresh === '-20' || _storedThresh === '-30') ? -50 : Number(_storedThresh);

let ollamaBaseURL = localStorage.getItem('noein.ollamaBaseURL') || 'http://localhost:11434';
let ollamaModel = localStorage.getItem('noein.ollamaModel') || '';
let ollamaModels = [];

$: localStorage.setItem('noein.ollamaBaseURL', ollamaBaseURL || '');
$: localStorage.setItem('noein.ollamaModel', ollamaModel || '');
$: localStorage.setItem('noein.whisperPath', whisperPath || '');
$: localStorage.setItem('noein.whisperModelPath', modelPath || '');
$: localStorage.setItem('noein.whisperModelName', modelName || '');
$: localStorage.setItem('noein.whisperDatasetManifestName', manifestName || '');
$: localStorage.setItem('noein.whisperMergeGapMs', String(mergeGapMs || 0));
$: localStorage.setItem('noein.whisperMinFragmentMs', String(minFragmentMs || 0));
$: localStorage.setItem('noein.whisperSplitOnSilence', String(splitOnSilence));
$: localStorage.setItem('noein.whisperSilenceDurationMs', String(silenceDurationMs || 0));
$: localStorage.setItem('noein.whisperSilenceThresholdDb', String(silenceThresholdDb || -30));

function formatTime(seconds) {
    const s = Math.max(0, seconds || 0);
    const hh = Math.floor(s / 3600);
    const mm = Math.floor((s % 3600) / 60);
    const ss = Math.floor(s % 60);
    const ms = Math.floor((s - Math.floor(s)) * 1000);
    const pad2 = (n) => String(n).padStart(2, '0');
    const pad3 = (n) => String(n).padStart(3, '0');
    return `${pad2(hh)}:${pad2(mm)}:${pad2(ss)}.${pad3(ms)}`;
}

async function pickWhisper() {
    try {
        const file = await SelectWhisperExecutable();
        if (file) whisperPath = file;
    } catch (e) {
        errorMessage = e?.message || String(e);
    }
}

async function pickModel() {
    try {
        const file = await SelectWhisperModel();
        if (file) modelPath = file;
    } catch (e) {
        errorMessage = e?.message || String(e);
    }
}

async function autoDownload() {
    errorMessage = '';
    isDownloading = true;
    try {
        whisperPath = await EnsureWhisperCLI();
        modelPath = await EnsureWhisperModel(modelName);
    } catch (e) {
        errorMessage = e?.message || String(e);
    } finally {
        isDownloading = false;
    }
}

async function analyze() {
    if (!$currentVideo) return;
    errorMessage = '';
    fragments = [];
    isAnalyzing = true;
    try {
        fragments = await DetectSpeechFragments($currentVideo.id, whisperPath, modelPath, mergeGapMs, minFragmentMs, splitOnSilence, silenceDurationMs, silenceThresholdDb);
    } catch (e) {
        errorMessage = e?.message || String(e);
    } finally {
        isAnalyzing = false;
    }
}

async function jumpTo(fragment) {
    if (!fragment) return;
    isPlaying.set(false);
    currentFrame.set(fragment.inFrame || 0);
    // Set IN/OUT marks to fragment boundaries
    await SetInPoint(fragment.inFrame || 0);
    inPoint.set(fragment.inFrame || 0);
    await SetOutPoint(fragment.outFrame || 0);
    outPoint.set(fragment.outFrame || 0);
}

function playFragment(fragment) {
    if (!fragment) return;
    // Stop any current playback first
    isPlaying.set(false);
    // Jump to fragment start and set end frame
    currentFrame.set(fragment.inFrame || 0);
    playbackEndFrame.set(fragment.outFrame || 0);
    // Small delay to let the frame seek settle, then trigger playback
    setTimeout(() => {
        if (window.__noeinStartPlayback) {
            window.__noeinStartPlayback();
        }
    }, 100);
}

async function exportOne(fragment) {
    if (!$currentVideo || !fragment) return;
    errorMessage = '';
    try {
        const outputPath = await SelectOutputFile();
        if (!outputPath) return;
        await ExportSegment($currentVideo.id, fragment.inFrame, fragment.outFrame, outputPath);
    } catch (e) {
        errorMessage = e?.message || String(e);
    }
}

async function exportAll() {
    if (!$currentVideo || fragments.length === 0) return;
    errorMessage = '';
    isExportingAll = true;
    try {
        const outputDir = await SelectOutputDirectory();
        if (!outputDir) return;
        const results = await ExportSpeechDataset($currentVideo.id, fragments, outputDir, manifestName);
        const failures = results.filter(r => !r.success);
        if (failures.length > 0) {
            errorMessage = `Exported with ${failures.length} failures. First error: ${failures[0].error || 'unknown'}`;
        }
    } catch (e) {
        errorMessage = e?.message || String(e);
    } finally {
        isExportingAll = false;
    }
}

async function exportTranscript(format) {
    if (!$currentVideo) return;
    errorMessage = '';
    isExportingTranscript = true;
    try {
        const outputDir = await SelectOutputDirectory();
        if (!outputDir) return;
        await ExportWhisperTranscript($currentVideo.id, whisperPath, modelPath, format, outputDir);
    } catch (e) {
        errorMessage = e?.message || String(e);
    } finally {
        isExportingTranscript = false;
    }
}

async function loadOllamaModels() {
    errorMessage = '';
    isLoadingOllamaModels = true;
    try {
        ollamaModels = await OllamaListModels(ollamaBaseURL);
        if (!ollamaModel && ollamaModels.length > 0) {
            ollamaModel = ollamaModels[0];
        }
    } catch (e) {
        errorMessage = e?.message || String(e);
    } finally {
        isLoadingOllamaModels = false;
    }
}

async function annotateWithOllama() {
    if (fragments.length === 0) return;
    errorMessage = '';
    isAnnotating = true;
    try {
        fragments = await OllamaAnnotateSpeechFragments(fragments, ollamaBaseURL, ollamaModel);
    } catch (e) {
        errorMessage = e?.message || String(e);
    } finally {
        isAnnotating = false;
    }
}

async function batchExtractAll() {
    const videos = $videoList;
    if (!videos || videos.length === 0) return;
    errorMessage = '';
    batchResults = [];
    isBatchExtracting = true;
    try {
        const outputDir = await SelectOutputDirectory();
        if (!outputDir) {
            isBatchExtracting = false;
            return;
        }

        for (let i = 0; i < videos.length; i++) {
            const v = videos[i];
            const videoName = v.name.replace(/\.[^.]+$/, '').replace(/[\\/:*?"<>|]/g, '_');
            batchProgress = `Processing ${i + 1}/${videos.length}: ${v.name}`;

            const result = { name: v.name, success: false, fragmentCount: 0, error: '' };
            try {
                const frags = await DetectSpeechFragments(v.id, whisperPath, modelPath, mergeGapMs, minFragmentMs, splitOnSilence, silenceDurationMs, silenceThresholdDb);
                result.fragmentCount = frags ? frags.length : 0;

                if (frags && frags.length > 0) {
                    const subDir = outputDir + '/' + videoName;
                    await ExportSpeechDataset(v.id, frags, subDir, manifestName);
                }
                result.success = true;
            } catch (e) {
                result.error = e?.message || String(e);
            }
            batchResults = [...batchResults, result];
        }
        batchProgress = `Done — ${videos.length} videos processed.`;
    } catch (e) {
        errorMessage = e?.message || String(e);
    } finally {
        isBatchExtracting = false;
    }
}
</script>

<div class="speech-tool">
    <div class="row">
        <button class="btn" on:click={pickWhisper}>Select Whisper CLI</button>
        <div class="path" title={whisperPath}>{whisperPath || 'Not set'}</div>
    </div>

    <div class="row">
        <button class="btn" on:click={pickModel}>Select Model (.bin)</button>
        <div class="path" title={modelPath}>{modelPath || 'Not set'}</div>
    </div>

    <div class="row">
        <button class="btn" on:click={autoDownload} disabled={isDownloading}>
            {isDownloading ? 'Downloading…' : 'Auto-download Whisper + Model'}
        </button>
        <select class="select" bind:value={modelName} disabled={isDownloading}>
            <option value="tiny">tiny (multilingual, fast)</option>
            <option value="tiny.en-q5_1">tiny.en-q5_1 (English, fast)</option>
            <option value="base">base (multilingual)</option>
            <option value="base.en">base.en (English)</option>
            <option value="small">small (multilingual)</option>
            <option value="small.en">small.en (English)</option>
            <option value="medium">medium (multilingual, slow)</option>
            <option value="medium.en">medium.en (English, slow)</option>
        </select>
    </div>

    <div class="grid">
        <label>
            <div class="label">Merge Gap (ms)</div>
            <input type="number" min="0" step="50" bind:value={mergeGapMs} />
        </label>
        <label>
            <div class="label">Min Fragment (ms)</div>
            <input type="number" min="0" step="50" bind:value={minFragmentMs} />
        </label>
    </div>

    <div class="row">
        <label class="checkbox-label">
            <input type="checkbox" bind:checked={splitOnSilence} />
            <span>Split on silence (pauses within speech)</span>
        </label>
    </div>

    {#if splitOnSilence}
    <div class="grid">
        <label>
            <div class="label">Min Silence (ms)</div>
            <input type="number" min="50" step="50" bind:value={silenceDurationMs} />
        </label>
        <label>
            <div class="label">Threshold (dB)</div>
            <input type="number" max="-1" step="5" bind:value={silenceThresholdDb} />
        </label>
    </div>
    {/if}

    <div class="row">
        <label class="manifest">
            <div class="label">Manifest Name</div>
            <input type="text" bind:value={manifestName} />
        </label>
        <button class="btn" on:click={() => exportTranscript('srt')} disabled={!$currentVideo || isExportingTranscript}>
            {isExportingTranscript ? 'Exporting…' : 'Export Transcript (SRT)'}
        </button>
        <button class="btn" on:click={() => exportTranscript('vtt')} disabled={!$currentVideo || isExportingTranscript}>
            {isExportingTranscript ? 'Exporting…' : 'VTT'}
        </button>
    </div>

    <div class="row">
        <button class="btn btn-primary" on:click={analyze} disabled={!$currentVideo || isAnalyzing}>
            {isAnalyzing ? 'Analyzing…' : 'Find Speech Fragments'}
        </button>
        <button class="btn" on:click={exportAll} disabled={!$currentVideo || fragments.length === 0 || isExportingAll}>
            {isExportingAll ? 'Exporting…' : `Export All (${fragments.length})`}
        </button>
    </div>

    <div class="row">
        <button class="btn" on:click={batchExtractAll} disabled={!$videoList || $videoList.length === 0 || isBatchExtracting}>
            {isBatchExtracting ? 'Batch Running…' : `Batch Extract All (${$videoList?.length || 0} videos)`}
        </button>
    </div>

    {#if batchProgress}
        <div class="hint">{batchProgress}</div>
    {/if}

    {#if batchResults.length > 0}
        <div class="batch-results">
            {#each batchResults as r, idx (idx)}
                <div class="batch-row" class:batch-fail={!r.success}>
                    <span class="batch-name" title={r.name}>{r.name}</span>
                    {#if r.success}
                        <span class="batch-ok">{r.fragmentCount} fragments</span>
                    {:else}
                        <span class="batch-err" title={r.error}>failed</span>
                    {/if}
                </div>
            {/each}
        </div>
    {/if}

    <div class="row">
        <label class="manifest">
            <div class="label">Ollama Base URL</div>
            <input type="text" bind:value={ollamaBaseURL} placeholder="http://localhost:11434" />
        </label>
        <button class="btn" on:click={loadOllamaModels} disabled={isLoadingOllamaModels}>
            {isLoadingOllamaModels ? 'Loading…' : 'Load Ollama Models'}
        </button>
    </div>

    <div class="row">
        <select class="select" bind:value={ollamaModel} disabled={isAnnotating || isLoadingOllamaModels}>
            {#if ollamaModels.length === 0}
                <option value="">(no models loaded)</option>
            {:else}
                {#each ollamaModels as m (m)}
                    <option value={m}>{m}</option>
                {/each}
            {/if}
        </select>
        <button class="btn" on:click={annotateWithOllama} disabled={fragments.length === 0 || !ollamaModel || isAnnotating}>
            {isAnnotating ? 'Annotating…' : `Annotate (${fragments.length})`}
        </button>
    </div>

    {#if errorMessage}
        <div class="error">{errorMessage}</div>
    {/if}

    {#if !$currentVideo}
        <div class="hint">Load a video to analyze speech.</div>
    {:else if fragments.length === 0}
        <div class="hint">No speech fragments yet.</div>
    {:else}
        <div class="fragments">
            {#each fragments as f, idx (f.id || idx)}
                <div class="fragment">
                    <div class="meta">
                        <div class="time">{formatTime(f.startSec)} → {formatTime(f.endSec)}</div>
                        <div class="text">{f.text}</div>
                        {#if f.label}
                            <div class="ollama-label">{f.label}</div>
                        {/if}
                        {#if f.tags && f.tags.length > 0}
                            <div class="ollama-tags">{f.tags.join(' · ')}</div>
                        {/if}
                    </div>
                    <div class="actions">
                        <button class="btn small" on:click={() => jumpTo(f)}>Jump</button>
                        <button class="btn small" on:click={() => playFragment(f)}>Play</button>
                        <button class="btn small" on:click={() => exportOne(f)}>Extract</button>
                    </div>
                </div>
            {/each}
        </div>
    {/if}
</div>

<style>
.speech-tool {
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.row {
    display: flex;
    gap: 8px;
    align-items: center;
}

.grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
}

.label {
    font-size: 11px;
    color: var(--text-secondary);
    margin-bottom: 4px;
}

input {
    width: 100%;
    padding: 8px 10px;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    color: var(--text-primary);
    font-size: 13px;
}

.checkbox-label {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: var(--text-primary);
    cursor: pointer;
}

.checkbox-label input[type="checkbox"] {
    width: auto;
    margin: 0;
}

.manifest {
    flex: 1;
    min-width: 0;
}

.manifest input {
    font-size: 12px;
}

.select {
    flex: 1;
    min-width: 0;
    padding: 8px 10px;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    color: var(--text-primary);
    font-size: 12px;
}

.path {
    flex: 1;
    min-width: 0;
    font-size: 11px;
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    border: 1px solid var(--border-color);
    border-radius: 4px;
    padding: 8px 10px;
    background: var(--bg-primary);
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

.btn.small {
    padding: 6px 8px;
    font-size: 11px;
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

.fragments {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 320px;
    overflow: auto;
}

.fragment {
    display: flex;
    gap: 10px;
    align-items: flex-start;
    padding: 10px;
    border: 1px solid var(--border-color);
    border-radius: 6px;
    background: var(--bg-secondary);
}

.meta {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.time {
    font-size: 11px;
    color: var(--text-secondary);
}

.ollama-label {
    font-size: 11px;
    color: var(--text-primary);
}

.ollama-tags {
    font-size: 10px;
    color: var(--text-secondary);
}

.text {
    font-size: 12px;
    color: var(--text-primary);
    word-break: break-word;
    line-height: 1.2;
}

.actions {
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.batch-results {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: 200px;
    overflow: auto;
}

.batch-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 6px 10px;
    border: 1px solid var(--border-color);
    border-radius: 4px;
    font-size: 11px;
    background: var(--bg-secondary);
}

.batch-row.batch-fail {
    border-color: rgba(255, 107, 107, 0.3);
}

.batch-name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text-primary);
}

.batch-ok {
    color: var(--accent-green);
    white-space: nowrap;
    margin-left: 8px;
}

.batch-err {
    color: #ff6b6b;
    white-space: nowrap;
    margin-left: 8px;
}
</style>
