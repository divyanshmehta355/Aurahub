<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchApi } from '#lib/api';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import { Clock, Trash2, X } from 'lucide-svelte';

	let videos: any[] = $state([]);
	let isLoading = $state(true);
	let error = $state('');

	async function loadWatchLater() {
		isLoading = true;
		error = '';
		try {
			const res = await fetchApi('/user/watch-later');
			videos = res.videos || res.items || [];
		} catch (err: any) {
			error = err.message || 'Failed to load watch later list.';
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		loadWatchLater();
	});

	async function clearAll() {
		if (!confirm('Are you sure you want to clear your entire Watch Later list?')) return;
		try {
			await fetchApi('/user/watch-later', { method: 'DELETE' });
			videos = [];
		} catch (err: any) {
			alert('Failed to clear list');
		}
	}

	async function removeVideo(e: Event, videoId: string | number) {
		e.preventDefault(); // Prevent link click
		try {
			// Check if backend uses generic delete or specific remove endpoint
			// We'll assume POST to toggle or specific delete
			await fetchApi('/user/watch-later', { 
				method: 'POST',
				body: JSON.stringify({ videoId })
			});
			videos = videos.filter(v => (v.video?.id || v.id) !== videoId);
		} catch (err: any) {
			alert('Failed to remove video');
		}
	}

	function formatDuration(seconds: number) {
		if (!seconds) return '0:00';
		const m = Math.floor(seconds / 60);
		const s = Math.floor(seconds % 60);
		return `${m}:${s.toString().padStart(2, '0')}`;
	}
</script>

<svelte:head>
	<title>Watch Later - Aurahub</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold tracking-tight">Watch Later</h1>
		{#if videos.length > 0}
			<button 
				onclick={clearAll}
				class="flex items-center gap-2 text-sm font-medium text-red-500 hover:text-red-600 transition-colors bg-red-50 dark:bg-red-950/30 px-4 py-2 rounded-lg"
			>
				<Trash2 class="w-4 h-4" />
				Clear list
			</button>
		{/if}
	</div>

	{#if isLoading}
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each Array(4) as _}
				<div class="flex animate-pulse flex-col space-y-3">
					<div class="bg-muted aspect-video w-full rounded-xl"></div>
				</div>
			{/each}
		</div>
	{:else if error}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="rounded-xl bg-red-100 p-4 text-red-600 dark:bg-red-900/30 dark:text-red-400">
				<p>{error}</p>
			</div>
		</div>
	{:else if videos.length === 0}
		<div class="border-muted-foreground/20 flex flex-col items-center justify-center rounded-2xl border-2 border-dashed py-24 text-center">
			<Clock class="text-muted-foreground mb-4 h-12 w-12 opacity-50" />
			<h2 class="mb-2 text-xl font-semibold">Your list is empty</h2>
			<p class="text-muted-foreground max-w-sm">
				Videos you save for later will show up here.
			</p>
		</div>
	{:else}
		<div class="flex flex-col gap-4 max-w-4xl">
			{#each videos as item}
				{@const video = item.video || item}
				<a href={`/watch/${video.fileId || video.id}`} class="group relative flex flex-col sm:flex-row gap-4 hover:bg-muted/50 p-2 rounded-xl transition-colors">
					<!-- Remove Button -->
					<button 
						onclick={(e) => removeVideo(e, video.id)}
						class="absolute top-4 right-4 p-2 bg-background/80 hover:bg-red-500 hover:text-white rounded-full opacity-0 group-hover:opacity-100 transition-all z-10 border shadow-sm"
						title="Remove from Watch Later"
					>
						<X class="w-4 h-4" />
					</button>

					<!-- Thumbnail -->
					<div class="w-full sm:w-64 shrink-0">
						<VideoThumbnail 
							{video} 
							class="aspect-video rounded-xl"
							imgClass="transition-transform duration-300 group-hover:scale-105"
						>
							<div class="absolute right-2 bottom-2 rounded bg-black/80 px-2 py-1 text-xs font-medium text-white backdrop-blur-sm">
								{formatDuration(video.duration)}
							</div>
						</VideoThumbnail>
					</div>
					<!-- Metadata -->
					<div class="flex flex-col flex-1 py-1 pr-10">
						<h3 class="line-clamp-2 text-lg leading-tight font-semibold transition-colors group-hover:text-primary mb-1">
							{video.title}
						</h3>
						<div class="text-muted-foreground space-y-1 text-sm">
							<p class="transition-colors hover:text-primary">{video.uploader_username || 'Unknown User'}</p>
							<div class="flex items-center gap-1">
								<span>{video.views || 0} views</span>
							</div>
						</div>
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
