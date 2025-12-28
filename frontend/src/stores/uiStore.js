import { writable } from 'svelte/store';

export const activeTool = writable('cut');
export const accordionExpanded = writable({
    cut: true,
    marks: false
});
export const isLoading = writable(false);
export const errorMessage = writable('');
