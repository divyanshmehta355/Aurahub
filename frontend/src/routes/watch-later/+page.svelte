<script lang="ts">
	import { fetchApi } from '#lib/api';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import WatchLaterButton from '#lib/components/WatchLaterButton.svelte';
	import { Clock, Trash2, X } from 'lucide-svelte';
	import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';

	const queryClient = useQueryClient();

	const watchLaterQuery = createQuery(() => ({
		queryKey: ['watch-later'],
		queryFn: async () => {
			const res = await fetchApi('/user/watch-later');
			return res.videos || res.items || [];
		}
	}), () => queryClient);

	const removeFromWatchLaterMutation = createMutation(() => ({
		mutationFn: async (videoId: string) => {
			await fetchApi(`/user/watch-later/${videoId}`, { method: 'DELETE' });
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['watch-later'] });
		}
	}), () => queryClient);

	const clearAllMutation = createMutation(() => ({
		mutationFn: async () => {
			await fetchApi('/user/watch-later', { method: 'DELETE' });
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: ['watch-later'] });
		}
	}), () => queryClient);

	function handleRemove(e: Event, videoId: string) {
		e.preventDefault();
		e.stopPropagation();
		removeFromWatchLaterMutation.mutate(videoId);
	}

	function handleClearAll() {
		if (confirm('Are you sure you want to remove all videos from Watch Later?')) {
			clearAllMutation.mutate();
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
</script>

<svelte:head>
	<title>Watch Later - Aurahub</title>
</svelte:head>

<div class="space-y-4 md:space-y-6 px-4 md:px-0">
	<div class="flex items-center justify-between border-b pb-4 pt-2 md:pt-0">
		<div class="flex items-center gap-3">
			<Clock class="h-6 w-6 text-primary" />
			<h1 class="text-xl md:text-2xl font-bold tracking-tight">Watch Later</h1>
		</div>
		
		{#if watchLaterQuery.data && watchLaterQuery.data.length > 0}
			<Button 
				variant="outline" 
				size="sm" 
				class="text-destructive hover:bg-destructive/10 hover:text-destructive"
				onclick={handleClearAll}
				disabled={clearAllMutation.isPending}
			>
				Clear All
			</Button>
		{/if}
	</div>

	{#if watchLaterQuery.isPending}
		<div class="flex max-w-4xl flex-col gap-4">
			{#each Array(5) as _}
				<div class="flex animate-pulse flex-col gap-4 sm:flex-row">
					<Skeleton class="aspect-video w-full shrink-0 rounded-xl sm:w-64" />
					<div class="flex-1 space-y-3 py-2">
						<Skeleton class="h-5 w-3/4 rounded" />
						<Skeleton class="h-4 w-1/4 rounded" />
					</div>
				</div>
			{/each}
		</div>
	{:else if watchLaterQuery.isError}
		<div class="flex flex-col items-center justify-center py-12 text-center">
			<div class="rounded-xl bg-destructive/10 p-4 text-destructive">
				<p>{watchLaterQuery.error.message || 'Failed to load Watch Later list.'}</p>
				<Button variant="outline" class="mt-4" onclick={() => watchLaterQuery.refetch()}>Try again</Button>
			</div>
		</div>
	{:else if !watchLaterQuery.data || watchLaterQuery.data.length === 0}
		<div
			class="border-muted-foreground/20 mx-4 md:mx-0 flex flex-col items-center justify-center rounded-2xl border-2 border-dashed py-16 md:py-24 text-center"
		>
			<Clock class="text-muted-foreground mb-4 h-12 w-12 opacity-50" />
			<h2 class="mb-2 text-lg md:text-xl font-semibold">Watch Later is empty</h2>
			<p class="text-muted-foreground max-w-sm text-sm md:text-base px-4">
				Save videos you want to watch later by clicking the clock icon on a video thumbnail.
			</p>
			<Button variant="default" class="mt-6 rounded-full" href="/">Discover Videos</Button>
		</div>
	{:else}
		<div class="flex max-w-4xl flex-col gap-6 md:gap-4">
			{#each watchLaterQuery.data as item}
				{@const video = item.video || item}
				<a
					href={`/watch/${video.fileId || video.id}`}
					class="group hover:bg-muted/50 flex flex-col gap-3 md:gap-4 rounded-xl transition-colors sm:flex-row sm:p-2"
				>
					<!-- Thumbnail -->
					<div class="w-full shrink-0 sm:w-64">
						<VideoThumbnail
							{video}
							class="aspect-video rounded-xl sm:rounded-lg"
							imgClass="transition-transform duration-300 group-hover:scale-105"
						/>
					</div>
					<!-- Metadata -->
					<div class="flex flex-1 flex-col py-1 px-1 sm:px-0 relative">
						<div class="pr-8">
							<h3 class="mb-1 line-clamp-2 text-base md:text-lg leading-tight font-semibold transition-colors group-hover:text-primary">
								{video.title}
							</h3>
							
							<div class="mb-1 flex items-center gap-2">
								<p class="text-[13px] md:text-sm font-medium transition-colors hover:text-primary">
									{video.uploader_username || video.uploader?.username}
								</p>
							</div>
							
							<div class="text-muted-foreground mb-3 text-[11px] md:text-xs">
								<span>{video.views || 0} views</span>
								<span class="mx-1">•</span>
								<span>{formatTimeAgo(video.createdAt || video.created_at || new Date().toISOString())}</span>
							</div>
							
							{#if video.description}
								<p class="text-muted-foreground line-clamp-1 md:line-clamp-2 text-[11px] md:text-xs">
									{video.description}
								</p>
							{/if}
						</div>
						
						<Button
							variant="ghost"
							size="icon"
							class="absolute top-0 right-0 h-8 w-8 text-muted-foreground opacity-100 md:opacity-0 transition-opacity group-hover:opacity-100 hover:text-destructive hover:bg-destructive/10"
							onclick={(e) => handleRemove(e, video.id || video.fileId)}
							title="Remove from Watch Later"
						>
							<X class="h-5 w-5" />
						</Button>
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
