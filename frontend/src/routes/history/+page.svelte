<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchApi } from '#lib/api';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import { Clock, Trash2 } from 'lucide-svelte';

	let videos: any[] = $state([]);
	let isLoading = $state(true);
	let error = $state('');

	async function loadHistory() {
		isLoading = true;
		error = '';
		try {
			const res = await fetchApi('/user/history');
			// Backend may return history objects with nested video or just flat videos array
			videos = res.history || res.videos || [];
		} catch (err: any) {
			error = err.message || 'Failed to load watch history.';
		} finally {
			isLoading = false;
		}
	}

	onMount(() => {
		loadHistory();
	});

	async function clearHistory() {
		if (!confirm('Are you sure you want to clear all your watch history?')) return;
		try {
			await fetchApi('/user/history', { method: 'DELETE' });
			videos = [];
		} catch (err: any) {
			alert('Failed to clear history');
		}
	}

	function formatTimeAgo(dateString: string) {
		const date = new Date(dateString);
		const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);

		let interval = seconds / 31536000;
		if (interval > 1) return Math.floor(interval) + ' years ago';
		interval = seconds / 2592000;
		if (interval > 1) return Math.floor(interval) + ' months ago';
		interval = seconds / 86400;
		if (interval > 1) return Math.floor(interval) + ' days ago';
		interval = seconds / 3600;
		if (interval > 1) return Math.floor(interval) + ' hours ago';
		interval = seconds / 60;
		if (interval > 1) return Math.floor(interval) + ' minutes ago';
		return Math.floor(seconds) + ' seconds ago';
	}

	function formatDuration(seconds: number) {
		if (!seconds) return '0:00';
		const m = Math.floor(seconds / 60);
		const s = Math.floor(seconds % 60);
		return `${m}:${s.toString().padStart(2, '0')}`;
	}
</script>

<svelte:head>
	<title>Watch History - Aurahub</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold tracking-tight">Watch History</h1>
		{#if videos.length > 0}
			<button 
				onclick={clearHistory}
				class="flex items-center gap-2 text-sm font-medium text-red-500 hover:text-red-600 transition-colors bg-red-50 dark:bg-red-950/30 px-4 py-2 rounded-lg"
			>
				<Trash2 class="w-4 h-4" />
				Clear all watch history
			</button>
		{/if}
	</div>

	{#if isLoading}
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each Array(4) as _}
				<div class="flex animate-pulse flex-col space-y-3">
					<div class="bg-muted aspect-video w-full rounded-xl"></div>
					<div class="flex gap-3">
						<div class="bg-muted h-10 w-10 shrink-0 rounded-full"></div>
						<div class="w-full space-y-2">
							<div class="bg-muted h-4 w-3/4 rounded"></div>
							<div class="bg-muted h-3 w-1/2 rounded"></div>
						</div>
					</div>
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
			<h2 class="mb-2 text-xl font-semibold">No watch history</h2>
			<p class="text-muted-foreground max-w-sm">
				Videos you watch will appear here.
			</p>
		</div>
	{:else}
		<div class="flex flex-col gap-4 max-w-4xl">
			{#each videos as item}
				<!-- Handle both flat video and nested video structures -->
				{@const video = item.video || item}
				<a href={`/watch/${video.fileId || video.id}`} class="group flex flex-col sm:flex-row gap-4 hover:bg-muted/50 p-2 rounded-xl transition-colors">
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
					<div class="flex flex-col flex-1 py-1">
						<h3 class="line-clamp-2 text-lg leading-tight font-semibold transition-colors group-hover:text-primary mb-1">
							{video.title}
						</h3>
						<div class="text-muted-foreground space-y-1 text-sm">
							<p class="transition-colors hover:text-primary">{video.uploader_username || 'Unknown User'}</p>
							<div class="flex items-center gap-1">
								<span>{video.views || 0} views</span>
								{#if item.watchedAt}
									<span class="text-[10px]">•</span>
									<span>Watched {formatTimeAgo(item.watchedAt)}</span>
								{/if}
							</div>
						</div>
						{#if video.description}
							<p class="mt-2 text-sm text-muted-foreground line-clamp-2 hidden sm:block">
								{video.description}
							</p>
						{/if}
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
