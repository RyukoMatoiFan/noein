<script>
export let title;
export let expanded = true;

function toggle() {
    expanded = !expanded;
}
</script>

<div class="accordion-group">
    <button class="accordion-header" on:click={toggle}>
        <span class="arrow" class:expanded>{expanded ? '▼' : '▶'}</span>
        <span class="title">{title}</span>
    </button>

    {#if expanded}
        <div class="accordion-content">
            <slot />
        </div>
    {/if}
</div>

<style>
.accordion-group {
    border-bottom: 1px solid var(--border-color);
}

.accordion-header {
    width: 100%;
    padding: 12px 16px;
    background: transparent;
    color: var(--text-primary);
    text-align: left;
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    font-weight: 600;
    transition: background 0.2s ease;
}

.accordion-header:hover {
    background: rgba(255, 255, 255, 0.05);
}

.arrow {
    font-size: 10px;
    transition: transform 0.2s ease;
    color: var(--text-secondary);
}

.title {
    flex: 1;
}

.accordion-content {
    padding: 16px;
    animation: slideDown 0.2s ease;
}

@keyframes slideDown {
    from {
        opacity: 0;
        transform: translateY(-8px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}
</style>
