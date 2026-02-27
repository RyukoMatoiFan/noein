import { writable, derived } from 'svelte/store';

export const currentVideo = writable(null);
export const videoList = writable([]);
export const currentFrame = writable(0);
export const isPlaying = writable(false);
export const playbackEndFrame = writable(null); // When set, playback stops at this frame
export const framePreview = writable(null);

// Derived stores
export const currentTimestamp = derived(
    [currentVideo, currentFrame],
    ([$currentVideo, $currentFrame]) => {
        if (!$currentVideo) return 0;
        return $currentFrame / $currentVideo.frameRate;
    }
);

export const totalDuration = derived(
    currentVideo,
    ($currentVideo) => $currentVideo?.duration || 0
);

export const totalFrames = derived(
    currentVideo,
    ($currentVideo) => $currentVideo?.totalFrames || 0
);
