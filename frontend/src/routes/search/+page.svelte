<script lang="ts">
	import { page } from '$app/state';
	import { fetchApi } from '#lib/api';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import { Search as SearchIcon } from 'lucide-svelte';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';

	const queryClient = useQueryClient();

	let queryStr = $derived(page.url.searchParams.get('q') || '');

	const searchQuery = createQuery(() => ({
		queryKey: ['search', queryStr],
		queryFn: async () => {
			if (!queryStr) return [];
			const res = await fetchApi(`/videos/search?q=${encodeURIComponent(queryStr)}`);
			return res.videos || res.data || [];
		}
	}), () => queryClient);

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
	<title>{queryStr ? `${queryStr} - Search` : 'Search'} - Aurahub</title>
</svelte:head>

<div class="space-y-4 md:space-y-6 px-4 md:px-0">
	<div class="flex items-center justify-between border-b pb-4 pt-2 md:pt-0">
		<h1 class="text-xl md:text-2xl font-bold tracking-tight">
			{#if queryStr}
				Results for "{queryStr}"
			{:else}
				Search Videos
			{/if}
		</h1>
	</div>

	{#if searchQuery.isPending && queryStr}
		<div class="flex max-w-4xl flex-col gap-4">
			{#each Array(5) as _}
				<div class="flex animate-pulse flex-col gap-4 sm:flex-row">
					<Skeleton class="aspect-video w-full shrink-0 rounded-xl sm:w-80" />
					<div class="flex-1 space-y-3 py-2">
						<Skeleton class="h-6 w-3/4 rounded" />
						<Skeleton class="h-4 w-1/4 rounded" />
						<div class="mt-4 flex items-center gap-3">
							<Skeleton class="h-8 w-8 rounded-full" />
							<Skeleton class="h-4 w-1/3 rounded" />
						</div>
					</div>
				</div>
			{/each}
		</div>
	{:else if searchQuery.isError}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="rounded-xl bg-destructive/10 p-4 text-destructive">
				<p>{searchQuery.error.message || 'Failed to search videos.'}</p>
				<Button variant="outline" class="mt-4" onclick={() => searchQuery.refetch()}>Try again</Button>
			</div>
		</div>
	{:else if !queryStr || searchQuery.data?.length === 0}
		<div
			class="border-muted-foreground/20 mx-4 md:mx-0 flex flex-col items-center justify-center rounded-2xl border-2 border-dashed py-16 md:py-24 text-center"
		>
			<SearchIcon class="text-muted-foreground mb-4 h-12 w-12 opacity-50" />
			<h2 class="mb-2 text-lg md:text-xl font-semibold">No results found</h2>
			<p class="text-muted-foreground max-w-sm text-sm md:text-base px-4">
				{#if queryStr}
					Try different keywords or remove search filters.
				{:else}
					Enter a search term above to find videos.
				{/if}
			</p>
		</div>
	{:else if searchQuery.data}
		<div class="flex max-w-4xl flex-col gap-6 md:gap-4">
			{#each searchQuery.data as video}
				<a
					href={`/watch/${video.fileId || video.id}`}
					class="group hover:bg-muted/50 flex flex-col gap-3 md:gap-4 rounded-xl transition-colors sm:flex-row sm:p-2"
				>
					<!-- Thumbnail -->
					<div class="w-full shrink-0 sm:w-80">
						<VideoThumbnail
							{video}
							class="aspect-video rounded-xl sm:rounded-lg"
							imgClass="transition-transform duration-300 group-hover:scale-105"
						>
							<WatchLaterButton videoId={video.id} />
						</VideoThumbnail>
					</div>
					<!-- Metadata -->
					<div class="flex flex-1 flex-col py-1 px-1 sm:px-0">
						<h3
							class="mb-1 line-clamp-2 text-base md:text-lg leading-tight font-semibold transition-colors group-hover:text-primary"
						>
							{video.title}
						</h3>
						<div class="text-muted-foreground mb-3 text-[11px] md:text-xs lg:text-sm">
							<span>{video.views || 0} views</span>
							<span class="mx-1">•</span>
							<span>{formatTimeAgo(video.created_at || video.createdAt)}</span>
						</div>

						<div class="mb-2 md:mb-3 flex items-center gap-2 md:gap-3">
							<img
								src={video.uploader_avatar ||
									video.uploader?.avatar ||
									`https://api.dicebear.com/7.x/identicon/svg?seed=${video.uploader_username || video.uploader?.username}`}
								alt={video.uploader_username || video.uploader?.username}
								class="bg-muted h-6 w-6 md:h-8 md:w-8 rounded-full object-cover"
							/>
							<p class="text-[13px] md:text-sm font-medium transition-colors hover:text-primary">
								{video.uploader_username || video.uploader?.username}
							</p>
						</div>
						
						{#if video.description}
							<p class="text-muted-foreground line-clamp-1 md:line-clamp-2 text-[11px] md:text-xs">
								{video.description}
							</p>
						{/if}
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
