<script>
export let preview;
</script>

{#if preview && preview.frames}
    <div class="frame-preview">
        <div class="frames">
            {#each preview.frames as frame, i}
                <div class="frame-item" class:center={i === 2}>
                    {#if frame.imageData}
                        <img src={frame.imageData} alt="Frame {frame.frameNumber}" />
                        <span class="frame-number">{frame.frameNumber}</span>
                    {:else}
                        <div class="frame-empty">—</div>
                    {/if}
                </div>
            {/each}
        </div>
    </div>
{/if}

<style>
.frame-preview {
    position: absolute;
    bottom: 80px;
    left: 50%;
    transform: translateX(-50%);
    background: rgba(0, 0, 0, 0.95);
    padding: 8px;
    border-radius: 6px;
    z-index: 100;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.5);
}

.frames {
    display: flex;
    gap: 4px;
}

.frame-item {
    position: relative;
    width: 120px;
    opacity: 0.5;
    transition: opacity 0.2s;
    border-radius: 2px;
    overflow: hidden;
}

.frame-item.center {
    opacity: 1;
    border: 2px solid var(--accent-blue);
    box-shadow: 0 0 12px rgba(100, 200, 255, 0.5);
}

.frame-item img {
    width: 100%;
    height: auto;
    display: block;
}

.frame-number {
    position: absolute;
    bottom: 4px;
    left: 4px;
    background: rgba(0, 0, 0, 0.8);
    color: #fff;
    padding: 2px 6px;
    font-size: 10px;
    font-family: 'Consolas', 'Monaco', monospace;
    border-radius: 2px;
}

.frame-empty {
    width: 120px;
    height: 67px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #1a1a1a;
    color: #666;
    font-size: 24px;
}
</style>
