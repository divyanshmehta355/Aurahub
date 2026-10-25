<script lang="ts">
	import { page } from '$app/state';
	import { fetchApi } from '#lib/api';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import { Search as SearchIcon } from 'lucide-svelte';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';
	import { contentModeState } from '#lib/consent.svelte';

	const queryClient = useQueryClient();

	let queryStr = $derived(page.url.searchParams.get('q') || '');

	const searchQuery = createQuery(
		() => ({
			queryKey: ['search', queryStr, contentModeState.isAdult],
			queryFn: async () => {
				if (!queryStr) return { videos: [], total: 0 };
				const res = await fetchApi(`/videos/search?q=${encodeURIComponent(queryStr)}`);
				return {
					videos: res.videos || res.data || [],
					total: res.total || (res.videos || []).length
				};
			}
		}),
		() => queryClient
	);

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

<div class="space-y-6 px-4 md:space-y-8 md:px-0 max-w-5xl">
	<div class="border-b border-border/50 pb-4">
		<h1 class="text-xl font-bold tracking-tight text-foreground md:text-2xl">
			{#if queryStr}
				Search results for <span class="text-primary font-semibold">"{queryStr}"</span>
			{:else}
				Explore Aurahub
			{/if}
		</h1>
		{#if queryStr && searchQuery.data}
			<p class="text-xs text-muted-silver mt-1">Found {searchQuery.data.videos?.length || 0} matching releases</p>
		{/if}
	</div>

	{#if searchQuery.isPending && queryStr}
		<div class="flex flex-col gap-4">
			{#each Array(5) as _}
				<div class="flex animate-pulse flex-col gap-4 sm:flex-row p-3 rounded-2xl bg-card/40 border border-border/40">
					<Skeleton class="aspect-video w-full shrink-0 rounded-2xl sm:w-72" />
					<div class="flex-1 space-y-3 py-2">
						<Skeleton class="h-5 w-3/4 rounded-lg" />
						<Skeleton class="h-3 w-1/4 rounded-lg" />
						<div class="mt-4 flex items-center gap-3">
							<Skeleton class="h-7 w-7 rounded-full" />
							<Skeleton class="h-3 w-1/3 rounded-lg" />
						</div>
					</div>
				</div>
			{/each}
		</div>
	{:else if searchQuery.isError}
		<div class="flex flex-col items-center justify-center py-16 text-center">
			<div class="bg-destructive/10 border border-destructive/20 text-destructive rounded-2xl p-6 max-w-md">
				<p class="font-medium text-sm">{searchQuery.error.message || 'Failed to search videos.'}</p>
				<Button variant="outline" class="mt-4 rounded-full" onclick={() => searchQuery.refetch()}
					>Try again</Button
				>
			</div>
		</div>
	{:else if !queryStr || searchQuery.data?.videos.length === 0}
		<div
			class="border-border/60 bg-card/40 mx-4 flex flex-col items-center justify-center rounded-3xl border-2 border-dashed py-20 text-center md:mx-0"
		>
			<div class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-secondary border border-border/80 text-primary">
				<SearchIcon class="h-7 w-7 opacity-75" />
			</div>
			<h2 class="mb-2 text-lg font-bold text-foreground md:text-xl">No matching releases found</h2>
			<p class="text-muted-foreground max-w-sm px-4 text-xs sm:text-sm">
				{#if queryStr}
					Try adjusting your keywords, searching for a creator name, or checking for typos.
				{:else}
					Enter a title, topic, or creator name above to discover cinema.
				{/if}
			</p>
		</div>
	{:else if searchQuery.data}
		<div class="flex flex-col gap-4">
			{#each searchQuery.data.videos.filter((v: any) => !v.isShort) as video}
				<a
					href={`/watch/${video.fileId || video.id}`}
					class="group hover:bg-card flex flex-col gap-4 rounded-2xl p-3 border border-transparent hover:border-border/70 hover:shadow-sm transition-all sm:flex-row min-w-0"
				>
					<!-- Thumbnail -->
					<div class="w-full shrink-0 sm:w-72 overflow-hidden rounded-xl border border-border/60 bg-card">
						<VideoThumbnail
							{video}
							class="aspect-video w-full rounded-xl"
							imgClass="transition-transform duration-500 ease-out group-hover:scale-105"
						>
							<WatchLaterButton videoId={video.id} />
						</VideoThumbnail>
					</div>

					<!-- Metadata -->
					<div class="flex flex-1 flex-col py-0.5 min-w-0 overflow-hidden justify-center">
						<h3
							class="line-clamp-2 text-base leading-snug font-bold text-foreground transition-colors group-hover:text-primary mb-1 break-words [overflow-wrap:anywhere]"
							title={video.title}
						>
							{video.title}
						</h3>
						<div class="text-muted-silver mb-2.5 text-xs flex items-center gap-1.5">
							<span>{video.views || 0} views</span>
							<span>•</span>
							<span>{formatTimeAgo(video.created_at || video.createdAt)}</span>
						</div>

						<div class="mb-2 flex items-center gap-2.5">
							<img
								src={video.uploader_avatar ||
									video.uploader?.avatar ||
									`https://api.dicebear.com/7.x/identicon/svg?seed=${video.uploader_username || video.uploader?.username}`}
								alt={video.uploader_username || video.uploader?.username}
								class="h-6 w-6 rounded-full border border-border/80 object-cover"
							/>
							<p class="text-xs font-semibold text-foreground transition-colors hover:text-primary">
								@{video.uploader_username || video.uploader?.username}
							</p>
						</div>

						{#if video.description}
							<p class="text-muted-foreground line-clamp-2 text-xs leading-relaxed">
								{video.description}
							</p>
						{/if}
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
