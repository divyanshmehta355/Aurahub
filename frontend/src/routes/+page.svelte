<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { fetchApi } from '#lib/api';
	import { getFallbackThumbnailUrl } from '#lib/thumbnailSvg';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import { Clock, Play } from 'lucide-svelte';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';

	const queryClient = useQueryClient();

	let currentPage = $derived(Number(page.url.searchParams.get('page')) || 1);
	const limit = 12;

	let query = createQuery(() => ({
		queryKey: ['videos', currentPage, limit],
		queryFn: async () => {
			const res = await fetchApi(`/videos?page=${currentPage}&limit=${limit}`);
			return res;
		}
	}), () => queryClient);

	function goToPage(p: number) {
		const totalPages = query.data?.totalPages || 1;
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

	{#if query.isPending}
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each Array(8) as _}
				<div class="flex animate-pulse flex-col space-y-3">
					<Skeleton class="aspect-video w-full rounded-xl" />
					<div class="flex gap-3">
						<Skeleton class="h-10 w-10 shrink-0 rounded-full" />
						<div class="w-full space-y-2">
							<Skeleton class="h-4 w-3/4 rounded" />
							<Skeleton class="h-3 w-1/2 rounded" />
						</div>
					</div>
				</div>
			{/each}
		</div>
	{:else if query.isError}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="bg-destructive/10 text-destructive rounded-xl p-4">
				<p>{query.error.message}</p>
				<Button variant="outline" class="mt-4" onclick={() => query.refetch()}>Try again</Button>
			</div>
		</div>
	{:else if !query.data?.videos || query.data.videos.length === 0}
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
			{#each query.data.videos as video}
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

		{#if query.data?.totalPages > 1}
			<div class="mt-12 flex justify-center pb-8">
				<nav class="flex items-center gap-2">
					<Button
						variant="outline"
						size="icon"
						disabled={currentPage === 1}
						onclick={() => goToPage(1)}
						class="hidden sm:flex"
					>
						«
					</Button>
					<Button
						variant="secondary"
						disabled={currentPage === 1}
						onclick={() => goToPage(currentPage - 1)}
					>
						Previous
					</Button>

					<div class="mx-2 flex items-center gap-1">
						{#each getVisiblePages(currentPage, query.data.totalPages) as p}
							<Button
								variant={currentPage === p ? 'default' : 'ghost'}
								size="icon"
								onclick={() => goToPage(p)}
							>
								{p}
							</Button>
						{/each}
					</div>

					<Button
						variant="secondary"
						disabled={currentPage === query.data.totalPages}
						onclick={() => goToPage(currentPage + 1)}
					>
						Next
					</Button>
					<Button
						variant="outline"
						size="icon"
						disabled={currentPage === query.data.totalPages}
						onclick={() => goToPage(query.data.totalPages)}
						class="hidden sm:flex"
					>
						»
					</Button>
				</nav>
			</div>
		{/if}
	{/if}
</div>
