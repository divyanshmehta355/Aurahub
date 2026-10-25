<script lang="ts">
	import { fetchApi } from '#lib/api';
	import {
		Plus,
		Trash2,
		CheckCircle,
		CheckCircle2,
		XCircle,
		Play,
		Square,
		ListVideo,
		FileJson,
		UploadCloud,
		AlertCircle,
		X
	} from 'lucide-svelte';

	let { playlists = [] } = $props<{ playlists: any[] }>();

	let playlistOption = $state('create_new');
	let selectedPlaylistId = $state('');
	let newPlaylistTitle = $state('');
	let category = $state('Other');
	let isShort = $state(false);
	let isAdult = $state(false);

	let items = $state([
		{ id: Date.now() + '1', title: '', videoUrl: '', status: 'idle', progress: 0, error: '' },
		{ id: Date.now() + '2', title: '', videoUrl: '', status: 'idle', progress: 0, error: '' }
	]);

	let isRunning = $state(false);
	let currentProcessingIndex = $state(-1);
	let completedCount = $state(0);
	let createdPlaylist: any = $state(null);
	let abortFlag = $state(false);

	$effect(() => {
		if (typeof window === 'undefined') return;

		const handleBatchProgress = (e: any) => {
			const data = e.detail;
			if (!data) return;

			const targetItem = items.find((i) => i.title.trim() === data.title?.trim());
			if (targetItem) {
				if (data.status === 'completed') {
					targetItem.status = 'completed';
					targetItem.progress = 100;
					completedCount = items.filter((i) => i.status === 'completed').length;
				} else if (data.status === 'downloading') {
					targetItem.status = 'downloading';
					targetItem.progress = data.progress || 50;
				} else if (data.status === 'error') {
					targetItem.status = 'error';
					targetItem.error = data.error || 'Failed';
				}
			}
		};

		window.addEventListener('batch_upload_progress', handleBatchProgress);
		return () => {
			window.removeEventListener('batch_upload_progress', handleBatchProgress);
		};
	});

	function addRow() {
		items = [
			...items,
			{
				id: Date.now().toString() + Math.random(),
				title: '',
				videoUrl: '',
				status: 'idle',
				progress: 0,
				error: ''
			}
		];
	}

	function removeRow(id: string) {
		if (items.length <= 1) return;
		items = items.filter((i) => i.id !== id);
	}

	function extractTitle(url: string) {
		try {
			const filename = new URL(url).pathname.split('/').filter(Boolean).pop() || '';
			const withoutExt = filename.replace(/\.[a-zA-Z0-9]+$/, '');
			return decodeURIComponent(withoutExt).replace(/[-_]+/g, ' ').trim();
		} catch {
			return '';
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
			if (abortFlag) throw new Error('Stopped by user');
			await new Promise((r) => setTimeout(r, 4000));
			try {
				const statusRes = await fetchApi(`/videos/remote-upload/status?id=${remoteId}`);
				const statusData = statusRes[remoteId];

				if (statusData) {
					if (statusData.status === 'finished') {
						item.progress = 100;
						return statusData.linkid;
					} else if (statusData.status === 'error') {
						throw new Error(statusData.error_message || 'Remote download error');
					} else {
						const loaded = statusData.bytes_loaded || 0;
						const total = statusData.bytes_total || 0;
						if (total > 0) item.progress = Math.floor((loaded / total) * 100);
					}
				}
				failedCount = 0;
			} catch (err) {
				failedCount++;
				if (failedCount > 5) throw new Error('Lost connection to status');
			}
		}
	}

	function handleBatchProgress(e: any) {
		const data = e.detail;
		if (!data) return;

		const item = items.find((i) => i.title.trim() === data.title?.trim());
		if (item) {
			if (data.status) item.status = data.status;
			if (typeof data.progress === 'number') item.progress = data.progress;
			if (data.error) item.error = data.error;
			if (data.status === 'completed') {
				completedCount = items.filter((i) => i.status === 'completed').length;
			}
		}
	}

	$effect(() => {
		if (typeof window !== 'undefined') {
			window.addEventListener('batch_upload_progress', handleBatchProgress);
			return () => {
				window.removeEventListener('batch_upload_progress', handleBatchProgress);
			};
		}
	});

	async function processSingleItem(item: any, targetPlaylistId: string | null) {
		item.status = 'queuing';
		item.progress = 0;
		item.error = '';

		try {
			const startRes = await fetchApi('/videos/remote-upload/start', {
				method: 'POST',
				body: JSON.stringify({ videoUrl: item.videoUrl.trim() })
			});
			if (!startRes.id) throw new Error('Did not receive remote ID');

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
			finalData.append('isAdult', isAdult.toString());
			if (targetPlaylistId) {
				finalData.append('playlistId', targetPlaylistId);
			}

			await fetchApi('/videos/create-record', {
				method: 'POST',
				body: finalData
			});

			item.status = 'completed';
			completedCount = items.filter((i) => i.status === 'completed').length;
		} catch (err: any) {
			item.status = 'error';
			item.error = err.message || 'Failed';
		}
	}

	async function startUpload() {
		const validItems = items.filter((i) => i.title.trim() && i.videoUrl.trim());
		if (validItems.length === 0)
			return alert('Please enter at least one valid video with Title and URL.');
		if (playlistOption === 'create_new' && !newPlaylistTitle.trim())
			return alert('Enter a name for the new playlist.');

		isRunning = true;
		abortFlag = false;
		let targetPlaylistId: string | null = null;

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

		// 1. Production Kafka Pipeline: Dispatch entire batch to background worker
		try {
			items.forEach((item) => {
				if (item.title.trim() && item.videoUrl.trim() && item.status !== 'completed') {
					item.status = 'queuing';
					item.progress = 5;
					item.error = '';
				}
			});

			const batchPayload = {
				playlistId: targetPlaylistId,
				category,
				visibility: 'public',
				isShort,
				isAdult,
				items: validItems.map((i) => ({ title: i.title.trim(), videoUrl: i.videoUrl.trim() }))
			};

			const batchRes = await fetchApi('/videos/remote-upload/batch', {
				method: 'POST',
				body: JSON.stringify(batchPayload)
			});

			if (batchRes && batchRes.enqueuedCount > 0) {
				// Successfully enqueued to Kafka! The server worker handles remote download & polling in parallel.
				isRunning = false;
				items.forEach((item) => {
					if (item.status === 'queuing') {
						item.status = 'downloading';
						item.progress = 10;
					}
				});
				return;
			}
		} catch (batchErr) {
			console.warn('Kafka batch endpoint unavailable, falling back to parallel worker pool:', batchErr);
		}

		// 2. Parallel Worker Pool Fallback (Concurrency = 3)
		const CONCURRENCY_LIMIT = 3;
		const queue = items.filter((i) => i.title.trim() && i.videoUrl.trim() && i.status !== 'completed');

		async function worker() {
			while (queue.length > 0 && !abortFlag) {
				const item = queue.shift();
				if (!item) break;
				await processSingleItem(item, targetPlaylistId);
			}
		}

		const workers = Array.from(
			{ length: Math.min(CONCURRENCY_LIMIT, queue.length) },
			() => worker()
		);

		await Promise.all(workers);
		isRunning = false;
	}

	function stopUpload() {
		abortFlag = true;
		isRunning = false;
	}

	let validCount = $derived(items.filter((i) => i.title.trim() && i.videoUrl.trim()).length);
	let overallPercent = $derived(
		validCount > 0 ? Math.floor((completedCount / validCount) * 100) : 0
	);

	let showPasteModal = $state(false);
	let rawPastedText = $state('');
	let cleanTitlePrefix = $state(true);
	let jsonFileInput = $state<HTMLInputElement | null>(null);

	let parsedData = $derived.by(() => {
		const text = rawPastedText.trim();
		if (!text) return { items: [], count: 0, error: '', format: 'empty' };

		// Check if input starts with JSON array or object
		if (text.startsWith('[') || text.startsWith('{')) {
			try {
				const parsed = JSON.parse(text);
				let list: any[] = [];
				if (Array.isArray(parsed)) {
					list = parsed;
				} else if (parsed && typeof parsed === 'object') {
					if (Array.isArray(parsed.videos)) list = parsed.videos;
					else if (Array.isArray(parsed.items)) list = parsed.items;
					else if (Array.isArray(parsed.data)) list = parsed.data;
					else list = [parsed];
				}

				const rows: any[] = [];
				list.forEach((entry, idx) => {
					if (!entry || typeof entry !== 'object') return;

					// Extract URL from downloadLink, videoUrl, url, or link
					const rawUrl = entry.downloadLink || entry.videoUrl || entry.url || entry.link || '';
					const urlStr = typeof rawUrl === 'string' ? rawUrl.trim() : '';

					// Extract title from movieTitle, title, name, etc.
					let rawTitle = entry.movieTitle || entry.title || entry.name || '';
					if (typeof rawTitle !== 'string') rawTitle = '';
					rawTitle = rawTitle.trim();

					if (cleanTitlePrefix) {
						// Strip leading "Download " or "Download: " prefix
						rawTitle = rawTitle.replace(/^download\s*[:\-\s]*/i, '').trim();
					}

					const finalTitle = rawTitle || extractTitle(urlStr) || `Video #${idx + 1}`;

					if (urlStr) {
						rows.push({
							id: (entry.id ? String(entry.id) : Date.now().toString()) + Math.random().toString(36).substring(5),
							title: finalTitle,
							videoUrl: urlStr,
							status: 'idle',
							progress: 0,
							error: '',
							quality: entry.qualitySelected || '',
							server: entry.serverName || ''
						});
					}
				});

				if (rows.length === 0) {
					return {
						items: [],
						count: 0,
						error: 'Valid JSON, but no items with "downloadLink" or "videoUrl" were found.',
						format: 'json'
					};
				}
				return { items: rows, count: rows.length, error: '', format: 'json' };
			} catch (e: any) {
				return { items: [], count: 0, error: e.message || 'Invalid JSON syntax', format: 'invalid_json' };
			}
		}

		// Fallback text parser (for line-by-line URLs if pasted)
		const lines = text.split(/\r?\n/).map((l) => l.trim()).filter(Boolean);
		const fallbackRows: any[] = [];
		lines.forEach((line) => {
			const urlMatches = [...line.matchAll(/https?:\/\/[^\s,;<>"']+/gi)];
			urlMatches.forEach((match, index) => {
				const parsedUrl = match[0].replace(/[)\]}]+$/, '');
				try {
					new URL(parsedUrl);
				} catch {
					return;
				}
				const titlePrefix =
					index === 0 ? line.slice(0, match.index).replace(/\s*(?:-|,|\|)\s*$/, '').trim() : '';
				let parsedTitle = titlePrefix || extractTitle(parsedUrl) || 'Untitled Video';
				if (cleanTitlePrefix) {
					parsedTitle = parsedTitle.replace(/^download\s*[:\-\s]*/i, '').trim();
				}
				fallbackRows.push({
					id: Date.now().toString() + Math.random().toString(36).substring(5),
					title: parsedTitle,
					videoUrl: parsedUrl,
					status: 'idle',
					progress: 0,
					error: ''
				});
			});
		});

		if (fallbackRows.length > 0) {
			return { items: fallbackRows, count: fallbackRows.length, error: '', format: 'text' };
		}

		return { items: [], count: 0, error: 'No valid video URLs found.', format: 'unknown' };
	});

	function handleBatchPaste() {
		if (parsedData.items.length === 0) return;

		if (items.length === 2 && !items[0].videoUrl && !items[1].videoUrl) {
			items = parsedData.items;
		} else {
			items = [...items, ...parsedData.items];
		}
		rawPastedText = '';
		showPasteModal = false;
	}

	function handleJsonFileUpload(e: Event) {
		const target = e.target as HTMLInputElement;
		if (target.files && target.files.length > 0) {
			const file = target.files[0];
			const reader = new FileReader();
			reader.onload = (event) => {
				const content = event.target?.result;
				if (typeof content === 'string') {
					rawPastedText = content;
				}
			};
			reader.readAsText(file);
			target.value = '';
		}
	}
</script>

<div class="space-y-6">
	<!-- Settings -->
	<div class="bg-muted/20 space-y-4 rounded-2xl border p-6">
		<h3 class="flex items-center gap-2 text-lg font-bold">
			<ListVideo class="text-muted-foreground h-5 w-5" /> Shared Playlist Settings
		</h3>

		<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
			<div class="space-y-1.5">
				<label for="add-to-playlist-select" class="text-sm font-medium">Add to Playlist</label>
				<select
					id="add-to-playlist-select"
					bind:value={playlistOption}
					onchange={() =>
						(selectedPlaylistId =
							playlistOption !== 'create_new' && playlistOption !== 'none' ? playlistOption : '')}
					disabled={isRunning}
					class="h-10 w-full rounded-md border bg-background px-3 text-sm"
				>
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
				<label for="upload-category-select" class="text-sm font-medium">Category</label>
				<select
					id="upload-category-select"
					bind:value={category}
					disabled={isRunning}
					class="h-10 w-full rounded-md border bg-background px-3 text-sm"
				>
					<option value="Other">Other</option>
					<option value="Education">Education</option>
					<option value="Gaming">Gaming</option>
					<option value="Music">Music</option>
				</select>
			</div>

			<div class="flex flex-col justify-end space-y-1.5 pb-1.5">
				<label class="flex cursor-pointer items-center gap-2 text-sm font-medium">
					<input
						type="checkbox"
						bind:checked={isAdult}
						disabled={isRunning}
						class="border-input h-4 w-4 rounded text-primary focus:ring-primary"
					/>
					Contains Adult Content
				</label>
			</div>
		</div>

		{#if playlistOption === 'create_new'}
			<div class="space-y-2 rounded-lg border bg-background p-4">
				<label for="new-playlist-name-input" class="text-sm font-medium">New Playlist Name *</label>
				<input
					id="new-playlist-name-input"
					type="text"
					bind:value={newPlaylistTitle}
					disabled={isRunning}
					placeholder="e.g. Next.js Masterclass"
					class="bg-muted/50 h-10 w-full rounded-md border px-3 text-sm focus:ring-primary"
				/>
			</div>
		{/if}
	</div>

	<!-- Queue -->
	<div class="space-y-3">
		<div class="flex items-end justify-between">
			<div>
				<h3 class="text-lg font-bold">Video Queue ({items.length})</h3>
				<p class="text-muted-foreground text-xs">Enter video Title and Remote URLs, or import in bulk via JSON.</p>
			</div>
			<button
				onclick={() => (showPasteModal = true)}
				disabled={isRunning}
				class="bg-secondary text-secondary-foreground hover:bg-secondary/80 inline-flex h-8 items-center gap-1.5 rounded-md px-3 text-xs font-semibold transition-colors disabled:opacity-50"
			>
				<FileJson class="h-3.5 w-3.5" />
				Import JSON Batch
			</button>
		</div>

		<div class="max-h-[50vh] space-y-3 overflow-y-auto pr-1">
			{#each items as item, i (item.id)}
				<div
					class="bg-card relative rounded-xl border p-4 {currentProcessingIndex === i
						? 'bg-muted/30 ring-2 ring-primary/20'
						: item.status === 'completed'
							? 'border-green-500/30 bg-green-500/5'
							: ''}"
				>
					<div class="mb-3 flex items-center justify-between">
						<div class="flex items-center gap-2">
							<span
								class="bg-muted text-muted-foreground rounded-full px-2 py-1 text-xs font-semibold"
								>Video #{i + 1}</span
							>
							{#if (item as any).quality}
								<span
									class="bg-primary/15 text-primary rounded-full px-2 py-0.5 text-[11px] font-bold"
									>{(item as any).quality}</span
								>
							{/if}
						</div>

						<div class="flex items-center gap-2">
							{#if item.status === 'idle'}
								<span class="text-muted-foreground text-[11px] font-medium">Ready</span>
							{:else if item.status === 'queuing'}
								<span class="text-muted-foreground flex items-center gap-1 text-[11px] font-medium"
									><div class="h-2 w-2 animate-spin rounded-full border border-current"></div>
									Queuing...</span
								>
							{:else if item.status === 'downloading'}
								<span class="flex items-center gap-1 text-[11px] font-semibold text-primary"
									><div class="h-2 w-2 animate-spin rounded-full border border-current"></div>
									Downloading {item.progress}%</span
								>
							{:else if item.status === 'publishing'}
								<span class="flex items-center gap-1 text-[11px] font-semibold text-primary"
									><div class="h-2 w-2 animate-spin rounded-full border border-current"></div>
									Publishing...</span
								>
							{:else if item.status === 'completed'}
								<span class="flex items-center gap-1 text-[11px] font-bold text-green-600"
									><CheckCircle class="h-3 w-3" /> Completed</span
								>
							{:else if item.status === 'error'}
								<span
									class="flex items-center gap-1 text-[11px] font-bold text-red-500"
									title={item.error}><XCircle class="h-3 w-3" /> Failed</span
								>
							{/if}

							{#if !isRunning && items.length > 1}
								<button
									onclick={() => removeRow(item.id)}
									class="text-muted-foreground p-1 transition-colors hover:text-red-500"
									title="Remove"><Trash2 class="h-4 w-4" /></button
								>
							{/if}
						</div>
					</div>

					<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
						<input
							type="text"
							bind:value={item.title}
							disabled={isRunning}
							placeholder="Video Title *"
							class="h-9 w-full rounded-md border bg-background px-3 text-sm focus:ring-primary"
						/>
						<input
							type="url"
							bind:value={item.videoUrl}
							onblur={(e) => handleUrlBlur(item, e.currentTarget.value)}
							disabled={isRunning}
							placeholder="Remote URL *"
							class="h-9 w-full rounded-md border bg-background px-3 font-mono text-sm focus:ring-primary"
						/>
					</div>

					{#if currentProcessingIndex === i && item.status === 'downloading'}
						<div class="bg-secondary mt-3 h-1.5 w-full overflow-hidden rounded-full">
							<div
								class="h-full bg-primary transition-all duration-300"
								style="width: {item.progress}%"
							></div>
						</div>
					{/if}
					{#if item.error}
						<p class="mt-2 text-xs text-red-500">{item.error}</p>
					{/if}
				</div>
			{/each}
		</div>

		{#if !isRunning}
			<button
				onclick={addRow}
				class="hover:bg-muted/50 text-muted-foreground flex h-10 w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed text-sm font-semibold transition-colors"
			>
				<Plus class="h-4 w-4" /> Add Another Video
			</button>
		{/if}
	</div>

	{#if isRunning}
		<div class="bg-muted/50 space-y-2 rounded-xl border p-4">
			<div class="flex justify-between text-xs font-bold">
				<span>Overall Progress</span>
				<span>{completedCount} / {validCount} ({overallPercent}%)</span>
			</div>
			<div class="bg-secondary h-2.5 w-full overflow-hidden rounded-full">
				<div
					class="h-full bg-primary transition-all duration-500"
					style="width: {overallPercent}%"
				></div>
			</div>
			<p class="text-muted-foreground animate-pulse text-center text-xs">
				Processing... please keep this tab open.
			</p>
		</div>
	{/if}

	{#if !isRunning && createdPlaylist && completedCount > 0}
		<div
			class="flex items-center justify-between rounded-xl border border-green-200 bg-green-50 p-4 text-green-700"
		>
			<span class="flex items-center gap-2 text-sm font-semibold"
				><CheckCircle class="h-5 w-5" /> Playlist "{createdPlaylist.title || newPlaylistTitle}" is
				ready!</span
			>
			<a
				href={`/playlist/${createdPlaylist.id || createdPlaylist._id}`}
				class="rounded-md bg-white px-4 py-1.5 text-sm font-medium text-green-700 shadow-sm hover:bg-green-50"
				>View</a
			>
		</div>
	{/if}

	<div class="pt-2">
		{#if isRunning}
			<button
				onclick={stopUpload}
				class="flex h-12 w-full items-center justify-center gap-2 rounded-lg bg-red-500 font-bold text-white transition-colors hover:bg-red-600"
			>
				<Square class="h-4 w-4 fill-current" /> Stop Uploads
			</button>
		{:else}
			<button
				onclick={startUpload}
				disabled={validCount === 0}
				class="text-primary-foreground flex h-12 w-full items-center justify-center gap-2 rounded-lg bg-primary font-bold transition-colors hover:bg-primary/90 disabled:opacity-50"
			>
				<Play class="h-4 w-4 fill-current" /> Start Playlist Upload {validCount > 0
					? `(${validCount})`
					: ''}
			</button>
		{/if}
	</div>
</div>

{#if showPasteModal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-sm"
		role="dialog"
		aria-modal="true"
	>
		<div
			class="bg-card animate-in fade-in zoom-in-95 w-full max-w-2xl overflow-hidden rounded-2xl border shadow-2xl duration-200"
		>
			<div class="flex items-center justify-between border-b px-6 py-4">
				<div class="flex items-center gap-2.5">
					<div class="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
						<FileJson class="h-5 w-5" />
					</div>
					<div>
						<h3 class="text-lg font-bold">Import Videos from JSON</h3>
						<p class="text-muted-foreground text-xs">
							Maps <code>movieTitle</code> &rarr; Title and <code>downloadLink</code> &rarr; Video URL.
						</p>
					</div>
				</div>
				<button
					type="button"
					onclick={() => (showPasteModal = false)}
					class="text-muted-foreground hover:text-foreground rounded-md p-1 transition-colors"
				>
					<X class="h-5 w-5" />
				</button>
			</div>

			<div class="space-y-4 p-6">
				<!-- File upload & controls toolbar -->
				<div class="flex flex-wrap items-center justify-between gap-3">
					<input
						type="file"
						bind:this={jsonFileInput}
						accept=".json,application/json"
						onchange={handleJsonFileUpload}
						class="hidden"
					/>
					<button
						type="button"
						onclick={() => jsonFileInput?.click()}
						class="hover:bg-muted/80 text-foreground inline-flex h-8 items-center gap-1.5 rounded-md border bg-muted/40 px-3 text-xs font-semibold transition-colors"
					>
						<UploadCloud class="h-3.5 w-3.5" />
						Upload .json File
					</button>

					<div class="flex items-center gap-4">
						<label class="flex cursor-pointer items-center gap-2 text-xs font-medium text-muted-foreground select-none">
							<input
								type="checkbox"
								bind:checked={cleanTitlePrefix}
								class="border-input h-3.5 w-3.5 rounded text-primary focus:ring-primary"
							/>
							<span>Clean "Download" prefix</span>
						</label>

						{#if rawPastedText.trim()}
							<button
								type="button"
								onclick={() => (rawPastedText = '')}
								class="text-muted-foreground hover:text-destructive text-xs font-medium transition-colors"
							>
								Clear
							</button>
						{/if}
					</div>
				</div>

				<!-- JSON Textarea -->
				<div class="relative">
					<textarea
						rows="9"
						bind:value={rawPastedText}
						placeholder={`[\n  {\n    "movieTitle": "Download VisionQuest S01E01 1080p...",\n    "downloadLink": "https://pixeldrain.dev/u/negn6f",\n    "qualitySelected": "1080p"\n  }\n]`}
						class="bg-muted/40 custom-scrollbar w-full resize-none rounded-xl border p-3.5 font-mono text-xs leading-relaxed outline-none focus:ring-2 focus:ring-primary/20"
					></textarea>
				</div>

				<!-- Live Parser Feedback & Preview -->
				{#if parsedData.format === 'json' && parsedData.count > 0}
					<div class="rounded-xl border border-emerald-500/30 bg-emerald-500/10 p-3 text-xs">
						<div class="mb-2 flex items-center justify-between font-semibold text-emerald-600 dark:text-emerald-400">
							<div class="flex items-center gap-1.5">
								<CheckCircle2 class="h-4 w-4 shrink-0" />
								<span>Detected {parsedData.count} valid video{parsedData.count === 1 ? '' : 's'}</span>
							</div>
							<span class="text-[11px] opacity-80">JSON array parsed</span>
						</div>

						<div class="custom-scrollbar max-h-32 space-y-1.5 overflow-y-auto rounded-lg border border-emerald-500/20 bg-background/80 p-2 font-mono">
							{#each parsedData.items.slice(0, 5) as preview, idx}
								<div class="flex items-center justify-between gap-2 border-b border-border/40 py-1 last:border-0">
									<div class="flex min-w-0 items-center gap-2">
										<span class="text-muted-foreground shrink-0 text-[10px]">{idx + 1}.</span>
										<span class="truncate font-medium text-foreground">{preview.title}</span>
									</div>
									{#if (preview as any).quality}
										<span class="shrink-0 rounded bg-primary/20 px-1.5 py-0.5 text-[10px] font-bold text-primary font-sans">
											{(preview as any).quality}
										</span>
									{/if}
								</div>
							{/each}
							{#if parsedData.items.length > 5}
								<p class="pt-1 text-center font-sans text-[11px] text-muted-foreground italic">
									+ {parsedData.items.length - 5} more video{parsedData.items.length - 5 === 1 ? '' : 's'}
								</p>
							{/if}
						</div>
					</div>
				{:else if parsedData.error && rawPastedText.trim()}
					<div class="flex items-center gap-2 rounded-xl border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive">
						<AlertCircle class="h-4 w-4 shrink-0" />
						<span>{parsedData.error}</span>
					</div>
				{:else if parsedData.format === 'text' && parsedData.count > 0}
					<div class="flex items-center justify-between rounded-xl border border-blue-500/30 bg-blue-500/10 p-3 text-xs text-blue-600 dark:text-blue-400">
						<div class="flex items-center gap-1.5">
							<CheckCircle2 class="h-4 w-4 shrink-0" />
							<span>Detected {parsedData.count} URL{parsedData.count === 1 ? '' : 's'} from plain text</span>
						</div>
					</div>
				{/if}
			</div>

			<div class="bg-muted/30 flex items-center justify-between border-t px-6 py-4">
				<p class="text-muted-foreground text-xs">
					{#if parsedData.count > 0}
						Ready to populate {parsedData.count} video{parsedData.count === 1 ? '' : 's'}.
					{:else}
						Paste JSON array or upload file to continue.
					{/if}
				</p>
				<div class="flex items-center gap-2">
					<button
						type="button"
						onclick={() => (showPasteModal = false)}
						class="hover:bg-muted rounded-lg px-4 py-2 text-xs font-semibold transition-colors"
					>
						Cancel
					</button>
					<button
						type="button"
						onclick={handleBatchPaste}
						disabled={parsedData.count === 0}
						class="text-primary-foreground flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-xs font-semibold shadow-sm transition-all hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
					>
						<Plus class="h-3.5 w-3.5" />
						Import {parsedData.count > 0 ? `(${parsedData.count}) Videos` : 'Videos'}
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}
