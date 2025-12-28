<script>
  import { AddFormatConversion } from '../../wailsjs/go/app/App.js';

  let selectedFormat = 'mp4';
  let selectedCodec = 'h264';

  const formats = [
    { value: 'mp4', label: 'MP4' },
    { value: 'avi', label: 'AVI' },
    { value: 'mkv', label: 'MKV' },
    { value: 'mov', label: 'MOV' },
    { value: 'webm', label: 'WebM' }
  ];

  const codecs = [
    { value: 'h264', label: 'H.264 (High Quality)' },
    { value: 'h265', label: 'H.265/HEVC (Smaller Size)' },
    { value: 'vp9', label: 'VP9 (WebM)' }
  ];

  async function handleApply() {
    try {
      await AddFormatConversion(selectedFormat, selectedCodec);
    } catch (error) {
      console.error('Failed to add format conversion:', error);
      alert(error);
    }
  }
</script>

<div class="format-conversion">
  <div class="control-group">
    <label for="format-select">Output Format:</label>
    <select id="format-select" bind:value={selectedFormat}>
      {#each formats as format}
        <option value={format.value}>{format.label}</option>
      {/each}
    </select>
  </div>

  <div class="control-group">
    <label for="codec-select">Video Codec:</label>
    <select id="codec-select" bind:value={selectedCodec}>
      {#each codecs as codec}
        <option value={codec.value}>{codec.label}</option>
      {/each}
    </select>
  </div>

  <button class="apply-btn" on:click={handleApply}>
    Add Conversion
  </button>

  <div class="info">
    <p>Convert video to selected format and codec. This will re-encode the video.</p>
  </div>
</div>

<style>
.format-conversion {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.control-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.control-group label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

select {
  padding: 8px 10px;
  background: var(--bg-primary);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
}

select:hover {
  border-color: #555;
}

select:focus {
  outline: none;
  border-color: var(--accent-blue);
}

.apply-btn {
  padding: 10px 16px;
  background: var(--accent-blue);
  color: white;
  border: none;
  border-radius: 4px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.apply-btn:hover {
  background: #50b4eb;
}

.info {
  padding: 10px;
  background: rgba(255, 255, 255, 0.03);
  border-radius: 4px;
  border: 1px solid var(--border-color);
}

.info p {
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.4;
  margin: 0;
}
</style>
