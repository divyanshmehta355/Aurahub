<script lang="ts">
	import { fetchApi } from '#lib/api';
	import { Plus, Trash2, CheckCircle, XCircle, Play, Square, ListVideo, ClipboardList } from 'lucide-svelte';

	let { playlists = [] } = $props<{ playlists: any[] }>();

	let playlistOption = $state('create_new');
	let selectedPlaylistId = $state('');
	let newPlaylistTitle = $state('');
	let category = $state('Other');
	let isShort = $state(false);

	let items = $state([
		{ id: Date.now() + '1', title: '', videoUrl: '', status: 'idle', progress: 0, error: '' },
		{ id: Date.now() + '2', title: '', videoUrl: '', status: 'idle', progress: 0, error: '' }
	]);

	let isRunning = $state(false);
	let currentProcessingIndex = $state(-1);
	let completedCount = $state(0);
	let createdPlaylist: any = $state(null);
	let abortFlag = $state(false);

	function addRow() {
		items = [...items, { id: Date.now().toString() + Math.random(), title: '', videoUrl: '', status: 'idle', progress: 0, error: '' }];
	}

	function removeRow(id: string) {
		if (items.length <= 1) return;
		items = items.filter(i => i.id !== id);
	}

	function extractTitle(url: string) {
		try {
			const filename = new URL(url).pathname.split("/").filter(Boolean).pop() || "";
			const withoutExt = filename.replace(/\.[a-zA-Z0-9]+$/, "");
			return decodeURIComponent(withoutExt).replace(/[-_]+/g, " ").trim();
		} catch {
			return "";
		}
	}

	function handleUrlBlur(item: any, url: string) {
		if (!item.title.trim() && url.trim()) {
			const title = extractTitle(url);
			if (title) item.title = title;
		}
	}

	async function pollRemote(remoteId: string, item: any) {
		let failedCount = 0;
		while (true) {
			if (abortFlag) throw new Error("Stopped by user");
			await new Promise(r => setTimeout(r, 4000));
			try {
				const statusRes = await fetchApi(`/videos/remote-upload/status?id=${remoteId}`);
				const statusData = statusRes[remoteId];

				if (statusData) {
					if (statusData.status === 'finished') {
						item.progress = 100;
						return statusData.linkid;
					} else if (statusData.status === 'error') {
						throw new Error(statusData.error_message || "Remote download error");
					} else {
						const loaded = statusData.bytes_loaded || 0;
						const total = statusData.bytes_total || 0;
						if (total > 0) item.progress = Math.floor((loaded / total) * 100);
					}
				}
				failedCount = 0;
			} catch (err) {
				failedCount++;
				if (failedCount > 5) throw new Error("Lost connection to status");
			}
		}
	}

	async function startUpload() {
		const validItems = items.filter(i => i.title.trim() && i.videoUrl.trim());
		if (validItems.length === 0) return alert("Please enter at least one valid video with Title and URL.");
		if (playlistOption === 'create_new' && !newPlaylistTitle.trim()) return alert("Enter a name for the new playlist.");

		isRunning = true;
		abortFlag = false;
		let targetPlaylistId = null;

		if (playlistOption === 'create_new') {
			try {
				const res = await fetchApi('/playlists', {
					method: 'POST',
					body: JSON.stringify({ title: newPlaylistTitle.trim(), isPublic: true })
				});
				targetPlaylistId = res.playlist?.id;
				createdPlaylist = res.playlist;
			} catch (e) {
				console.error(e);
			}
		} else if (playlistOption !== 'none') {
			targetPlaylistId = selectedPlaylistId;
		}

		let successCount = 0;
		for (let i = 0; i < items.length; i++) {
			if (abortFlag) break;
			const item = items[i];
			if (!item.title.trim() || !item.videoUrl.trim() || item.status === 'completed') continue;

			currentProcessingIndex = i;
			item.status = 'queuing';
			item.progress = 0;
			item.error = '';

			try {
				const startRes = await fetchApi('/videos/remote-upload/start', {
					method: 'POST',
					body: JSON.stringify({ videoUrl: item.videoUrl.trim() })
				});
				if (!startRes.id) throw new Error("Did not receive remote ID");

				item.status = 'downloading';
				const finalVideoId = await pollRemote(startRes.id, item);
				
				item.status = 'publishing';
				
				const finalData = new FormData();
				finalData.append('videoId', finalVideoId);
				finalData.append('title', item.title.trim());
				finalData.append('description', '');
				finalData.append('category', category);
				finalData.append('visibility', 'public');
				finalData.append('isShort', isShort.toString());
				if (targetPlaylistId) {
					finalData.append('playlistId', targetPlaylistId);
				}

				await fetchApi('/videos/create-record', {
					method: 'POST',
					body: finalData
				});

				item.status = 'completed';
				successCount++;
				completedCount = successCount;
			} catch (err: any) {
				item.status = 'error';
				item.error = err.message || "Failed";
			}
		}

		isRunning = false;
		currentProcessingIndex = -1;
	}

	function stopUpload() {
		abortFlag = true;
		isRunning = false;
	}

	let validCount = $derived(items.filter(i => i.title.trim() && i.videoUrl.trim()).length);
	let overallPercent = $derived(validCount > 0 ? Math.floor((completedCount / validCount) * 100) : 0);

	let showPasteModal = $state(false);
	let rawPastedText = $state('');

	function handleBatchPaste() {
		if (!rawPastedText.trim()) return;

		const lines = rawPastedText.split(/\r?\n/).map(line => line.trim()).filter(Boolean);
		const newRows: any[] = [];

		lines.forEach(line => {
			const urlMatches = [...line.matchAll(/https?:\/\/[^\s,;<>"']+/gi)];
			urlMatches.forEach((match, index) => {
				const parsedUrl = match[0].replace(/[)\]}]+$/, "");
				try {
					new URL(parsedUrl);
				} catch {
					return;
				}
				const titlePrefix = index === 0 ? line.slice(0, match.index).replace(/\s*(?:-|,|\|)\s*$/, "").trim() : "";
				const parsedTitle = titlePrefix || extractTitle(parsedUrl) || "Untitled Video";
				newRows.push({
					id: Date.now().toString() + Math.random().toString(36).substring(5),
					title: parsedTitle,
					videoUrl: parsedUrl,
					status: "idle",
					progress: 0,
					error: ""
				});
			});
		});

		if (newRows.length > 0) {
			if (items.length === 2 && !items[0].videoUrl && !items[1].videoUrl) {
				items = newRows;
			} else {
				items = [...items, ...newRows];
			}
		}
		rawPastedText = "";
		showPasteModal = false;
	}
</script>

<div class="space-y-6">
	<!-- Settings -->
	<div class="p-6 bg-muted/20 border rounded-2xl space-y-4">
		<h3 class="font-bold flex items-center gap-2 text-lg"><ListVideo class="w-5 h-5 text-muted-foreground" /> Shared Playlist Settings</h3>
		
		<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
			<div class="space-y-1.5">
				<label class="text-sm font-medium">Add to Playlist</label>
				<select bind:value={playlistOption} onchange={() => selectedPlaylistId = playlistOption !== 'create_new' && playlistOption !== 'none' ? playlistOption : ''} disabled={isRunning} class="w-full h-10 px-3 rounded-md border bg-background text-sm">
					<option value="create_new">+ Create New (Recommended)</option>
					{#if playlists.length > 0}
						<optgroup label="Existing Playlists">
							{#each playlists as p}
								<option value={p.id || p._id}>{p.title}</option>
							{/each}
						</optgroup>
					{/if}
					<option value="none">None (Individual Videos)</option>
				</select>
			</div>

			<div class="space-y-1.5">
				<label class="text-sm font-medium">Category</label>
				<select bind:value={category} disabled={isRunning} class="w-full h-10 px-3 rounded-md border bg-background text-sm">
					<option value="Other">Other</option>
					<option value="Education">Education</option>
					<option value="Gaming">Gaming</option>
					<option value="Music">Music</option>
				</select>
			</div>
		</div>

		{#if playlistOption === 'create_new'}
			<div class="p-4 bg-background border rounded-lg space-y-2">
				<label class="text-sm font-medium">New Playlist Name *</label>
				<input type="text" bind:value={newPlaylistTitle} disabled={isRunning} placeholder="e.g. Next.js Masterclass" class="w-full h-10 px-3 rounded-md border bg-muted/50 text-sm focus:ring-primary" />
			</div>
		{/if}
	</div>

	<!-- Queue -->
	<div class="space-y-3">
		<div class="flex justify-between items-end">
			<div>
				<h3 class="font-bold text-lg">Video Queue ({items.length})</h3>
				<p class="text-xs text-muted-foreground">Enter video Title and Remote MP4 URLs.</p>
			</div>
			<button 
				onclick={() => showPasteModal = true}
				disabled={isRunning}
				class="inline-flex items-center gap-1.5 h-8 px-3 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-xs font-semibold disabled:opacity-50"
			>
				<ClipboardList class="w-3.5 h-3.5" />
				Paste Batch URLs
			</button>
		</div>

		<div class="space-y-3 max-h-[50vh] overflow-y-auto pr-1">
			{#each items as item, i (item.id)}
				<div class="p-4 border rounded-xl bg-card relative {currentProcessingIndex === i ? 'ring-2 ring-primary/20 bg-muted/30' : item.status === 'completed' ? 'border-green-500/30 bg-green-500/5' : ''}">
					<div class="flex justify-between items-center mb-3">
						<span class="text-xs font-semibold px-2 py-1 bg-muted rounded-full text-muted-foreground">Video #{i + 1}</span>
						
						<div class="flex items-center gap-2">
							{#if item.status === 'idle'}
								<span class="text-[11px] font-medium text-muted-foreground">Ready</span>
							{:else if item.status === 'queuing'}
								<span class="text-[11px] font-medium text-muted-foreground flex items-center gap-1"><div class="w-2 h-2 border border-current rounded-full animate-spin"></div> Queuing...</span>
							{:else if item.status === 'downloading'}
								<span class="text-[11px] font-semibold text-primary flex items-center gap-1"><div class="w-2 h-2 border border-current rounded-full animate-spin"></div> Downloading {item.progress}%</span>
							{:else if item.status === 'publishing'}
								<span class="text-[11px] font-semibold text-primary flex items-center gap-1"><div class="w-2 h-2 border border-current rounded-full animate-spin"></div> Publishing...</span>
							{:else if item.status === 'completed'}
								<span class="text-[11px] font-bold text-green-600 flex items-center gap-1"><CheckCircle class="w-3 h-3" /> Completed</span>
							{:else if item.status === 'error'}
								<span class="text-[11px] font-bold text-red-500 flex items-center gap-1" title={item.error}><XCircle class="w-3 h-3" /> Failed</span>
							{/if}

							{#if !isRunning && items.length > 1}
								<button onclick={() => removeRow(item.id)} class="text-muted-foreground hover:text-red-500 transition-colors p-1" title="Remove"><Trash2 class="w-4 h-4" /></button>
							{/if}
						</div>
					</div>

					<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
						<input type="text" bind:value={item.title} disabled={isRunning} placeholder="Video Title *" class="w-full h-9 px-3 rounded-md border text-sm focus:ring-primary bg-background" />
						<input type="url" bind:value={item.videoUrl} onblur={(e) => handleUrlBlur(item, e.currentTarget.value)} disabled={isRunning} placeholder="Remote URL *" class="w-full h-9 px-3 rounded-md border text-sm font-mono focus:ring-primary bg-background" />
					</div>

					{#if currentProcessingIndex === i && item.status === 'downloading'}
						<div class="mt-3 w-full h-1.5 bg-secondary rounded-full overflow-hidden">
							<div class="h-full bg-primary transition-all duration-300" style="width: {item.progress}%"></div>
						</div>
					{/if}
					{#if item.error}
						<p class="mt-2 text-xs text-red-500">{item.error}</p>
					{/if}
				</div>
			{/each}
		</div>

		{#if !isRunning}
			<button onclick={addRow} class="w-full h-10 border-2 border-dashed rounded-lg flex items-center justify-center gap-2 text-sm font-semibold hover:bg-muted/50 transition-colors text-muted-foreground">
				<Plus class="w-4 h-4" /> Add Another Video
			</button>
		{/if}
	</div>

	{#if isRunning}
		<div class="p-4 bg-muted/50 border rounded-xl space-y-2">
			<div class="flex justify-between text-xs font-bold">
				<span>Overall Progress</span>
				<span>{completedCount} / {validCount} ({overallPercent}%)</span>
			</div>
			<div class="w-full h-2.5 bg-secondary rounded-full overflow-hidden">
				<div class="h-full bg-primary transition-all duration-500" style="width: {overallPercent}%"></div>
			</div>
			<p class="text-xs text-center text-muted-foreground animate-pulse">Processing... please keep this tab open.</p>
		</div>
	{/if}

	{#if !isRunning && createdPlaylist && completedCount > 0}
		<div class="p-4 bg-green-50 text-green-700 rounded-xl border border-green-200 flex items-center justify-between">
			<span class="text-sm font-semibold flex items-center gap-2"><CheckCircle class="w-5 h-5" /> Playlist "{createdPlaylist.title || newPlaylistTitle}" is ready!</span>
			<a href={`/playlist/${createdPlaylist.id || createdPlaylist._id}`} class="px-4 py-1.5 bg-white rounded-md shadow-sm text-sm font-medium text-green-700 hover:bg-green-50">View</a>
		</div>
	{/if}

	<div class="pt-2">
		{#if isRunning}
			<button onclick={stopUpload} class="w-full h-12 bg-red-500 text-white font-bold rounded-lg hover:bg-red-600 transition-colors flex justify-center items-center gap-2">
				<Square class="w-4 h-4 fill-current" /> Stop Uploads
			</button>
		{:else}
			<button onclick={startUpload} disabled={validCount === 0} class="w-full h-12 bg-primary text-primary-foreground font-bold rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50 flex justify-center items-center gap-2">
				<Play class="w-4 h-4 fill-current" /> Start Playlist Upload {validCount > 0 ? `(${validCount})` : ''}
			</button>
		{/if}
	</div>
</div>

{#if showPasteModal}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
		<div class="bg-card w-full max-w-lg rounded-xl shadow-xl overflow-hidden animate-in fade-in zoom-in-95 duration-200">
			<div class="px-6 py-4 border-b">
				<h3 class="text-lg font-semibold flex items-center gap-2">
					<ClipboardList class="w-5 h-5 text-muted-foreground" />
					Paste Multiple Video URLs
				</h3>
				<p class="text-sm text-muted-foreground mt-1">
					Paste URLs separated by spaces, commas, or new lines. Titles are taken from each URL filename, or use <code>Title - URL</code> to set one:
				</p>
			</div>
			
			<div class="p-6">
				<textarea 
					rows="8"
					bind:value={rawPastedText}
					placeholder="https://example.com/episode-1.mp4, https://example.com/episode-2.mp4&#10;Episode 3 - https://example.com/episode-3.mp4"
					class="w-full rounded-md border bg-muted/50 p-3 text-sm font-mono focus:ring-2 focus:ring-primary/20 resize-none outline-none custom-scrollbar"
				></textarea>
			</div>
			
			<div class="px-6 py-4 bg-muted/30 border-t flex justify-end gap-2">
				<button 
					onclick={() => showPasteModal = false}
					class="px-4 py-2 rounded-md hover:bg-muted text-sm font-medium transition-colors"
				>
					Cancel
				</button>
				<button 
					onclick={handleBatchPaste}
					disabled={!rawPastedText.trim()}
					class="px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm font-medium transition-colors disabled:opacity-50"
				>
					Populate Videos
				</button>
			</div>
		</div>
	</div>
{/if}
