<script>
import './styles/global.css';
import VideoList from './components/VideoList.svelte';
import VideoPlayer from './components/VideoPlayer.svelte';
import ToolPanel from './components/ToolPanel.svelte';
</script>

<main class="app-container">
    <div class="panel panel-left">
        <VideoList />
    </div>

    <div class="panel panel-center">
        <VideoPlayer />
    </div>

    <div class="panel panel-right">
        <ToolPanel />
    </div>
</main>

<style>
.app-container {
    display: grid;
    grid-template-columns: 280px 1fr 320px;
    height: 100vh;
    gap: 1px;
    background: var(--border-color);
}

.panel {
    background: var(--bg-tertiary);
    overflow: hidden;
}

.panel-left {
    background: var(--bg-secondary);
}

.panel-center {
    display: flex;
    flex-direction: column;
    background: var(--bg-primary);
}

.panel-right {
    background: var(--bg-secondary);
}
</style>
