<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { fetchApi } from '#lib/api';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import { Search as SearchIcon } from 'lucide-svelte';

	let query = $derived(page.url.searchParams.get('q') || '');
	let videos: any[] = $state([]);
	let isLoading = $state(true);
	let error = $state('');

	async function performSearch() {
		if (!query) {
			videos = [];
			isLoading = false;
			return;
		}

		isLoading = true;
		error = '';
		
		try {
			const res = await fetchApi(`/videos/search?q=${encodeURIComponent(query)}`);
			videos = res.videos || res.data || [];
		} catch (err: any) {
			error = err.message || 'Failed to search videos.';
		} finally {
			isLoading = false;
		}
	}

	// Trigger search on mount or when query changes
	$effect(() => {
		if (query !== undefined) {
			performSearch();
		}
	});

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
	<title>{query ? `${query} - Search` : 'Search'} - Aurahub</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between pb-4 border-b">
		<h1 class="text-2xl font-bold tracking-tight">
			{#if query}
				Results for "{query}"
			{:else}
				Search Videos
			{/if}
		</h1>
	</div>

	{#if isLoading}
		<div class="flex flex-col gap-4 max-w-4xl">
			{#each Array(5) as _}
				<div class="flex flex-col sm:flex-row gap-4 animate-pulse">
					<div class="w-full sm:w-80 aspect-video bg-muted rounded-xl shrink-0"></div>
					<div class="flex-1 space-y-3 py-2">
						<div class="h-6 bg-muted rounded w-3/4"></div>
						<div class="h-4 bg-muted rounded w-1/4"></div>
						<div class="flex items-center gap-3 mt-4">
							<div class="h-8 w-8 bg-muted rounded-full"></div>
							<div class="h-4 bg-muted rounded w-1/3"></div>
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
			<SearchIcon class="text-muted-foreground mb-4 h-12 w-12 opacity-50" />
			<h2 class="mb-2 text-xl font-semibold">No results found</h2>
			<p class="text-muted-foreground max-w-sm">
				Try different keywords or remove search filters.
			</p>
		</div>
	{:else}
		<div class="flex flex-col gap-4 max-w-4xl">
			{#each videos as video}
				<a href={`/watch/${video.fileId || video.id}`} class="group flex flex-col sm:flex-row gap-4 hover:bg-muted/50 p-2 rounded-xl transition-colors">
					<!-- Thumbnail -->
					<div class="w-full sm:w-80 shrink-0">
						<VideoThumbnail 
							{video} 
							class="aspect-video rounded-xl"
							imgClass="transition-transform duration-300 group-hover:scale-105"
						>
							<WatchLaterButton videoId={video.id} />
						</VideoThumbnail>
					</div>
					<!-- Metadata -->
					<div class="flex flex-col flex-1 py-1">
						<h3 class="text-lg leading-tight font-semibold transition-colors group-hover:text-primary mb-1 line-clamp-2">
							{video.title}
						</h3>
						<div class="text-muted-foreground text-xs sm:text-sm mb-3">
							<span>{video.views || 0} views</span>
							<span class="mx-1">•</span>
							<span>{formatTimeAgo(video.created_at || video.createdAt)}</span>
						</div>
						
						<div class="flex items-center gap-2 mb-3">
							<img
								src={video.uploader_avatar || video.uploader?.avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=${video.uploader_username || video.uploader?.username}`}
								alt={video.uploader_username || video.uploader?.username}
								class="bg-muted h-6 w-6 rounded-full object-cover"
							/>
							<p class="text-sm font-medium transition-colors hover:text-primary">
								{video.uploader_username || video.uploader?.username}
							</p>
						</div>

						<p class="text-xs sm:text-sm text-muted-foreground line-clamp-2">
							{video.description || "No description."}
						</p>
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
