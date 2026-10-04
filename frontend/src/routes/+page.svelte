<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { fetchApi } from '#lib/api';
	import { getFallbackThumbnailUrl } from '#lib/thumbnailSvg';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import { Clock, Play } from 'lucide-svelte';

	let videos: any[] = $state([]);
	let isLoading = $state(true);
	let error = $state('');
	
	let currentPage = $derived(Number(page.url.searchParams.get('page')) || 1);
	let totalPages = $state(1);
	const limit = 12;

	$effect(() => {
		if (currentPage) {
			loadVideos();
		}
	});

	async function loadVideos() {
		isLoading = true;
		error = '';

		try {
			const res = await fetchApi(`/videos?page=${currentPage}&limit=${limit}`);
			videos = res.videos || [];
			totalPages = res.totalPages || 1;
		} catch (err: any) {
			error = err.message || 'Failed to load videos';
		} finally {
			isLoading = false;
		}
	}

	function goToPage(p: number) {
		if (p < 1 || p > totalPages) return;
		const url = new URL(page.url);
		url.searchParams.set('page', p.toString());
		goto(url.toString());
	}

	function getVisiblePages(current: number, total: number) {
		let start = Math.max(1, current - 2);
		let end = Math.min(total, current + 2);
		
		if (current <= 3) {
			end = Math.min(total, 5);
		}
		if (current >= total - 2) {
			start = Math.max(1, total - 4);
		}

		const pages = [];
		for (let i = start; i <= end; i++) {
			pages.push(i);
		}
		return pages;
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
	<title>Aurahub - Feed</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold tracking-tight">Recommended for you</h1>
	</div>

	{#if isLoading}
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each Array(8) as _}
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
		<div
			class="border-muted-foreground/20 flex flex-col items-center justify-center rounded-2xl border-2 border-dashed py-24 text-center"
		>
			<Play class="text-muted-foreground mb-4 h-12 w-12 opacity-50" />
			<h2 class="mb-2 text-xl font-semibold">No videos yet</h2>
			<p class="text-muted-foreground max-w-sm">
				When creators upload videos, they will appear here in your feed.
			</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each videos as video}
				<a href={`/watch/${video.fileId}`} class="group flex flex-col space-y-3">
					<!-- Thumbnail -->
					<VideoThumbnail 
						{video} 
						class="aspect-video rounded-xl"
						imgClass="transition-transform duration-300 group-hover:scale-105"
					>
						<WatchLaterButton videoId={video.id} />
					</VideoThumbnail>
					<!-- Metadata -->
					<div class="flex gap-3">
						<img
							src={video.uploader_avatar ||
								`https://api.dicebear.com/7.x/identicon/svg?seed=${video.uploader_username}`}
							alt={video.uploader_username}
							class="bg-muted h-10 w-10 shrink-0 rounded-full object-cover"
						/>
						<div class="flex flex-col overflow-hidden">
							<h3
								class="line-clamp-2 text-sm leading-tight font-semibold transition-colors group-hover:text-primary"
							>
								{video.title}
							</h3>
							<div class="text-muted-foreground mt-1 space-y-0.5 text-xs">
								<p class="transition-colors hover:text-primary">{video.uploader_username}</p>
								<div class="flex items-center gap-1">
									<span>{video.views || 0} views</span>
									<span class="text-[10px]">•</span>
									<span>{formatTimeAgo(video.createdAt || video.created_at)}</span>
								</div>
							</div>
						</div>
					</div>
				</a>
			{/each}
		</div>

		{#if totalPages > 1}
			<div class="mt-12 flex justify-center pb-8">
				<nav class="flex items-center gap-1 sm:gap-2">
					<button 
						onclick={() => goToPage(1)} 
						disabled={currentPage === 1}
						class="hidden sm:inline-flex items-center justify-center h-10 w-10 rounded-lg hover:bg-muted transition-colors disabled:opacity-50 disabled:pointer-events-none text-muted-foreground hover:text-foreground"
						aria-label="First page"
					>
						«
					</button>
					<button 
						onclick={() => goToPage(currentPage - 1)} 
						disabled={currentPage === 1}
						class="inline-flex items-center justify-center h-10 px-3 sm:px-4 rounded-lg bg-secondary text-secondary-foreground hover:bg-secondary/80 font-medium transition-colors disabled:opacity-50 disabled:pointer-events-none"
					>
						Previous
					</button>

					<div class="flex items-center gap-1 mx-2">
						{#each getVisiblePages(currentPage, totalPages) as p}
							<button 
								onclick={() => goToPage(p)}
								class="inline-flex items-center justify-center h-10 w-10 rounded-lg font-medium transition-colors {currentPage === p ? 'bg-primary text-primary-foreground shadow-sm' : 'hover:bg-muted text-muted-foreground hover:text-foreground'}"
							>
								{p}
							</button>
						{/each}
					</div>

					<button 
						onclick={() => goToPage(currentPage + 1)} 
						disabled={currentPage === totalPages}
						class="inline-flex items-center justify-center h-10 px-3 sm:px-4 rounded-lg bg-secondary text-secondary-foreground hover:bg-secondary/80 font-medium transition-colors disabled:opacity-50 disabled:pointer-events-none"
					>
						Next
					</button>
					<button 
						onclick={() => goToPage(totalPages)} 
						disabled={currentPage === totalPages}
						class="hidden sm:inline-flex items-center justify-center h-10 w-10 rounded-lg hover:bg-muted transition-colors disabled:opacity-50 disabled:pointer-events-none text-muted-foreground hover:text-foreground"
						aria-label="Last page"
					>
						»
					</button>
				</nav>
			</div>
		{/if}
	{/if}
</div>
