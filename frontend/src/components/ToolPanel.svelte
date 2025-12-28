<script>
import { onMount } from 'svelte';
import AccordionGroup from './AccordionGroup.svelte';
import CutTool from './CutTool.svelte';
import EditHistory from './EditHistory.svelte';
import CropTool from './CropTool.svelte';
import TransformTool from './TransformTool.svelte';
import FrameOpsTool from './FrameOpsTool.svelte';
import AdjustmentsTool from './AdjustmentsTool.svelte';
import AdvancedTool from './AdvancedTool.svelte';
import FormatConversionTool from './FormatConversionTool.svelte';
import FileManagementTool from './FileManagementTool.svelte';
import { accordionExpanded } from '../stores/uiStore.js';
import { LoadPanelStates, SavePanelStates } from '../../wailsjs/go/app/App.js';

let cutExpanded = false;
let historyExpanded = false;
let cropExpanded = false;
let transformExpanded = false;
let frameOpsExpanded = false;
let adjustmentsExpanded = false;
let advancedExpanded = false;
let formatExpanded = false;
let fileManagementExpanded = false;
let infoExpanded = false;
let isLoading = true; // Prevent saving during initial load

// Load panel states on mount
onMount(async () => {
    try {
        const states = await LoadPanelStates();
        if (states) {
            cutExpanded = states.cut || false;
            historyExpanded = states.history || false;
            cropExpanded = states.crop || false;
            transformExpanded = states.transform || false;
            frameOpsExpanded = states.frameOps || false;
            adjustmentsExpanded = states.adjustments || false;
            advancedExpanded = states.advanced || false;
            formatExpanded = states.format || false;
            fileManagementExpanded = states.fileManagement || false;
            infoExpanded = states.info || false;
        }
    } catch (error) {
        console.error('Failed to load panel states:', error);
    } finally {
        isLoading = false;
    }
});

// Auto-save panel states when they change (but not during initial load)
async function savePanelStates() {
    if (isLoading) return;

    try {
        const states = {
            cut: cutExpanded,
            history: historyExpanded,
            crop: cropExpanded,
            transform: transformExpanded,
            frameOps: frameOpsExpanded,
            adjustments: adjustmentsExpanded,
            advanced: advancedExpanded,
            format: formatExpanded,
            fileManagement: fileManagementExpanded,
            info: infoExpanded
        };
        await SavePanelStates(states);
    } catch (error) {
        console.error('Failed to save panel states:', error);
    }
}

function expandAll() {
    cutExpanded = true;
    cropExpanded = true;
    transformExpanded = true;
    frameOpsExpanded = true;
    adjustmentsExpanded = true;
    advancedExpanded = true;
    formatExpanded = true;
    fileManagementExpanded = true;
    infoExpanded = true;
    savePanelStates();
}

function collapseAll() {
    cutExpanded = false;
    cropExpanded = false;
    transformExpanded = false;
    frameOpsExpanded = false;
    adjustmentsExpanded = false;
    advancedExpanded = false;
    formatExpanded = false;
    fileManagementExpanded = false;
    infoExpanded = false;
    savePanelStates();
}

$: accordionExpanded.set({
    cut: cutExpanded,
    history: historyExpanded,
    crop: cropExpanded,
    transform: transformExpanded,
    frameOps: frameOpsExpanded,
    adjustments: adjustmentsExpanded,
    advanced: advancedExpanded,
    format: formatExpanded,
    fileManagement: fileManagementExpanded,
    info: infoExpanded
});

// Auto-save when any panel state changes (after initial load)
$: if (!isLoading) {
    savePanelStates();
}
</script>

<div class="tool-panel">
    <div class="header">
        <h2>Tools</h2>
        <div class="panel-controls">
            <button class="panel-btn" on:click={expandAll} title="Expand all panels">
                ▼ Expand All
            </button>
            <button class="panel-btn" on:click={collapseAll} title="Collapse all panels">
                ▲ Collapse All
            </button>
        </div>
    </div>

    <div class="tools-container">
        <AccordionGroup title="Trim Operations" bind:expanded={cutExpanded}>
            <CutTool />
        </AccordionGroup>

        <AccordionGroup title="Crop Region" bind:expanded={cropExpanded}>
            <CropTool />
        </AccordionGroup>

        <AccordionGroup title="Transform & Scale" bind:expanded={transformExpanded}>
            <TransformTool />
        </AccordionGroup>

        <AccordionGroup title="Frame Operations" bind:expanded={frameOpsExpanded}>
            <FrameOpsTool />
        </AccordionGroup>

        <AccordionGroup title="Quality Adjustments" bind:expanded={adjustmentsExpanded}>
            <AdjustmentsTool />
        </AccordionGroup>

        <AccordionGroup title="Advanced Operations" bind:expanded={advancedExpanded}>
            <AdvancedTool />
        </AccordionGroup>

        <AccordionGroup title="Format Conversion" bind:expanded={formatExpanded}>
            <FormatConversionTool />
        </AccordionGroup>

        <AccordionGroup title="File Management" bind:expanded={fileManagementExpanded}>
            <FileManagementTool />
        </AccordionGroup>

        <AccordionGroup title="Edit Stack & Save" bind:expanded={historyExpanded}>
            <EditHistory expanded={historyExpanded} />
        </AccordionGroup>

        <AccordionGroup title="Info" bind:expanded={infoExpanded}>
            <div class="info-section">
                <p class="info-text">
                    Use the video player to navigate your video frame-by-frame.
                </p>
                <div class="shortcuts">
                    <h4>Keyboard Shortcuts</h4>
                    <div class="shortcut-item">
                        <kbd>Space</kbd> <span>Play/Pause</span>
                    </div>
                    <div class="shortcut-item">
                        <kbd>←</kbd> <span>Previous Frame</span>
                    </div>
                    <div class="shortcut-item">
                        <kbd>→</kbd> <span>Next Frame</span>
                    </div>
                    <div class="shortcut-item">
                        <kbd>I</kbd> <span>Set In Point</span>
                    </div>
                    <div class="shortcut-item">
                        <kbd>O</kbd> <span>Set Out Point</span>
                    </div>
                </div>
            </div>
        </AccordionGroup>
    </div>
</div>

<style>
.tool-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.header {
    padding: 12px 16px;
    border-bottom: 1px solid var(--border-color);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
}

.header h2 {
    font-size: 16px;
    color: var(--text-primary);
    margin: 0;
}

.panel-controls {
    display: flex;
    gap: 6px;
}

.panel-btn {
    padding: 4px 8px;
    background: var(--bg-secondary);
    color: var(--text-primary);
    border: 1px solid var(--border-color);
    border-radius: 3px;
    font-size: 10px;
    cursor: pointer;
    transition: all 0.15s ease;
    white-space: nowrap;
}

.panel-btn:hover {
    background: #333;
    border-color: #555;
}

.tools-container {
    flex: 1;
    overflow-y: auto;
}

.info-section {
    font-size: 12px;
    color: var(--text-secondary);
}

.info-text {
    margin-bottom: 16px;
    line-height: 1.5;
}

.shortcuts h4 {
    font-size: 11px;
    text-transform: uppercase;
    color: var(--text-secondary);
    margin-bottom: 10px;
    letter-spacing: 0.5px;
}

.shortcut-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 0;
    color: var(--text-primary);
}

kbd {
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 3px;
    padding: 2px 8px;
    font-family: 'Consolas', 'Monaco', monospace;
    font-size: 11px;
    min-width: 32px;
    text-align: center;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}
</style>
