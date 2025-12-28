import { writable, derived } from 'svelte/store';

export const inPoint = writable(null);
export const outPoint = writable(null);
export const cropEnabled = writable(false);
export const cropClearSignal = writable(0); // Increment to signal crop rectangle should be cleared
export const selectedVideos = writable(new Set()); // Set of selected video IDs for batch processing
export const skipFrameReset = writable(false); // Flag to prevent auto-reset to frame 0 during save operations
export const frameBeforeEdits = writable(null); // Frame position in original video before any edits

export const currentMark = derived(
    [inPoint, outPoint],
    ([$inPoint, $outPoint]) => {
        if ($inPoint !== null && $outPoint !== null) {
            return {
                inFrame: $inPoint,
                outFrame: $outPoint,
                duration: $outPoint - $inPoint
            };
        }
        return null;
    }
);

export const hasSelection = derived(
    [inPoint, outPoint],
    ([$inPoint, $outPoint]) => $inPoint !== null && $outPoint !== null
);
