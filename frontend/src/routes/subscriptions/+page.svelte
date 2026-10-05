<script lang="ts">
	import { page } from '$app/state';
	import { fetchApi } from '#lib/api';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import { PlaySquare } from 'lucide-svelte';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';

	const queryClient = useQueryClient();

	let currentPage = $derived(Number(page.url.searchParams.get('page')) || 1);
	const limit = 12;

	const subscriptionsQuery = createQuery(() => ({
		queryKey: ['subscriptions', currentPage],
		queryFn: async () => {
			const res = await fetchApi(`/videos/feed/subscriptions?page=${currentPage}&limit=${limit}`);
			return res;
		}
	}), () => queryClient);

	function getVisiblePages(current: number, total: number) {
		const pages = [];
		const maxVisible = 5;
		let start = Math.max(1, current - Math.floor(maxVisible / 2));
		let end = Math.min(total, start + maxVisible - 1);

		if (end - start + 1 < maxVisible) {
			start = Math.max(1, end - maxVisible + 1);
		}

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
</script>

<svelte:head>
	<title>Subscriptions - Aurahub</title>
</svelte:head>

<div class="px-4 md:px-0 space-y-6">
	<div class="flex items-center gap-3 border-b pb-4 pt-2 md:pt-0">
		<PlaySquare class="h-6 w-6 text-primary" />
		<h1 class="text-xl md:text-2xl font-bold tracking-tight">Subscriptions</h1>
	</div>

	{#if subscriptionsQuery.isPending}
		<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each Array(8) as _}
				<div class="flex animate-pulse flex-col space-y-3">
					<Skeleton class="aspect-video w-full rounded-xl" />
					<div class="flex gap-3 px-1">
						<Skeleton class="h-10 w-10 shrink-0 rounded-full" />
						<div class="flex-1 space-y-2">
							<Skeleton class="h-4 w-[90%] rounded" />
							<Skeleton class="h-3 w-[60%] rounded" />
						</div>
					</div>
				</div>
			{/each}
		</div>
	{:else if subscriptionsQuery.isError}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="rounded-xl bg-destructive/10 p-4 text-destructive">
				<p>{subscriptionsQuery.error.message || 'Failed to load subscriptions.'}</p>
				<Button variant="outline" class="mt-4" onclick={() => subscriptionsQuery.refetch()}>Try again</Button>
			</div>
		</div>
	{:else if !subscriptionsQuery.data?.videos || subscriptionsQuery.data.videos.length === 0}
		<div
			class="border-muted-foreground/20 mx-4 md:mx-0 flex flex-col items-center justify-center rounded-2xl border-2 border-dashed py-16 md:py-24 text-center"
		>
			<PlaySquare class="text-muted-foreground mb-4 h-12 w-12 opacity-50" />
			<h2 class="mb-2 text-lg md:text-xl font-semibold">No new videos</h2>
			<p class="text-muted-foreground max-w-sm text-sm md:text-base px-4">
				The channels you subscribe to haven't uploaded any videos recently.
			</p>
			<Button variant="default" class="mt-6 rounded-full" href="/">Discover Channels</Button>
		</div>
	{:else}
		<div class="grid grid-cols-1 gap-4 sm:gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
			{#each subscriptionsQuery.data.videos as video}
				<a href={`/watch/${video.fileId || video.id}`} class="group flex flex-col space-y-2 sm:space-y-3">
					<!-- Thumbnail -->
					<VideoThumbnail
						{video}
						class="aspect-video w-full rounded-xl sm:rounded-2xl"
						imgClass="transition-transform duration-300 group-hover:scale-105"
					>
						<WatchLaterButton videoId={video.id} />
					</VideoThumbnail>
					<!-- Metadata -->
					<div class="flex gap-3 px-1">
						<div class="h-8 w-8 sm:h-10 sm:w-10 shrink-0 overflow-hidden rounded-full">
							<img
								src={video.uploader?.avatar ||
									`https://api.dicebear.com/7.x/identicon/svg?seed=${video.uploader?.username}`}
								alt={video.uploader?.username}
								class="h-full w-full object-cover transition-transform group-hover:scale-110"
							/>
						</div>
						<div class="flex flex-col">
							<h3
								class="line-clamp-2 text-sm sm:text-base leading-tight font-semibold transition-colors group-hover:text-primary"
								title={video.title}
							>
								{video.title}
							</h3>
							<p class="text-muted-foreground mt-1 text-xs sm:text-sm transition-colors hover:text-foreground">
								{video.uploader?.username}
							</p>
							<div class="text-muted-foreground mt-0.5 text-[11px] sm:text-xs">
								<span>{video.views || 0} views</span>
								<span class="mx-1">•</span>
								<span>{formatTimeAgo(video.createdAt || video.created_at || new Date().toISOString())}</span>
							</div>
						</div>
					</div>
				</a>
			{/each}
		</div>

		{#if subscriptionsQuery.data?.totalPages > 1}
			<div class="mt-8 md:mt-12 flex justify-center pb-8">
				<nav class="flex items-center gap-1 md:gap-2">
					<Button
						variant="outline"
						size="sm"
						disabled={currentPage === 1}
						href={`/subscriptions?page=${currentPage - 1}`}
					>
						Previous
					</Button>
					
					<div class="hidden sm:flex mx-2 items-center gap-1">
						{#each getVisiblePages(currentPage, subscriptionsQuery.data.totalPages) as p}
							<Button
								variant={currentPage === p ? 'default' : 'ghost'}
								size="sm"
								href={`/subscriptions?page=${p}`}
							>
								{p}
							</Button>
						{/each}
					</div>

					<Button
						variant="outline"
						size="sm"
						disabled={currentPage === subscriptionsQuery.data.totalPages}
						href={`/subscriptions?page=${currentPage + 1}`}
					>
						Next
					</Button>
				</nav>
			</div>
		{/if}
	{/if}
</div>
